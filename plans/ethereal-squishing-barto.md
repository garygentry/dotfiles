# Graceful 1Password Handling & Bootstrap UX Fixes

## Context

Running `curl -sfL .../bootstrap.sh | bash` on a fresh system with no 1Password account configured reveals several UX issues:

1. **Double interactive prompt** — `IsAuthenticated()` runs `op vault list` which triggers the op CLI's "Do you want to add an account manually?" prompt. The user says no, then `Authenticate()` runs the same command, triggering the identical prompt again.
2. **Non-interactive stdin** — `curl | bash` pipes curl's output to bash stdin, so the Go binary sees stdin as a pipe and forces unattended mode. But the user IS at an interactive terminal (proven by `op` successfully prompting via `/dev/tty`).
3. **Spinner/sudo collision** — The spinner goroutine writes `\r⠏ Installing starship...` to stdout every 80ms. When starship's installer calls `sudo`, the password prompt writes to `/dev/tty`, colliding on the same line: `⠏ Installing starship...[sudo] password for gary:`.
4. **No skip workflow** — When 1Password is configured but not authenticated, the user has no option to skip and continue. They must endure the double-prompt failure before the install proceeds.

## Changes

### 1. Fix bootstrap.sh stdin — redirect from `/dev/tty`
**File:** `bootstrap.sh` (line 143)

`curl | bash` makes stdin a pipe (at EOF after the script is read). The Go binary inherits this pipe as stdin, causing `detectInteractive()` to return false. Fix by redirecting stdin from `/dev/tty` when available:

```bash
if [ -t 0 ] || [ -e /dev/tty ]; then
    exec ./bin/dotfiles install "$@" < /dev/tty
else
    exec ./bin/dotfiles install "$@"
fi
```

This makes stdin a real TTY for the Go binary, so prompts work and the "Non-interactive stdin detected" message goes away. The fallback preserves behavior in truly headless environments (Docker, CI).

### 2. Make `IsAuthenticated()` truly non-interactive
**File:** `internal/secrets/onepassword.go`

The current `IsAuthenticated()` runs `op --account <acct> vault list` which triggers the op CLI's interactive account-setup prompt via `/dev/tty`. Replace with a two-step non-interactive check:

1. Run `op account list` — lists configured accounts without requiring auth, never prompts. If the output is empty or errors, no accounts exist → return false.
2. Run `op whoami --account <acct>` — checks if a valid session exists. Fails silently if no session (no interactive prompt). Returns quickly.

Both commands get `Stdin` set to nil (Go maps this to `/dev/null`) and short timeouts (5s). Neither triggers the "Do you want to add an account manually?" prompt.

### 3. Rewrite `Authenticate()` to use `op signin`
**File:** `internal/secrets/onepassword.go`

Replace `op vault list` with `op signin --account <acct>` which is the proper authentication command. Connect stdin/stdout/stderr to the terminal so the user can interact with the sign-in flow. This is cleaner semantics — `vault list` was a side-effect-based auth check, `signin` is the intended auth command.

### 4. Rework install.go Phase 2 — user-driven secrets flow
**File:** `cmd/dotfiles/install.go`

Replace the current Phase 2 with a structured flow:

```
provider configured?
├── no → skip silently (NoopProvider)
├── yes, CLI not installed → warn and continue
└── yes, CLI installed
    ├── already authenticated → success message
    └── not authenticated
        ├── unattended mode → info message, skip
        └── interactive → prompt user:
            "1Password is not authenticated. Set up now? [y/N]"
            ├── yes → run Authenticate(), handle success/failure
            └── no → info: "Skipping secrets. Modules that use secrets
                     will fall back to defaults. Run 'dotfiles install'
                     later to set up 1Password."
```

This eliminates the double-prompt entirely (we never call `IsAuthenticated()` followed by `Authenticate()` with the same interactive command) and gives the user a clear choice.

### 5. Stop spinner during script execution
**File:** `internal/module/runner.go`

The spinner goroutine runs every 80ms writing `\r<frame> msg` to stdout. When a subprocess writes to `/dev/tty` (e.g. sudo password prompt), the spinner and subprocess output collide on the same line.

Fix: Stop the spinner before each `runScript()` call and restart after. The `runScript()` function uses `cmd.CombinedOutput()` which blocks until the script finishes, so there's no visual gap — the user sees:

```
• Installing starship...        ← static info line before scripts
[sudo] password for gary: ***   ← sudo prompt on its own clean line
✓ Installed starship            ← result after scripts finish
```

In `runModule()`:
- Print `Info("Installing <name>...")` at the start (static, no spinner)
- Run OS script, install script without any active spinner
- Start spinner only for file deployment (Go-native, no subprocess /dev/tty writes)
- Stop spinner, run verify script without spinner
- Print final success/fail message

### 6. Update `get_secret.go` error for unauthenticated state
**File:** `cmd/dotfiles/get_secret.go`

The `IsAuthenticated()` method is also called here (line 42). With our new non-interactive `IsAuthenticated()`, this now works cleanly. No code change needed — just noting it benefits from change #2.

## Files Modified

| File | Change |
|------|--------|
| `bootstrap.sh` | Redirect stdin from `/dev/tty` on line 143 |
| `internal/secrets/onepassword.go` | Rewrite `IsAuthenticated()` to use `op account list` + `op whoami`; rewrite `Authenticate()` to use `op signin` |
| `cmd/dotfiles/install.go` | Rework Phase 2 with user choice flow |
| `internal/module/runner.go` | Stop spinner during script execution |

## Files NOT Modified

- **`internal/secrets/provider.go`** — Interface and factory unchanged
- **`internal/secrets/noop.go`** — Unchanged
- **`internal/ui/ui.go`** — Spinner start/stop methods already have `\033[K` line-clear; no new methods needed
- **`modules/ssh/install.sh`** — Already guards with `command -v op`
- **`config.yml`** — Keep `provider: 1password` as shipped default

## Verification

1. `go build ./...` — compiles
2. `go test ./...` — all tests pass
3. Update `internal/secrets/provider_test.go` if needed for new `IsAuthenticated` / `Authenticate` behavior
4. Manual test on VM: `curl -sfL .../bootstrap.sh | bash` with no 1Password account configured:
   - Should see interactive prompt asking to set up 1Password (not the op CLI's own prompt)
   - Saying "no" should show skip message and continue
   - No double-prompt
   - Spinner should not collide with sudo prompt
   - All modules should install with interactive prompts (not unattended defaults)
