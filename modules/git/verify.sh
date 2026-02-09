# git/verify.sh - Verify git configuration

_git_errors=0

# Check git user.name is set
_git_name="$(git config --global user.name 2>/dev/null || true)"
if [[ -n "$_git_name" ]]; then
    log_success "git user.name is set: ${_git_name}"
else
    log_warn "git user.name is not set"
    _git_errors=$((_git_errors + 1))
fi

# Check git user.email is set
_git_email="$(git config --global user.email 2>/dev/null || true)"
if [[ -n "$_git_email" ]]; then
    log_success "git user.email is set: ${_git_email}"
else
    log_warn "git user.email is not set"
    _git_errors=$((_git_errors + 1))
fi

# Check gitignore_global is linked
_git_gitignore="${DOTFILES_HOME}/.gitignore_global"
if [[ -L "$_git_gitignore" ]]; then
    log_success "Global gitignore is symlinked: ${_git_gitignore}"
elif [[ -f "$_git_gitignore" ]]; then
    log_info "Global gitignore exists but is not a symlink: ${_git_gitignore}"
else
    log_warn "Global gitignore not found: ${_git_gitignore}"
    _git_errors=$((_git_errors + 1))
fi

# Check commit template is linked
_git_template="${DOTFILES_HOME}/.gitmessage"
if [[ -L "$_git_template" ]]; then
    log_success "Commit template is symlinked: ${_git_template}"
elif [[ -f "$_git_template" ]]; then
    log_info "Commit template exists but is not a symlink: ${_git_template}"
else
    log_warn "Commit template not found: ${_git_template}"
    _git_errors=$((_git_errors + 1))
fi

# Check init.defaultBranch is set
_git_default_branch="$(git config --global init.defaultBranch 2>/dev/null || true)"
if [[ "$_git_default_branch" == "main" ]]; then
    log_success "git init.defaultBranch is set to 'main'"
else
    log_warn "git init.defaultBranch is '${_git_default_branch}', expected 'main'"
    _git_errors=$((_git_errors + 1))
fi

if [[ $_git_errors -gt 0 ]]; then
    log_warn "Git verification completed with ${_git_errors} warning(s)"
else
    log_success "Git verification passed"
fi
