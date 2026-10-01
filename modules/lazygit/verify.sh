#!/usr/bin/env bash
# lazygit/verify.sh - Verify lazygit runs (PATH, ~/.local/bin or mise shims).
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
while IFS= read -r _p; do
    if [[ -n "$_p" ]] && "$_p" --version >/dev/null 2>&1; then
        log_success "lazygit verification passed (${_p})"
        exit 0
    fi
done < <(type -ap lazygit 2>/dev/null || true; printf '%s\n' "${_home}/.local/bin/lazygit" "${_home}/.local/share/mise/shims/lazygit")
log_error "lazygit not found or not runnable"
exit 1
