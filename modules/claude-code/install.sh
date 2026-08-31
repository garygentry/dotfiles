#!/usr/bin/env bash
# claude-code/install.sh - Install Claude Code CLI (self-managed, non-interactive)
#
# Why not `curl -fsSL https://claude.ai/install.sh | bash`? That upstream
# installer's final step runs `claude install`, an interactive TUI that never
# completes under automation: given a controlling terminal it waits for
# keypresses; fully detached (setsid, closed stdin) it hangs before printing a
# single line, and occasionally exits 0 having installed nothing. Reproduced on
# a clean guest even with telemetry/autoupdate/nonessential-traffic disabled and
# with the documented Docker WORKDIR workaround. So `curl install.sh | bash` is
# unusable unattended.
#
# Instead we use Anthropic's documented direct-download path
# (https://code.claude.com/docs/en/setup#binary-integrity-and-code-signing):
# resolve the release, GPG-verify the signed manifest, sha256-check the native
# binary against it, and install into the same on-disk layout the native
# installer uses — launcher at ~/.local/bin/claude symlinked into
# ~/.local/share/claude/versions/<version>. Per the docs, a hand-placed launcher
# is supported (v2.1.207+): auto-update and `claude update` leave it in place.
# This path needs no sudo and no Node.

_cc_home="${DOTFILES_HOME:-$HOME}"
_cc_launcher="${_cc_home}/.local/bin/claude"

# Fast path: already present. Claude Code self-updates at runtime, so an existing
# install needs no action here (and we deliberately do NOT call `claude update`,
# which could re-enter an interactive path).
if command -v claude >/dev/null 2>&1 || [[ -x "$_cc_launcher" ]]; then
    _cc_have="$(command -v claude 2>/dev/null || echo "$_cc_launcher")"
    log_info "Claude Code already installed ($("$_cc_have" --version 2>/dev/null | head -1))"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Claude Code (self-managed native binary)"
    return 0
fi

_cc_base="https://downloads.claude.ai/claude-code-releases"
_cc_key_url="https://downloads.claude.ai/keys/claude-code.asc"
# Pinned fingerprint of the Anthropic Claude Code release signing key
# (security@anthropic.com). Verified out of band against the official docs; the
# module refuses a key that does not match this.
_cc_key_fpr="31DDDE24DDFAB679F42D7BD2BAA929FF1A7ECACE"

# --- Platform detection (mirrors upstream install.sh) ---
case "$(uname -s)" in
    Linux)  _cc_os="linux" ;;
    Darwin) _cc_os="darwin" ;;
    *) log_error "Claude Code: unsupported OS $(uname -s)"; return 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64)  _cc_arch="x64" ;;
    arm64|aarch64) _cc_arch="arm64" ;;
    *) log_error "Claude Code: unsupported architecture $(uname -m)"; return 1 ;;
esac
_cc_platform="${_cc_os}-${_cc_arch}"
# musl libc gets its own build (Alpine and friends).
if [[ "$_cc_os" == "linux" ]] && { [[ -e /lib/libc.musl-x86_64.so.1 ]] || [[ -e /lib/libc.musl-aarch64.so.1 ]] || ldd /bin/ls 2>&1 | grep -q musl; }; then
    _cc_platform="${_cc_platform}-musl"
fi

