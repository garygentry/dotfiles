#!/usr/bin/env bash
# pi/install.sh - Install the pi coding agent CLI (generic, homelab-agnostic).
#
# gnet-lg ADR 0025 Decision 1 — "CLI authority, the generic module owns the pinned
# CLI":
#   * install the EXACT version from the overlay BOM:
#       npm i -g --ignore-scripts @earendil-works/pi-coding-agent@<pin>
#   * the pin comes from modules.pi.version in the content overlay (exposed here as
#     DOTFILES_SETTING_VERSION); this module carries NO operator/homelab specific.
#   * NO dual ownership: never run `pi update`. A pi installed at some other version
#     (a manual `pi update`, or a non-npm install) is corrected back to the pin.
#   * verify the resolved node/npm/pi, the Node minimum (derived from pi's own
#     engines.node, not hard-coded), and provenance (no WSL Windows-path binary).
#
# --ignore-scripts: pi is a coding agent with a broad dependency tree; skipping
# install scripts is the safe default for an unattended fleet install (the CLI
# needs none). Resources/settings are the overlay pi-config adapter's job, not this.

_pi_pkg="@earendil-works/pi-coding-agent"
_pi_pin="${DOTFILES_SETTING_VERSION:-}"          # from modules.pi.version (overlay BOM)
_pi_home="${DOTFILES_HOME:-$HOME}"

# Resolve pi, ignoring a Windows binary shadowing PATH under WSL (/mnt/...): a
# Windows pi cannot be managed by this Linux/macOS module and must not count as
# "installed" (same provenance guard as the nodejs module).
_pi_resolved() {
    local p
    p="$(command -v pi 2>/dev/null || true)"
    [[ -n "$p" && "$p" == /mnt/* ]] && p=""
    [[ -z "$p" && -x "${_pi_home}/.local/bin/pi" ]] && p="${_pi_home}/.local/bin/pi"
    printf '%s' "$p"
}

_pi_bin="$(_pi_resolved)"
_pi_cur=""
[[ -n "$_pi_bin" ]] && _pi_cur="$("$_pi_bin" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"

# Idempotent fast path: already at the pin (or present with no pin requested) ->
# nothing to do, and NO network/npm call (offline-safe, fast).
if [[ -n "$_pi_cur" ]]; then
    if [[ -z "$_pi_pin" ]]; then
        log_info "pi already installed (${_pi_cur}); no pin set (modules.pi.version) — leaving as-is"
        return 0 2>/dev/null || exit 0
    fi
    if [[ "$_pi_cur" == "$_pi_pin" ]]; then
        log_info "pi already at the pinned version (${_pi_pin})"
        return 0 2>/dev/null || exit 0
    fi
    log_info "pi is ${_pi_cur}, pin is ${_pi_pin} — reinstalling to the pin (no 'pi update')"
fi

# Target spec: pinned when the BOM supplies one, else npm 'latest' with a warning
# (a bare `dotfiles` install with no overlay is still usable, just unpinned).
if [[ -n "$_pi_pin" ]]; then
    _pi_spec="${_pi_pkg}@${_pi_pin}"
else
    log_warn "pi: no version pin (modules.pi.version) — installing '${_pi_pkg}@latest'; set a pin in your overlay BOM for a reproducible fleet"
    _pi_spec="${_pi_pkg}@latest"
fi

if is_dry_run; then
    log_info "[dry-run] Would install ${_pi_spec} (npm i -g --ignore-scripts)"
    return 0 2>/dev/null || exit 0
fi

# npm must be runnable (the nodejs dependency provides it). Guard explicitly so a
# missing/broken npm gives a clear message instead of an opaque failure.
if ! command -v npm >/dev/null 2>&1; then
    log_error "pi: npm not found — the 'nodejs' module must run first (declared as a dependency)"
    return 1 2>/dev/null || exit 1
fi

log_info "Installing ${_pi_spec} (sudo-free npm global)..."
if ! npm install -g --ignore-scripts "$_pi_spec"; then
    log_error "pi: 'npm install -g ${_pi_spec}' failed"
    return 1 2>/dev/null || exit 1
fi

# --- Post-install verification (ADR 0025 Decision 1) ---------------------------
_pi_bin="$(_pi_resolved)"
if [[ -z "$_pi_bin" ]]; then
    log_error "pi: installed via npm but 'pi' is not on PATH — check the npm global prefix / bin dir"
    return 1 2>/dev/null || exit 1
fi

# Provenance: a managed pi must be a real Linux/macOS binary, never a /mnt WSL path.
if [[ "$_pi_bin" == /mnt/* ]]; then
    log_error "pi: resolved to a Windows path (${_pi_bin}) under WSL — not a manageable install"
    return 1 2>/dev/null || exit 1
fi

# It must actually run (a present-but-broken binary must fail here, not report a
# hollow success).
_pi_cur="$("$_pi_bin" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
if [[ -z "$_pi_cur" ]]; then
    log_error "pi: installed but not runnable (${_pi_bin})"
    return 1 2>/dev/null || exit 1
fi
if [[ -n "$_pi_pin" && "$_pi_cur" != "$_pi_pin" ]]; then
    log_error "pi: installed ${_pi_cur} but the pin is ${_pi_pin}"
    return 1 2>/dev/null || exit 1
fi

# Node minimum, derived from pi's OWN engines.node (no hard-coded version fact —
# ADR 0025: version facts live in the BOM/package, not in this module). Non-fatal:
# the authoritative proof is that `pi --version` ran above; this is a clear
# diagnostic when a too-old node slips through.
_pi_root="$(npm root -g 2>/dev/null || true)"
_pi_pkgjson="${_pi_root:+${_pi_root}/${_pi_pkg}/package.json}"
if [[ -n "$_pi_pkgjson" && -f "$_pi_pkgjson" ]]; then
    _node_min="$(grep -oE '"node"[[:space:]]*:[[:space:]]*"[^"]*"' "$_pi_pkgjson" 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
    _node_cur="$(node --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
    if [[ -n "$_node_min" && -n "$_node_cur" ]]; then
        # numeric major.minor.patch compare: is _node_cur >= _node_min?
        _lowest="$(printf '%s\n%s\n' "$_node_min" "$_node_cur" | sort -t. -k1,1n -k2,2n -k3,3n | head -1)"
        if [[ "$_lowest" != "$_node_min" ]]; then
            log_warn "pi: node ${_node_cur} is below pi's engines.node minimum ${_node_min} — bump DOTFILES_NODE_MAJOR"
        fi
    fi
fi

log_success "pi installed: ${_pi_cur} (${_pi_bin})"
