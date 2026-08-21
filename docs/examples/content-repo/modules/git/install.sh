#!/usr/bin/env bash
# git/install.sh — OVERRIDE example.
#
# Because overrides are whole-module replacement (not a merge), this file
# REPLACES the engine's git install.sh entirely — the built-in one does not run.
# A real override should reproduce the behavior you still want and then add or
# change what you need. This minimal version just sets identity + a couple of
# defaults so the example stays readable.

_git_user_name="${DOTFILES_USER_NAME:-}"
_git_user_email="${DOTFILES_USER_EMAIL:-}"

if is_dry_run; then
    log_info "[dry-run] Would apply custom git configuration (content override)"
    return 0
fi

[[ -n "$_git_user_name" ]]  && git config --global user.name  "$_git_user_name"
[[ -n "$_git_user_email" ]] && git config --global user.email "$_git_user_email"

# modules.git.default_branch is still exposed as DOTFILES_SETTING_DEFAULT_BRANCH.
git config --global init.defaultBranch "${DOTFILES_SETTING_DEFAULT_BRANCH:-main}"
git config --global pull.rebase true

log_success "Applied custom git configuration (content override)"
