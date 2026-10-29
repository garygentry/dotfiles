#!/usr/bin/env bash
# Hermetic test for the zsh module's startup-timing probe (issue #56).
#
# An interactive zsh opens /dev/tty for its line editor even when stdin is
# /dev/null, so the probe used to block on a hand install until Ctrl-D. We run
# modules/zsh/verify.sh (sourced after lib/helpers.sh, as the runner does) inside
# a pseudo-terminal from util-linux `script`, with a sandbox HOME, and assert it
# finishes promptly, passes, and still reports a startup time, both with setsid
# and on the `zsh -i -c exit` fallback used where setsid is missing (macOS).
# Each case is capped by `timeout`, so a regression fails instead of hanging.
#
# Linux only (needs util-linux script + setsid, and zsh).
#
#   test/bootstrap/zsh_startup_probe_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

for _tool in zsh script setsid timeout; do
    if ! command -v "$_tool" &>/dev/null; then
        printf 'SKIP: %s not found\n' "$_tool"
        exit 0
    fi
done

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

# Sandbox home that satisfies verify.sh's other checks, with a trivial .zshrc.
SANDBOX="${TMPROOT}/home"
mkdir -p "${SANDBOX}/.config/zsh" "${SANDBOX}/.local/share/zinit/zinit.git"
printf 'PS1="> "\n' > "${SANDBOX}/.zshrc"
printf '# >>> dotfiles: path >>>\n# <<< dotfiles: path <<<\n' > "${SANDBOX}/.zshenv"
ln -s "${REPO_ROOT}/modules/zsh/aliases.zsh" "${SANDBOX}/.config/zsh/aliases.zsh"
ln -s "${REPO_ROOT}/modules/zsh/functions.zsh" "${SANDBOX}/.config/zsh/functions.zsh"

# A PATH with every system command except setsid, to exercise the fallback.
NOSETSID="${TMPROOT}/nosetsid-bin"
mkdir -p "${NOSETSID}"
for _dir in /usr/local/bin /usr/bin /bin; do
    [[ -d "$_dir" ]] || continue
    for _f in "$_dir"/*; do
        _name="${_f##*/}"
        [[ "$_name" == setsid || -e "${NOSETSID}/${_name}" || -L "${NOSETSID}/${_name}" ]] && continue
        ln -s "$_f" "${NOSETSID}/${_name}"
    done
done

# run_case <name> <PATH>
run_case() {
    local name="$1" path="$2" out rc start elapsed
    local inner
    inner="set -euo pipefail; source '${REPO_ROOT}/lib/helpers.sh'; source '${REPO_ROOT}/modules/zsh/verify.sh'"
    start=$SECONDS
    rc=0
    out="$(PATH="$path" DOTFILES_HOME="${SANDBOX}" HOME="${SANDBOX}" \
        DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/zsh" DOTFILES_DRY_RUN=false \
        timeout 15 script -qec "bash -c \"${inner}\"" /dev/null </dev/null 2>&1)" || rc=$?
    elapsed=$((SECONDS - start))

    if [[ $rc -eq 124 ]]; then
        fail "${name}: verify hung (killed after 15s)"
        return
    fi
    if [[ $rc -ne 0 ]]; then
        fail "${name}: verify exited ${rc}"
        printf '%s\n' "$out" >&2
        return
    fi
    if [[ "$out" == *"Zsh interactive startup:"* ]]; then
        pass "${name}: startup time reported in ${elapsed}s"
    else
        fail "${name}: no startup time reported"
        printf '%s\n' "$out" >&2
    fi
}

run_case "setsid" "$PATH"
run_case "no setsid (zsh -i -c exit)" "${NOSETSID}"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
