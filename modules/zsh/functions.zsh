# functions.zsh - Utility shell functions
# Managed by dotfiles

# =============================================================================
# mkcd - Create a directory and cd into it
# =============================================================================
mkcd() {
    if [[ -z "$1" ]]; then
        echo "Usage: mkcd <directory>" >&2
        return 1
    fi
    mkdir -p "$1" && cd "$1"
}

# =============================================================================
# extract - Extract most common archive formats
# =============================================================================
extract() {
    if [[ -z "$1" ]]; then
        echo "Usage: extract <archive>" >&2
        return 1
    fi

    if [[ ! -f "$1" ]]; then
        echo "File not found: $1" >&2
        return 1
    fi

    case "$1" in
        *.tar.bz2)  tar xjf "$1"    ;;
        *.tar.gz)   tar xzf "$1"    ;;
        *.tar.xz)   tar xJf "$1"    ;;
        *.bz2)      bunzip2 "$1"    ;;
        *.rar)      unrar x "$1"    ;;
        *.gz)       gunzip "$1"     ;;
        *.tar)      tar xf "$1"     ;;
        *.tbz2)     tar xjf "$1"    ;;
        *.tgz)      tar xzf "$1"    ;;
        *.zip)      unzip "$1"      ;;
        *.Z)        uncompress "$1" ;;
        *.7z)       7z x "$1"       ;;
        *.xz)       unxz "$1"       ;;
        *.zst)      unzstd "$1"     ;;
        *)
            echo "Cannot extract '$1': unrecognized format" >&2
            return 1
            ;;
    esac
}

# =============================================================================
# weather - Quick weather report
# =============================================================================
weather() {
    local location="${1:-}"
    curl -s "wttr.in/${location}?format=3"
}

# =============================================================================
# port - Check what process is using a port
# =============================================================================
port() {
    if [[ -z "$1" ]]; then
        echo "Usage: port <port_number>" >&2
        return 1
    fi
    lsof -i :"$1" 2>/dev/null || ss -tulnp | grep ":$1 " 2>/dev/null
}

# =============================================================================
# backup - Create a timestamped backup of a file
# =============================================================================
backup() {
    if [[ -z "$1" ]]; then
        echo "Usage: backup <file>" >&2
        return 1
    fi
    cp -a "$1" "${1}.backup.$(date +%Y%m%d%H%M%S)"
}

# =============================================================================
# tre - tree with sensible defaults
# =============================================================================
tre() {
    tree -aC -I '.git|node_modules|.venv|__pycache__' --dirsfirst "$@" | less -FRNX
}
