# aliases.zsh - Common shell aliases
# Managed by dotfiles

# =============================================================================
# Directory Listing
# =============================================================================

alias ls='ls --color=auto'
alias l='ls -lah'
alias ll='ls -lh'
alias la='ls -lAh'

# =============================================================================
# Navigation
# =============================================================================

alias ..='cd ..'
alias ...='cd ../..'
alias ....='cd ../../..'
alias -- -='cd -'

# =============================================================================
# Safety
# =============================================================================

alias rm='rm -i'
alias cp='cp -i'
alias mv='mv -i'
alias mkdir='mkdir -p'

# =============================================================================
# Grep
# =============================================================================

alias grep='grep --color=auto'
alias fgrep='fgrep --color=auto'
alias egrep='egrep --color=auto'

# =============================================================================
# Misc
# =============================================================================
# Tool- and workflow-specific shortcuts (git, docker, personal utilities) are
# intentionally NOT shipped here — add them from your content overlay.

alias cls='clear'
alias reload='exec zsh'
alias h='history'
alias j='jobs -l'
alias path='echo -e ${PATH//:/\\n}'
alias now='date +"%Y-%m-%d %H:%M:%S"'
alias week='date +%V'
alias ports='ss -tulnp'
alias df='df -h'
alias du='du -h'
alias free='free -h'
