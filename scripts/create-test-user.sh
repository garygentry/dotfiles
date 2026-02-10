#!/usr/bin/env bash
# scripts/create-test-user.sh - Create a clean user account for testing dotfiles
#
# Must be run as root or with sudo.
# Usage: sudo ./scripts/create-test-user.sh

set -euo pipefail

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    BLUE='\033[0;34m'; BOLD='\033[1m'; RESET='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; BOLD=''; RESET=''
fi

info()  { printf "${BLUE}${BOLD}[info]${RESET}  %s\n" "$*"; }
ok()    { printf "${GREEN}${BOLD}[ok]${RESET}    %s\n" "$*"; }
warn()  { printf "${YELLOW}${BOLD}[warn]${RESET}  %s\n" "$*" >&2; }
fatal() { printf "${RED}${BOLD}[error]${RESET} %s\n" "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Root check
# ---------------------------------------------------------------------------
if [ "$(id -u)" -ne 0 ]; then
    fatal "This script must be run as root (use sudo)"
fi

# ---------------------------------------------------------------------------
# Prompt for username
# ---------------------------------------------------------------------------
read -rp "Enter username for test user: " USERNAME

if [ -z "$USERNAME" ]; then
    fatal "Username cannot be empty"
fi

# Reject obviously bad usernames
if [[ ! "$USERNAME" =~ ^[a-z_][a-z0-9_-]*$ ]]; then
    fatal "Invalid username '$USERNAME' (use lowercase letters, digits, hyphens, underscores)"
fi

# ---------------------------------------------------------------------------
# Handle existing user
# ---------------------------------------------------------------------------
if id "$USERNAME" &>/dev/null; then
    warn "User '$USERNAME' already exists"

    printf "\n"
    printf "${RED}${BOLD}  WARNING: This will permanently delete user '%s' and ALL their data.${RESET}\n" "$USERNAME"
    printf "${RED}${BOLD}  Home directory /home/%s will be removed.${RESET}\n" "$USERNAME"
    printf "${RED}${BOLD}  Any running processes owned by this user will be killed.${RESET}\n" "$USERNAME"
    printf "\n"
    read -rp "Type the username '$USERNAME' to confirm deletion: " CONFIRM

    if [ "$CONFIRM" != "$USERNAME" ]; then
        fatal "Confirmation did not match. Aborting."
    fi

    info "Killing any processes owned by '$USERNAME'..."
    pkill -u "$USERNAME" 2>/dev/null || true
    sleep 1
    pkill -9 -u "$USERNAME" 2>/dev/null || true

    info "Removing user '$USERNAME' and home directory..."
    userdel -r "$USERNAME" 2>/dev/null || {
        # userdel -r may fail if home dir doesn't exist; force cleanup
        userdel "$USERNAME" 2>/dev/null || true
        rm -rf "/home/$USERNAME"
    }

    # Clean up any lingering state
    rm -rf "/var/mail/$USERNAME" 2>/dev/null || true

    ok "User '$USERNAME' removed"
fi

# ---------------------------------------------------------------------------
# Create user
# ---------------------------------------------------------------------------
info "Creating user '$USERNAME'..."
useradd -m -s /bin/bash "$USERNAME"
ok "User '$USERNAME' created with home at /home/$USERNAME"

# ---------------------------------------------------------------------------
# Set password
# ---------------------------------------------------------------------------
info "Set a password for '$USERNAME':"
passwd "$USERNAME"

# ---------------------------------------------------------------------------
# Add to sudo group
# ---------------------------------------------------------------------------
if getent group sudo &>/dev/null; then
    SUDO_GROUP="sudo"
elif getent group wheel &>/dev/null; then
    SUDO_GROUP="wheel"
else
    fatal "Neither 'sudo' nor 'wheel' group found. Add the user to the appropriate group manually."
fi

usermod -aG "$SUDO_GROUP" "$USERNAME"
ok "Added '$USERNAME' to '$SUDO_GROUP' group"

# ---------------------------------------------------------------------------
# Ensure sudo is installed
# ---------------------------------------------------------------------------
if ! command -v sudo &>/dev/null; then
    warn "sudo is not installed. Installing..."
    if command -v apt-get &>/dev/null; then
        apt-get update -qq && apt-get install -y -qq sudo
    elif command -v pacman &>/dev/null; then
        pacman -Sy --noconfirm sudo
    else
        warn "Could not install sudo automatically. Please install it manually."
    fi
fi

# ---------------------------------------------------------------------------
# Install basic prerequisites (git, curl)
# ---------------------------------------------------------------------------
info "Ensuring basic packages are available (git, curl)..."
if command -v apt-get &>/dev/null; then
    apt-get update -qq && apt-get install -y -qq git curl
elif command -v pacman &>/dev/null; then
    pacman -Sy --noconfirm git curl
fi
ok "Basic packages available"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
printf "\n"
printf "${GREEN}${BOLD}============================================${RESET}\n"
printf "${GREEN}${BOLD}  Test user '%s' is ready${RESET}\n" "$USERNAME"
printf "${GREEN}${BOLD}============================================${RESET}\n"
printf "\n"
printf "  Switch to the user:\n"
printf "    ${BOLD}su - %s${RESET}\n" "$USERNAME"
printf "\n"
printf "  Run dotfiles bootstrap:\n"
printf "    ${BOLD}curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash${RESET}\n"
printf "\n"
printf "  Or clone and run locally:\n"
printf "    ${BOLD}git clone https://github.com/garygentry/dotfiles.git ~/.dotfiles && cd ~/.dotfiles${RESET}\n"
printf "    ${BOLD}sudo ./scripts/create-test-user.sh  # (you're here)${RESET}\n"
printf "\n"
