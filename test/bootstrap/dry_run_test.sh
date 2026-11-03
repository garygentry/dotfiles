#!/usr/bin/env bash
# Hermetic test: `bootstrap.sh --dry-run` changes nothing (#58).
#
# No network, no root: a bare git repo on the local filesystem is the "remote",
# and a fake `go` on PATH "builds" a stub binary that records its arguments.
# bootstrap.sh is sourced as a library (DOTFILES_BOOTSTRAP_LIB=1) and main runs
# in a subshell per case, against a temp HOME.
#
#   test/bootstrap/dry_run_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck source=test/bootstrap/lib.sh
source "${SCRIPT_DIR}/lib.sh"

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }
check() { local desc="$1"; shift; if "$@"; then pass "$desc"; else fail "$desc"; fi; }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

git_q() { git -c user.email=t@example.com -c user.name=tester -c init.defaultBranch=main \
              -c commit.gpgsign=false "$@"; }

# Remote: main C1 -> C2.
WORK="${TMPROOT}/work"
BARE="${TMPROOT}/remote.git"
git_q init -q "$WORK"
(cd "$WORK" && echo one > f && git_q add f && git_q commit -q -m C1 && echo two > f && git_q commit -qam C2)
git_q clone -q --bare "$WORK" "$BARE"
C1="$(git_q -C "$BARE" rev-parse main~1)"

