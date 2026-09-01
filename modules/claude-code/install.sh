#!/usr/bin/env bash
# claude-code/install.sh - Install Claude Code CLI (self-managed, non-interactive)
#
# Why not `curl -fsSL https://claude.ai/install.sh | bash`? That upstream
# installer's final step runs `claude install`, an interactive TUI that never
# completes under automation: given a controlling terminal it waits for
# keypresses; fully detached (setsid, closed stdin) it hangs before printing a
# single line, and occasionally exits 0 having installed nothing. Reproduced on
# a clean guest even with telemetry/autoupdate/nonessential-traffic disabled and
# with the documented Docker WORKDIR workaround. So it is unusable unattended.
#
# Instead we use Anthropic's documented direct-download path
# (https://code.claude.com/docs/en/setup#binary-integrity-and-code-signing):
# resolve the release, GPG-verify the signed manifest (checking the SIGNER is the
# pinned Anthropic release key, not merely that the key is present), sha256-check
# the native binary against the now-trusted manifest, and install into the layout
# the native installer uses — launcher at ~/.local/bin/claude symlinked into
# ~/.local/share/claude/versions/<version>. A hand-placed launcher is supported
# (v2.1.207+): auto-update and `claude update` leave it in place. No sudo, no Node.
#
# NOTE ON TRUST: gpg is best-effort. If gpg is absent or the signature is missing
# (releases <2.1.89), we fall back to the sha256 in manifest.json, whose only
# transport authentication is then TLS to downloads.claude.ai — weaker than the
# GPG chain. A *failed* GPG verification is always fatal.

_cc_home="${DOTFILES_HOME:-$HOME}"
_cc_launcher="${_cc_home}/.local/bin/claude"

# Fast path: already present (self-updates at runtime; nothing to do). Also match
# ~/.local/bin/claude directly, since it may not be on PATH yet on a fresh host.
if command -v claude >/dev/null 2>&1 || [[ -x "$_cc_launcher" ]]; then
    _cc_have="$(command -v claude 2>/dev/null || echo "$_cc_launcher")"
    log_info "Claude Code already installed ($("$_cc_have" --version 2>/dev/null | head -1))"
    return 0 2>/dev/null || exit 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Claude Code (self-managed native binary)"
    return 0 2>/dev/null || exit 0
fi

_cc_base="https://downloads.claude.ai/claude-code-releases"
_cc_key_url="https://downloads.claude.ai/keys/claude-code.asc"
# Pinned fingerprint of the Anthropic Claude Code release signing key
# (security@anthropic.com), verified out of band against the official docs.
_cc_key_fpr="31DDDE24DDFAB679F42D7BD2BAA929FF1A7ECACE"

# --- Platform detection (mirrors upstream install.sh) ---
case "$(uname -s)" in
    Linux)  _cc_os="linux" ;;
    Darwin) _cc_os="darwin" ;;
    *) log_error "Claude Code: unsupported OS $(uname -s)"; return 1 2>/dev/null || exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64)  _cc_arch="x64" ;;
    arm64|aarch64) _cc_arch="arm64" ;;
    *) log_error "Claude Code: unsupported architecture $(uname -m)"; return 1 2>/dev/null || exit 1 ;;
esac
_cc_platform="${_cc_os}-${_cc_arch}"
if [[ "$_cc_os" == "linux" ]] && { [[ -e /lib/libc.musl-x86_64.so.1 ]] || [[ -e /lib/libc.musl-aarch64.so.1 ]] || ldd /bin/ls 2>&1 | grep -q musl; }; then
    _cc_platform="${_cc_platform}-musl"
fi

