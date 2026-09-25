#!/usr/bin/env bash
# zsh/verify.sh - Verify Zsh installation and configuration

_zsh_errors=0
_zsh_framework="${DOTFILES_PROMPT_ZSH_FRAMEWORK:-zinit}"

# Check zsh is installed
if command -v zsh &>/dev/null; then
    _zsh_version="$(zsh --version 2>/dev/null | head -n1)"
    log_success "Zsh is installed: ${_zsh_version}"
else
    log_error "Zsh is not installed"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check .zshrc exists (symlink for zinit, regular file for template)
_zsh_rc="${DOTFILES_HOME}/.zshrc"
if [[ -L "$_zsh_rc" ]]; then
    log_success ".zshrc is symlinked: ${_zsh_rc}"
elif [[ -f "$_zsh_rc" ]]; then
    log_success ".zshrc exists: ${_zsh_rc}"
else
    log_warn ".zshrc not found: ${_zsh_rc}"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check plugin framework
if [[ "$_zsh_framework" == "ohmyzsh" ]]; then
    _zsh_omz_dir="${DOTFILES_HOME}/.oh-my-zsh"
    if [[ -d "$_zsh_omz_dir" ]]; then
        log_success "Oh My Zsh is installed at ${_zsh_omz_dir}"
    else
        log_warn "Oh My Zsh is not installed at ${_zsh_omz_dir}"
        _zsh_errors=$((_zsh_errors + 1))
    fi
else
    _zsh_zinit_home="${DOTFILES_HOME}/.local/share/zinit/zinit.git"
    if [[ -d "$_zsh_zinit_home" ]]; then
        log_success "Zinit is installed at ${_zsh_zinit_home}"
    else
        log_warn "Zinit is not installed at ${_zsh_zinit_home}"
        _zsh_errors=$((_zsh_errors + 1))
    fi
fi

# Check aliases file is linked
_zsh_aliases="${DOTFILES_HOME}/.config/zsh/aliases.zsh"
if [[ -L "$_zsh_aliases" ]]; then
    log_success "aliases.zsh is symlinked"
elif [[ -f "$_zsh_aliases" ]]; then
    log_info "aliases.zsh exists but is not a symlink"
else
    log_warn "aliases.zsh not found: ${_zsh_aliases}"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check functions file is linked
_zsh_functions="${DOTFILES_HOME}/.config/zsh/functions.zsh"
if [[ -L "$_zsh_functions" ]]; then
    log_success "functions.zsh is symlinked"
elif [[ -f "$_zsh_functions" ]]; then
    log_info "functions.zsh exists but is not a symlink"
else
    log_warn "functions.zsh not found: ${_zsh_functions}"
    _zsh_errors=$((_zsh_errors + 1))
fi

# Check the managed user-PATH block in ~/.zshenv (read by non-interactive zsh too)
if grep -qxF "# >>> dotfiles: path >>>" "${DOTFILES_HOME}/.zshenv" 2>/dev/null; then
    log_success "${DOTFILES_HOME}/.zshenv carries the managed PATH block"
else
    log_warn "${DOTFILES_HOME}/.zshenv is missing the managed PATH block (non-interactive zsh may miss ~/.local/bin)"
    _zsh_errors=$((_zsh_errors + 1))
fi

if [[ $_zsh_errors -gt 0 ]]; then
    log_warn "Zsh verification completed with ${_zsh_errors} warning(s)"
else
    log_success "Zsh verification passed"
fi

# Interactive startup time: median of 5 runs of `zsh -i </dev/null`, after one
# untimed warm-up (first start may clone plugins or rebuild the completion
# dump). Not `zsh -i -c exit`: the rc skips setup meant only for a real prompt
# under -c, and the budget should include it. (fzf's bindings need a terminal on
# stdin, so they are the one piece not measured; they cost a few ms.) Measured inside zsh via EPOCHREALTIME so it works on macOS too (BSD
# date has no %N). Always reported; it FAILS verification only when a budget is
# set (modules.zsh.startup_budget_ms), so generic users are never failed on it.
_zsh_budget="${DOTFILES_SETTING_STARTUP_BUDGET_MS:-}"
if [[ -n "$_zsh_budget" && ! "$_zsh_budget" =~ ^[0-9]+$ ]]; then
    log_warn "modules.zsh.startup_budget_ms must be a whole number of milliseconds (got '${_zsh_budget}'); ignoring it"
    _zsh_budget=""
fi
if command -v zsh &>/dev/null && [[ -f "${DOTFILES_HOME}/.zshrc" ]] && ! is_dry_run; then
    _zsh_median="$(HOME="${DOTFILES_HOME}" zsh -f -c '
        zmodload zsh/datetime
        zsh -i </dev/null >/dev/null 2>&1
        local -a t; local i s
        for i in 1 2 3 4 5; do
            s=$EPOCHREALTIME
            zsh -i </dev/null >/dev/null 2>&1
            t+=( $(( (EPOCHREALTIME - s) * 1000 )) )
        done
        t=( ${(on)t} )
        printf "%.0f" "${t[3]}"' 2>/dev/null || true)"
    if [[ -n "$_zsh_median" ]]; then
        if [[ -n "$_zsh_budget" ]] && (( _zsh_median > _zsh_budget )); then
            log_error "Zsh interactive startup ${_zsh_median}ms exceeds budget ${_zsh_budget}ms (profile: zmodload zsh/zprof at the top of ~/.zshrc, zprof at the end)"
            exit 1
        fi
        log_success "Zsh interactive startup: ${_zsh_median}ms (median of 5)${_zsh_budget:+, budget ${_zsh_budget}ms}"
    else
        log_warn "Could not measure zsh startup time"
    fi
fi
