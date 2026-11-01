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

# run_main <name> <args...>: HOME=${TMPROOT}/<name>; output in <name>.log.
run_main() {
    local name="$1"; shift
    mkdir -p "${TMPROOT}/${name}"
    (
        export HOME="${TMPROOT}/${name}" PATH="${TMPROOT}/bin:${PATH}"
        export DOTFILES_REPO="$BARE" DOTFILES_DIR="${DOTFILES_DIR_UNDER_TEST}"
        export STUB_LOG="${TMPROOT}/${name}.stub"
        unset DOTFILES_CONTENT_DIR
        export DOTFILES_BOOTSTRAP_LIB=1
        # shellcheck source=/dev/null
        source "${REPO_ROOT}/bootstrap.sh"
        main "$@" < /dev/null
    ) > "${TMPROOT}/${name}.log" 2>&1
}
logs() { grep -qF -- "$2" "${TMPROOT}/$1.log"; }

# T1: fresh host — nothing is cloned; the preview says why it can't run.
DOTFILES_DIR_UNDER_TEST="${TMPROOT}/fresh/.dotfiles"
check "T1 dry run exits 0 on a fresh host" run_main fresh --dry-run
check "T1 no engine checkout created" test ! -e "$DOTFILES_DIR_UNDER_TEST"
check "T1 says it would clone" logs fresh "would clone ${BARE}"
check "T1 says the preview needs prep" logs fresh "No install preview"

# T2: checkout at an older commit — HEAD stays; no binary; preview blocked.
OLD="${TMPROOT}/old/.dotfiles"
git_q clone -q "$BARE" "$OLD" && git_q -C "$OLD" reset -q --hard "$C1"
DOTFILES_DIR_UNDER_TEST="$OLD"
check "T2 dry run exits 0" run_main old --dry-run
check "T2 HEAD unchanged" test "$(git -C "$OLD" rev-parse HEAD)" = "$C1"
check "T2 says it would reset" logs old "would reset ${OLD} from ${C1:0:7}"
check "T2 no bin/dotfiles written" test ! -e "${OLD}/bin/dotfiles"

# T3: checkout at the ref with local edits and an untracked file — both kept.
DIRTY="${TMPROOT}/dirty/.dotfiles"
git_q clone -q "$BARE" "$DIRTY"
echo edited > "${DIRTY}/f"; echo stray > "${DIRTY}/stray"
DOTFILES_DIR_UNDER_TEST="$DIRTY"
check "T3 dry run exits 0" run_main dirty --dry-run
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
check "T4 dry run exits 0" run_main clean --dry-run --content-path "${TMPROOT}/content" --profile x
check "T4 install ran with --dry-run" grep -qx -- "install --dry-run --profile x" "${TMPROOT}/clean.stub"
check "T4 no bin/dotfiles written" test ! -e "${CLEAN}/bin/dotfiles"
check "T4 ~/.zshenv not written" test ! -e "${TMPROOT}/clean/.zshenv"
check "T4 says it would persist the content dir" logs clean "would append that export line"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
