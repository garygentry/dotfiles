#!/usr/bin/env bash
# nodejs/verify.sh - Verify Node.js installation
#
# gnet-lg ADR 0025 Decision 6 additions:
#   * the global npm prefix is the stable ~/.local (node-switch-safe);
#   * node meets the required floor (modules.nodejs.min_version), if set;
#   * node/npm resolve in BOTH a login and a non-login shell (a global install
#     that only exists on an interactive PATH is the NVM-prefix trap).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin_dir="${_home}/.local/bin"
_min="${DOTFILES_SETTING_MIN_VERSION:-}"
_node="$(command -v node 2>/dev/null || { [[ -x "${_bin_dir}/node" ]] && echo "${_bin_dir}/node"; } || true)"
_npm="$(command -v npm 2>/dev/null || { [[ -x "${_bin_dir}/npm" ]] && echo "${_bin_dir}/npm"; } || true)"

if [[ -z "$_node" ]]; then
    log_error "node not found"
    exit 1
fi
if [[ -z "$_npm" ]]; then
    log_error "npm not found"
    exit 1
fi

# Actually execute them — a present-but-non-runnable binary (e.g. a glibc build
# on musl) must fail here, not report a hollow success.
_nv="$("$_node" --version 2>/dev/null || true)"
if [[ -z "$_nv" ]]; then
    log_error "node is present but not runnable ($_node)"
    exit 1
fi
_npv="$("$_npm" --version 2>/dev/null || true)"
if [[ -z "$_npv" ]]; then
    log_error "npm is present but not runnable ($_npm)"
    exit 1
fi

# Floor (ADR 0025 D6): node must meet the required minimum, if one is set.
if [[ -n "$_min" ]]; then
    _nvbare="$(printf '%s' "$_nv" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
    _lowest="$(printf '%s\n%s\n' "$_min" "$_nvbare" | sort -t. -k1,1n -k2,2n -k3,3n | head -1)"
    if [[ -z "$_nvbare" || "$_lowest" != "$_min" ]]; then
        log_error "node ${_nv} is below the required floor ${_min} (set modules.nodejs.min_version / bump DOTFILES_NODE_MAJOR)"
        exit 1
    fi
fi

# Stable global npm prefix (ADR 0025 D6): the whole node-switch-safety guarantee.
# A prefix inside a versioned node dir means `npm i -g` bins vanish on a switch.
_prefix="$("$_npm" config get prefix 2>/dev/null || true)"
if [[ "$_prefix" != "${_home}/.local" ]]; then
    log_error "npm global prefix is '${_prefix:-unset}', expected '${_home}/.local' (global installs must survive a Node switch — see ~/.npmrc)"
    exit 1
fi

# Resolution in a login AND a non-login shell. A global tool that only exists on
# the interactive PATH breaks non-interactive callers (Pi spawning subprocesses,
# cron, module runners). Verify node/npm resolve to a runnable binary in both.
_shell_resolves() {   # $1 = extra shell flag(s)
    # shellcheck disable=SC2086
    bash $1 -c 'command -v node >/dev/null 2>&1 && command -v npm >/dev/null 2>&1 && node --version >/dev/null 2>&1'
}
if ! _shell_resolves "-l"; then
    log_error "node/npm do not resolve in a login shell (bash -lc) — check the interactive PATH"
    exit 1
fi
if ! _shell_resolves ""; then
    log_warn "node/npm do not resolve in a non-login shell (bash -c) — non-interactive callers may fail; ensure ${_bin_dir} is on PATH for non-login shells"
fi

log_success "Node.js verification passed: ${_nv} (npm ${_npv}, prefix ${_prefix}${_min:+, floor >=${_min}})"