# Fake go: `go build -o OUT .` writes a stub that logs its args to $STUB_LOG.
mkdir -p "${TMPROOT}/bin"
cat > "${TMPROOT}/bin/go" <<'GO'
#!/usr/bin/env bash
[ "$1" = version ] && { echo "go version fake"; exit 0; }
out=""; while [ $# -gt 0 ]; do [ "$1" = -o ] && out="$2"; shift; done
printf '#!/usr/bin/env bash\necho "$*" > "$STUB_LOG"\n' > "$out"; chmod +x "$out"
GO
chmod +x "${TMPROOT}/bin/go"

# run_main <name> <args...>: HOME=${TMPROOT}/<name>; output in <name>.log, status in
# ERREXIT_RC. Never call it from a condition (so not through check): see lib.sh.
main_case() {
    local name="$1"; shift
    export HOME="${TMPROOT}/${name}" PATH="${TMPROOT}/bin:${PATH}"
    export DOTFILES_REPO="$BARE" DOTFILES_DIR="${DOTFILES_DIR_UNDER_TEST}"
    export STUB_LOG="${TMPROOT}/${name}.stub"
    unset DOTFILES_CONTENT_DIR
    [ -z "${PRESET_CONTENT_DIR:-}" ] || export DOTFILES_CONTENT_DIR="$PRESET_CONTENT_DIR"
    export DOTFILES_BOOTSTRAP_LIB=1
    # shellcheck source=/dev/null
    source "${REPO_ROOT}/bootstrap.sh"
    main "$@" < /dev/null
}
run_main() {
    mkdir -p "${TMPROOT}/$1"
    run_errexit main_case "$@" > "${TMPROOT}/$1.log" 2>&1
}
check_main() {
    local desc="$1"; shift
    run_main "$@"
    if [ "$ERREXIT_RC" -eq 0 ]; then
        pass "$desc"
    else
        fail "$desc (rc=${ERREXIT_RC})"
        tail -n 5 "${TMPROOT}/$1.log" >&2
    fi
}
logs() { grep -qF -- "$2" "${TMPROOT}/$1.log"; }

# T1: fresh host — nothing is cloned; the preview says why it can't run.
DOTFILES_DIR_UNDER_TEST="${TMPROOT}/fresh/.dotfiles"
check_main "T1 dry run exits 0 on a fresh host" fresh --dry-run
check "T1 no engine checkout created" test ! -e "$DOTFILES_DIR_UNDER_TEST"
check "T1 says it would clone" logs fresh "would clone ${BARE}"
check "T1 says the preview needs prep" logs fresh "No install preview"

# T2: checkout at an older commit — HEAD stays; no binary; preview blocked.
OLD="${TMPROOT}/old/.dotfiles"
git_q clone -q "$BARE" "$OLD" && git_q -C "$OLD" reset -q --hard "$C1"
DOTFILES_DIR_UNDER_TEST="$OLD"
check_main "T2 dry run exits 0" old --dry-run
check "T2 HEAD unchanged" test "$(git -C "$OLD" rev-parse HEAD)" = "$C1"
check "T2 says it would reset" logs old "would reset ${OLD} from ${C1:0:7}"
check "T2 no bin/dotfiles written" test ! -e "${OLD}/bin/dotfiles"

# T3: checkout at the ref with local edits and an untracked file — both kept.
DIRTY="${TMPROOT}/dirty/.dotfiles"
git_q clone -q "$BARE" "$DIRTY"
echo edited > "${DIRTY}/f"; echo stray > "${DIRTY}/stray"
DOTFILES_DIR_UNDER_TEST="$DIRTY"
check_main "T3 dry run exits 0" dirty --dry-run
check "T3 local edit kept" grep -qx edited "${DIRTY}/f"
check "T3 untracked file kept" test -e "${DIRTY}/stray"
check "T3 no .backups patch written" test ! -e "${DIRTY}/.backups"
check "T3 says it would discard changes" logs dirty "would save local changes"

# T4: clean checkout at the ref — preview runs from a temp build; bin/ untouched;
# --content-path persists nothing to ~/.zshenv.
CLEAN="${TMPROOT}/clean/.dotfiles"
git_q clone -q "$BARE" "$CLEAN"
mkdir -p "${TMPROOT}/content"
DOTFILES_DIR_UNDER_TEST="$CLEAN"
check_main "T4 dry run exits 0" clean --dry-run --content-path "${TMPROOT}/content" --profile x
check "T4 install ran with --dry-run" grep -qx -- "install --dry-run --profile x" "${TMPROOT}/clean.stub"
check "T4 no bin/dotfiles written" test ! -e "${CLEAN}/bin/dotfiles"
check "T4 ~/.zshenv not written" test ! -e "${TMPROOT}/clean/.zshenv"
check "T4 says it would persist the content dir" logs clean "would append that export line"

# T5: --content-repo with nothing cloned yet — no clone; the preview says why it can't run.
CLEAN5="${TMPROOT}/repo-fresh/.dotfiles"
git_q clone -q "$BARE" "$CLEAN5"
DOTFILES_DIR_UNDER_TEST="$CLEAN5"
check_main "T5 dry run exits 0" repo-fresh --dry-run --content-repo "$BARE"
check "T5 no overlay cloned" test ! -e "${TMPROOT}/repo-fresh/.config/dotfiles"
check "T5 says it would clone the overlay" logs repo-fresh "would clone ${BARE} to"

# T6: --content-repo already cloned, behind its remote — it stays where it is.
CLEAN6="${TMPROOT}/repo-old/.dotfiles"
OVERLAY6="${TMPROOT}/repo-old/.config/dotfiles"
git_q clone -q "$BARE" "$CLEAN6"
git_q clone -q "$BARE" "$OVERLAY6" && git_q -C "$OVERLAY6" reset -q --hard "$C1"
DOTFILES_DIR_UNDER_TEST="$CLEAN6"
check_main "T6 dry run exits 0" repo-old --dry-run --content-repo "$BARE"
check "T6 overlay HEAD unchanged" test "$(git -C "$OVERLAY6" rev-parse HEAD)" = "$C1"
check "T6 says it would fetch the overlay" logs repo-old "would fetch ${OVERLAY6}"

# T7: DOTFILES_CONTENT_DIR already set, behind its upstream — it stays where it is.
CLEAN7="${TMPROOT}/preset/.dotfiles"
OVERLAY7="${TMPROOT}/preset-overlay"
git_q clone -q "$BARE" "$CLEAN7"
git_q clone -q "$BARE" "$OVERLAY7" && git_q -C "$OVERLAY7" reset -q --hard "$C1"
DOTFILES_DIR_UNDER_TEST="$CLEAN7"
PRESET_CONTENT_DIR="$OVERLAY7"
check_main "T7 dry run exits 0" preset --dry-run
PRESET_CONTENT_DIR=""
check "T7 overlay HEAD unchanged" test "$(git -C "$OVERLAY7" rev-parse HEAD)" = "$C1"
check "T7 says it would fast-forward the overlay" logs preset "would fast-forward the content overlay at ${OVERLAY7}"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
