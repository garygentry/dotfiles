#!/usr/bin/env bash
# pi/verify.sh - Verify the pi coding agent CLI (gnet-lg ADR 0025 Decision 1).
#
# Structural only, no provider auth: pi resolves, is a real (non-WSL) binary, runs,
# and — when the overlay BOM pins a version (DOTFILES_SETTING_VERSION) — is AT the
# pin. A richer RPC-`sourceInfo` provenance check for managed resources belongs to
# the overlay pi-config adapter (ADR 0025 Decision 8), not this generic module.

_pi_home="${DOTFILES_HOME:-$HOME}"
_pi_pin="${DOTFILES_SETTING_VERSION:-}"

_pi_bin="$(command -v pi 2>/dev/null || true)"
[[ -n "$_pi_bin" && "$_pi_bin" == /mnt/* ]] && _pi_bin=""
[[ -z "$_pi_bin" && -x "${_pi_home}/.local/bin/pi" ]] && _pi_bin="${_pi_home}/.local/bin/pi"

if [[ -z "$_pi_bin" ]]; then
    log_error "pi not found"
    return 1 2>/dev/null || exit 1
fi

_pi_ver="$("$_pi_bin" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
if [[ -z "$_pi_ver" ]]; then
    log_error "pi is present but not runnable (${_pi_bin})"
    return 1 2>/dev/null || exit 1
fi

if [[ -n "$_pi_pin" && "$_pi_ver" != "$_pi_pin" ]]; then
    log_error "pi is ${_pi_ver} but the pin (modules.pi.version) is ${_pi_pin}"
    return 1 2>/dev/null || exit 1
fi

# Resolution in a login AND a non-login shell (ADR 0025 D6): pi is a global npm bin;
# a stable prefix (~/.local, the nodejs module's job) is what keeps it on PATH across
# a Node switch. A login-only pi breaks non-interactive callers. Login is required;
# a non-login miss is a warning (the shell PATH config, not pi, is then at fault).
if ! bash -lc 'command -v pi >/dev/null 2>&1'; then
    log_error "pi does not resolve in a login shell (bash -lc) — check the interactive PATH / npm global prefix"
    return 1 2>/dev/null || exit 1
fi
if ! bash -c 'command -v pi >/dev/null 2>&1'; then
    log_warn "pi does not resolve in a non-login shell (bash -c) — non-interactive callers may fail; ensure the npm global bin dir is on PATH for non-login shells"
fi

log_success "pi verification passed: ${_pi_ver}${_pi_pin:+ (pinned)}"
