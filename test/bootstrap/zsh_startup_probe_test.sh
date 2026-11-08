#!/usr/bin/env bash
# Hermetic test for the zsh module's startup-timing probe (issue #56).
#
# An interactive zsh opens /dev/tty for its line editor even when stdin is
# /dev/null, so the probe used to block on a hand install until Ctrl-D. We run
# modules/zsh/verify.sh (sourced after lib/helpers.sh, as the runner does) inside
# a pseudo-terminal from util-linux `script`, with a sandbox HOME, and assert it
# finishes promptly, passes, and still reports a startup time, both with setsid
# and on the perl detach used where setsid is missing (macOS). The sandbox .zshrc
# runs compinit over a world-writable directory, so an attached probe would be
# asked about it on the terminal (issue #84). With neither setsid nor perl the
# probe is skipped. Each case is capped by `timeout`, so a regression fails
# instead of hanging.
#
# Linux only (needs util-linux script + setsid, and zsh).
#
#   test/bootstrap/zsh_startup_probe_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

for _tool in zsh script setsid timeout perl; do
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
mkdir "${SANDBOX}/insecure-completions"
# World-writable: compaudit forgives group-writable directories under a private group.
chmod 777 "${SANDBOX}/insecure-completions"
cat > "${SANDBOX}/.zshrc" <<RC
PS1="> "
fpath+=("${SANDBOX}/insecure-completions")
autoload -Uz compinit
compinit -d "${SANDBOX}/.zcompdump"
RC
printf '# >>> dotfiles: path >>>\n# <<< dotfiles: path <<<\n' > "${SANDBOX}/.zshenv"
ln -s "${REPO_ROOT}/modules/zsh/aliases.zsh" "${SANDBOX}/.config/zsh/aliases.zsh"
ln -s "${REPO_ROOT}/modules/zsh/functions.zsh" "${SANDBOX}/.config/zsh/functions.zsh"

# make_path <dir> <excluded command>...: a PATH directory with every system
# command except the excluded ones.
make_path() {
    local dir="$1" _dir _f _name
    shift
    mkdir -p "$dir"
    for _dir in /usr/local/bin /usr/bin /bin; do
        [[ -d "$_dir" ]] || continue
        for _f in "$_dir"/*; do
            _name="${_f##*/}"
            [[ " $* " == *" ${_name} "* || -e "${dir}/${_name}" || -L "${dir}/${_name}" ]] && continue
            ln -s "$_f" "${dir}/${_name}"
        done
    done
}
NOSETSID="${TMPROOT}/nosetsid-bin"
make_path "$NOSETSID" setsid
NODETACH="${TMPROOT}/nodetach-bin"
make_path "$NODETACH" setsid perl
# Like a stock Mac: no setsid and no timeout, so perl both detaches and caps.
MACLIKE="${TMPROOT}/maclike-bin"
make_path "$MACLIKE" setsid timeout gtimeout
TIMEOUT_BIN="$(command -v timeout)"

# The terminal's input stays open and silent. With `</dev/null`, script passes EOF
# to the terminal, which can answer compinit's question and hide the hang.
mkfifo "${TMPROOT}/tty-input"
exec 3<>"${TMPROOT}/tty-input"

# run_case <name> <PATH> [expected output]
run_case() {
    local name="$1" path="$2" expected="${3:-Zsh interactive startup:}" out rc start elapsed
    local inner
    inner="set -euo pipefail; source '${REPO_ROOT}/lib/helpers.sh'; source '${REPO_ROOT}/modules/zsh/verify.sh'"
    start=$SECONDS
    rc=0
    out="$(PATH="$path" DOTFILES_HOME="${SANDBOX}" HOME="${SANDBOX}" \
        DOTFILES_MODULE_DIR="${REPO_ROOT}/modules/zsh" DOTFILES_DRY_RUN=false \
        "$TIMEOUT_BIN" 15 script -qec "bash -c \"${inner}\"" /dev/null <&3 2>&1)" || rc=$?
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
    if [[ "$out" == *"$expected"* ]]; then
        pass "${name}: '${expected}' in ${elapsed}s"
    else
        fail "${name}: output lacks '${expected}'"
        printf '%s\n' "$out" >&2
    fi
}

run_case "setsid" "$PATH"
run_case "no setsid (perl detach)" "${NOSETSID}"
run_case "no setsid or timeout (macOS)" "${MACLIKE}"
run_case "no setsid or perl (skipped)" "${NODETACH}" "Skipping the zsh startup time check"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[[ $FAIL -eq 0 ]]