# --- Resolve version (override with DOTFILES_CLAUDE_CODE_VERSION=latest|X.Y.Z). ---
_cc_channel="${DOTFILES_CLAUDE_CODE_VERSION:-stable}"
_cc_version="$(curl -fsSL "${_cc_base}/${_cc_channel}" 2>/dev/null || true)"
if [[ ! "$_cc_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    log_error "Claude Code: could not resolve a version from ${_cc_base}/${_cc_channel} (got: '${_cc_version:0:40}'). The download service may be unreachable or unavailable in this region."
    return 1 2>/dev/null || exit 1
fi

# --- Work area; cleaned up on any exit (best-effort; not on SIGKILL). ---
_cc_work="$(mktemp -d)"
trap 'rm -rf "$_cc_work"' EXIT

# Fetch the signed manifest (portable curl/wget via helpers).
if ! download_file "${_cc_base}/${_cc_version}/manifest.json" "${_cc_work}/manifest.json"; then
    log_error "Claude Code: failed to download manifest for ${_cc_version}"
    return 1 2>/dev/null || exit 1
fi

# --- Verify the manifest's GPG signature: confirm the SIGNER is the pinned key
#     (a keyring-membership check would accept any key present). Best-effort —
#     a missing sig or missing gpg degrades to sha256-only (see NOTE above); a
#     failed verification is fatal. ---
if download_file "${_cc_base}/${_cc_version}/manifest.json.sig" "${_cc_work}/manifest.json.sig" >/dev/null 2>&1 \
   && [[ -s "${_cc_work}/manifest.json.sig" ]]; then
    if command -v gpg >/dev/null 2>&1; then
        _cc_gnupg="${_cc_work}/gnupg"
        mkdir -p "$_cc_gnupg"; chmod 700 "$_cc_gnupg"
        if curl -fsSL "$_cc_key_url" 2>/dev/null | GNUPGHOME="$_cc_gnupg" gpg --batch --import >/dev/null 2>&1; then
            _cc_status="$(GNUPGHOME="$_cc_gnupg" gpg --batch --status-fd=1 --verify "${_cc_work}/manifest.json.sig" "${_cc_work}/manifest.json" 2>/dev/null || true)"
            if grep -qE "^\[GNUPG:\] VALIDSIG ${_cc_key_fpr} " <<<"$_cc_status"; then
                log_info "Claude Code: manifest signature verified (Anthropic release key ${_cc_key_fpr:0:8}…)"
            else
                log_error "Claude Code: manifest signature is not a valid signature from the pinned key — refusing to install"
                return 1 2>/dev/null || exit 1
            fi
        else
            log_warn "Claude Code: could not import the signing key — skipping signature check (sha256 vs the TLS-fetched manifest still enforced)"
        fi
    else
        log_warn "Claude Code: gpg not available — skipping manifest signature check (sha256 vs the TLS-fetched manifest still enforced)"
    fi
else
    log_warn "Claude Code: no manifest signature for ${_cc_version} (published for >= 2.1.89) — sha256 vs the TLS-fetched manifest still enforced"
fi

# --- Checksum for this platform (sha256 of the raw binary). ---
if command -v jq >/dev/null 2>&1; then
    _cc_sum="$(jq -r ".platforms[\"${_cc_platform}\"].checksum // empty" "${_cc_work}/manifest.json" 2>/dev/null || true)"
else
    # jq-free fallback: keep the match inside the platform's own {...} object.
    _cc_sum="$(tr -d '\n\r\t' < "${_cc_work}/manifest.json" \
        | grep -oE "\"${_cc_platform}\"[[:space:]]*:[[:space:]]*\{[^{}]*\"checksum\"[[:space:]]*:[[:space:]]*\"[a-f0-9]{64}\"" \
        | grep -oE '[a-f0-9]{64}' | head -1 || true)"
fi
if [[ ! "$_cc_sum" =~ ^[a-f0-9]{64}$ ]]; then
    log_error "Claude Code: no checksum for platform '${_cc_platform}' in manifest ${_cc_version}"
    return 1 2>/dev/null || exit 1
fi

# --- Download the binary and verify it against the (now-trusted) checksum. ---
log_info "Installing Claude Code ${_cc_version} (${_cc_platform})..."
if ! download_file "${_cc_base}/${_cc_version}/${_cc_platform}/claude" "${_cc_work}/claude" "$_cc_sum"; then
    log_error "Claude Code: binary download/checksum verification failed"
    return 1 2>/dev/null || exit 1
fi

# --- Install into the native installer's layout. ---
_cc_versions="${_cc_home}/.local/share/claude/versions"
_cc_target="${_cc_versions}/${_cc_version}"
mkdir -p "$_cc_versions" "${_cc_home}/.local/bin"
chmod +x "${_cc_work}/claude"
mv -f "${_cc_work}/claude" "$_cc_target"
ln -sf "$_cc_target" "$_cc_launcher"

log_success "Claude Code installed: $("$_cc_launcher" --version 2>/dev/null | head -1)"