# --- Resolve version. Default to the `stable` channel; override with
#     DOTFILES_CLAUDE_CODE_VERSION=latest (or a pinned X.Y.Z) for a fleet pin. ---
_cc_channel="${DOTFILES_CLAUDE_CODE_VERSION:-stable}"
_cc_version="$(curl -fsSL "${_cc_base}/${_cc_channel}" 2>/dev/null)"
if [[ ! "$_cc_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    log_error "Claude Code: could not resolve a version from ${_cc_base}/${_cc_channel} (got: '${_cc_version:0:40}'). The download service may be unreachable or unavailable in this region."
    return 1
fi

# --- Work area (manifest, signature, binary download); always cleaned up. ---
_cc_work="$(mktemp -d)"
_cc_cleanup() { [[ -n "${_cc_work:-}" ]] && rm -rf "$_cc_work"; }

# Fetch the signed manifest.
if ! curl -fsSL -o "${_cc_work}/manifest.json" "${_cc_base}/${_cc_version}/manifest.json" 2>/dev/null; then
    log_error "Claude Code: failed to download manifest for ${_cc_version}"
    _cc_cleanup; return 1
fi

# --- Verify the manifest's GPG signature (defense-in-depth over TLS).
#     Best-effort: signatures exist only for >= 2.1.89, and gpg may be absent on
#     a minimal host. A missing sig or missing gpg degrades to a warning (the
#     sha256 check below is always enforced); a *failed* verification is fatal. ---
if curl -fsSL -o "${_cc_work}/manifest.json.sig" "${_cc_base}/${_cc_version}/manifest.json.sig" 2>/dev/null \
   && [[ -s "${_cc_work}/manifest.json.sig" ]]; then
    if command -v gpg >/dev/null 2>&1; then
        _cc_gnupg="${_cc_work}/gnupg"
        mkdir -p "$_cc_gnupg"; chmod 700 "$_cc_gnupg"
        if curl -fsSL "$_cc_key_url" 2>/dev/null | GNUPGHOME="$_cc_gnupg" gpg --batch --import >/dev/null 2>&1 \
           && GNUPGHOME="$_cc_gnupg" gpg --batch --with-colons --fingerprint 2>/dev/null | grep -q "^fpr:::::::::${_cc_key_fpr}:"; then
            if GNUPGHOME="$_cc_gnupg" gpg --batch --verify "${_cc_work}/manifest.json.sig" "${_cc_work}/manifest.json" >/dev/null 2>&1; then
                log_info "Claude Code: manifest signature verified (Anthropic release key ${_cc_key_fpr:0:8}…)"
            else
                log_error "Claude Code: manifest GPG signature verification FAILED — refusing to install"
                _cc_cleanup; return 1
            fi
        else
            log_warn "Claude Code: could not import/confirm the pinned signing key — skipping signature check (sha256 still enforced)"
        fi
    else
        log_warn "Claude Code: gpg not available — skipping manifest signature check (sha256 still enforced)"
    fi
else
    log_warn "Claude Code: no manifest signature for ${_cc_version} (published for >= 2.1.89) — sha256 still enforced"
fi

# --- Checksum for this platform (sha256 of the raw binary). ---
if command -v jq >/dev/null 2>&1; then
    _cc_sum="$(jq -r ".platforms[\"${_cc_platform}\"].checksum // empty" "${_cc_work}/manifest.json" 2>/dev/null)"
else
    # jq-free fallback: keep the match inside the platform's own {...} object.
    _cc_sum="$(tr -d '\n\r\t' < "${_cc_work}/manifest.json" \
        | grep -oE "\"${_cc_platform}\"[[:space:]]*:[[:space:]]*\{[^{}]*\"checksum\"[[:space:]]*:[[:space:]]*\"[a-f0-9]{64}\"" \
        | grep -oE '[a-f0-9]{64}' | head -1)"
fi
if [[ ! "$_cc_sum" =~ ^[a-f0-9]{64}$ ]]; then
    log_error "Claude Code: no checksum for platform '${_cc_platform}' in manifest ${_cc_version}"
    _cc_cleanup; return 1
fi

# --- Download the binary and verify it against the (now-trusted) checksum. ---
log_info "Installing Claude Code ${_cc_version} (${_cc_platform})..."
if ! curl -fsSL -o "${_cc_work}/claude" "${_cc_base}/${_cc_version}/${_cc_platform}/claude" 2>/dev/null; then
    log_error "Claude Code: binary download failed"
    _cc_cleanup; return 1
fi
_cc_actual="$(sha256sum "${_cc_work}/claude" | cut -d' ' -f1)"
if [[ "$_cc_actual" != "$_cc_sum" ]]; then
    log_error "Claude Code: checksum mismatch (expected ${_cc_sum}, got ${_cc_actual})"
    _cc_cleanup; return 1
fi

# --- Install into the native installer's layout: versioned binary under
#     ~/.local/share/claude/versions/<version>, launcher symlink in ~/.local/bin. ---
_cc_versions="${_cc_home}/.local/share/claude/versions"
_cc_target="${_cc_versions}/${_cc_version}"
mkdir -p "$_cc_versions" "${_cc_home}/.local/bin"
chmod +x "${_cc_work}/claude"
mv -f "${_cc_work}/claude" "$_cc_target"
ln -sf "$_cc_target" "$_cc_launcher"
_cc_cleanup

log_success "Claude Code installed: $("$_cc_launcher" --version 2>/dev/null | head -1)"
