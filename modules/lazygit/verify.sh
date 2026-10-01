#!/usr/bin/env bash
# lazygit/verify.sh - Verify lazygit runs (mise shims, PATH or ~/.local/bin).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_p="$(command -v lazygit 2>/dev/null || true)"
[[ -z "$_p" && -x "${_home}/.local/bin/lazygit" ]] && _p="${_home}/.local/bin/lazygit"
[[ -z "$_p" && -x "${_home}/.local/share/mise/shims/lazygit" ]] && _p="${_home}/.local/share/mise/shims/lazygit"
if [[ -z "$_p" ]] || ! "$_p" --version >/dev/null 2>&1; then
    log_error "lazygit not found or not runnable"
    exit 1
fi
log_success "lazygit verification passed"
