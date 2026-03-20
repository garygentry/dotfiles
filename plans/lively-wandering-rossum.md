# Fix Neovim Install Errors and LSPConfig Deprecation

## Context

Two issues when installing neovim via the dotfiles manager:

1. **Noisy pre-skip verify output**: When neovim was previously installed but the binary was removed, the pre-skip verify check shows `✗ Failed neovim` with full verify.sh error output before proceeding to reinstall successfully. These errors are confusing since the install ultimately succeeds.

2. **LSPConfig deprecation warning on first nvim launch**: The `nvim-lspconfig` plugin triggers a deprecation warning (`require('lspconfig')` framework is deprecated, use `vim.lsp.config`). Our init.lua already uses the native `vim.lsp.config()` API, but the plugin itself fires the warning when loaded by lazy.nvim.

## Fix 1: Silent Pre-Skip Verify Check

**File**: `internal/module/runner.go` (lines 239-254)

**Problem**: The pre-skip verify calls `runScript()` which shows a spinner and prints `✗ Failed {module}` + full script output on failure. This is correct for real install failures but wrong for the idempotence check where we only need the exit code.

**Fix**: Replace the `runScript()` call with an inline `exec.CommandContext` that discards all output. Same wrapper/env setup as `runScript`, but with `cmd.Stdout = io.Discard` and `cmd.Stderr = io.Discard`.

```go
if decision == ExecutionSkip {
    verifyScript := filepath.Join(mod.Dir, "verify.sh")
    if _, statErr := os.Stat(verifyScript); statErr == nil && !cfg.DryRun {
        helpersPath := filepath.Join(cfg.SysInfo.DotfilesDir, "lib", "helpers.sh")
        var wrapper strings.Builder
        wrapper.WriteString("set -euo pipefail\n")
        wrapper.WriteString(fmt.Sprintf("if [ -f %q ]; then source %q; fi\n", helpersPath, helpersPath))
        wrapper.WriteString(fmt.Sprintf("source %q\n", verifyScript))

        vCtx, vCancel := context.WithTimeout(context.Background(), 30*time.Second)
        cmd := exec.CommandContext(vCtx, "bash", "-c", wrapper.String())
        cmd.Env = os.Environ()
        envVars := buildEnvVars(cfg, mod, nil)
        for k, v := range envVars {
            cmd.Env = append(cmd.Env, k+"="+v)
        }
        cmd.Stdout = io.Discard
        cmd.Stderr = io.Discard
        if err := cmd.Run(); err != nil {
            decision = ExecutionInstallRetry
            reason = "verify failed: module not functional"
        }
        vCancel()
    }
    if decision == ExecutionSkip {
        cfg.UI.Info(fmt.Sprintf("✓ %s (skipped: %s)", mod.Name, reason))
        return RunResult{Module: mod, Success: true, Skipped: true, Duration: time.Since(start)}
    }
}
```

No new imports needed (`io`, `context`, `exec`, etc. are all already imported).

## Fix 2: Remove nvim-lspconfig and mason-lspconfig

**File**: `modules/neovim/init.lua`

**Problem**: The `nvim-lspconfig` plugin fires a deprecation warning when loaded. Since we already use native `vim.lsp.config()` and `vim.lsp.enable()`, we don't need it. `mason-lspconfig` bridges mason↔lspconfig, so it's also unnecessary.

**Changes**:

### A. Remove mason-lspconfig plugin spec (lines 277-295)
Delete the entire `williamboman/mason-lspconfig.nvim` spec block.

### B. Remove nvim-lspconfig plugin spec (lines 297-401)
Delete the entire `neovim/nvim-lspconfig` spec block. Preserve the config contents for reinsertion below.

### C. Add ensure_installed helper to mason.nvim config (lines 268-275)
Update mason's config function to auto-install LSP servers using `mason-registry` directly:

```lua
{
    "williamboman/mason.nvim",
    config = function()
      require("mason").setup({
        ui = { border = "rounded" },
      })

      -- Auto-install LSP servers and tools
      local ensure_installed = {
        "typescript-language-server",
        "lua-language-server",
        "gopls",
        "html-lsp",
        "css-lsp",
        "json-lsp",
        "yaml-language-server",
        "eslint-lsp",
      }
      local registry = require("mason-registry")
      registry.refresh(function()
        for _, name in ipairs(ensure_installed) do
          local ok, pkg = pcall(registry.get_package, name)
          if ok and not pkg:is_installed() then
            pkg:install()
          end
        end
      end)
    end,
  },
```

Note: mason uses different package names than lspconfig (e.g. `lua-language-server` not `lua_ls`).

### D. Move LSP configuration out of plugin spec
Move the keybindings, `vim.lsp.config()`, `vim.lsp.enable()`, and `vim.diagnostic.config()` to run after `lazy.setup()` completes (after the closing `})` of the setup call). This config uses only native nvim APIs and doesn't need any plugin:

```lua
-- After lazy.setup({...}) ends:

-- LSP keybindings via LspAttach autocmd
vim.api.nvim_create_autocmd("LspAttach", {
  callback = function(ev)
    local map = function(keys, func, desc)
      vim.keymap.set("n", keys, func, { buffer = ev.buf, desc = "LSP: " .. desc })
    end
    map("gd",         vim.lsp.buf.definition,       "Go to Definition")
    -- ... (all existing keymaps)
    -- ESLint fix-on-save autocmd
  end,
})

-- Completion capabilities
local capabilities = require("cmp_nvim_lsp").default_capabilities()
vim.lsp.config("*", { capabilities = capabilities })

-- Per-server configs (ts_ls, lua_ls, gopls)
-- ... (existing vim.lsp.config calls, unchanged)

vim.lsp.enable({ "ts_ls", "lua_ls", "eslint", "gopls", "html", "cssls", "jsonls", "yamlls" })

vim.diagnostic.config({ ... })
```

## Verification

1. **Issue 1**: Uninstall neovim (`sudo apt remove neovim`), run `dotfiles install` selecting neovim — should NOT show `✗ Failed neovim` or verify.sh error output. Should show `• Updating neovim (verify failed: module not functional)...` directly.
2. **Issue 2**: Launch `nvim` — should NOT show lspconfig deprecation warning. LSP servers should still auto-install via mason and attach correctly.
3. Run `make test` to verify no Go test regressions.
4. Run `make lint-shell` to verify shell scripts still pass.

## Critical Files
- `internal/module/runner.go` — pre-skip verify logic (lines 239-254)
- `modules/neovim/init.lua` — full neovim config (plugin specs + LSP setup)
