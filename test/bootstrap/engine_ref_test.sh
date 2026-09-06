#!/usr/bin/env bash
# Hermetic tests for bootstrap.sh engine-ref checkout (ensure_repo / update_repo).
#
# No network, no root: a bare git repo on the local filesystem is the "remote".
# bootstrap.sh is sourced as a library (DOTFILES_BOOTSTRAP_LIB=1) so we can call
# ensure_repo directly. The point is the client-deploy-model D5 pin: DOTFILES_REF
# must resolve a full commit SHA, a tag, AND a branch — fresh clone and update —
# not just a branch. The prior code (`clone --branch <ref>` + `reset --hard
# origin/<ref>`) worked for branches only: a SHA silently fell back to the default
# branch on a fresh clone and fatal'd on update. These tests would fail against
# that code and pass against the fetch + `reset --hard FETCH_HEAD` fix.
#
#   test/bootstrap/engine_ref_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

export DOTFILES_BOOTSTRAP_LIB=1
# shellcheck source=/dev/null
source "${REPO_ROOT}/bootstrap.sh"

PASS=0
FAIL=0
fail() { printf 'FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
pass() { printf 'PASS: %s\n' "$*"; PASS=$((PASS + 1)); }

TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

git_q() { git -c user.email=t@example.com -c user.name=tester -c init.defaultBranch=main \
              -c commit.gpgsign=false -c tag.gpgSign=false -c tag.forceSignAnnotated=false "$@"; }

# --- build a bare "remote" with a known history -----------------------------
# main:  C1 -> C2 -> C3 (tip);  tag v1 at C2;  branch feature off C2 -> F1.
WORK="${TMPROOT}/work"
BARE="${TMPROOT}/remote.git"
git_q init -q "${WORK}"
(
  cd "${WORK}"
  echo one   > f; git_q add f; git_q commit -q -m C1; C1="$(git_q rev-parse HEAD)"
  echo two   > f; git_q add f; git_q commit -q -m C2; C2="$(git_q rev-parse HEAD)"
  git_q tag v1 "${C2}"
  git_q checkout -q -b feature "${C2}"; echo feat > f; git_q add f; git_q commit -q -m F1
  FEAT="$(git_q rev-parse HEAD)"
  git_q checkout -q main
  echo three > f; git_q add f; git_q commit -q -m C3
  printf '%s %s %s\n' "$C1" "$C2" "$FEAT" > "${TMPROOT}/shas"
)
git_q clone -q --bare "${WORK}" "${BARE}"
read -r C1 C2 FEAT < "${TMPROOT}/shas"
C3="$(git_q -C "${BARE}" rev-parse main)"

export DOTFILES_REPO="${BARE}"

# head_of DIR → the resolved HEAD sha of a checkout
head_of() { git -C "$1" rev-parse HEAD 2>/dev/null || echo MISSING; }

# run ensure_repo for REF into DIR, in a subshell (fatal calls exit); returns rc
run_ensure() {
  local dir="$1" ref="$2"
  ( DOTFILES_DIR="$dir" DOTFILES_REF="$ref" ensure_repo ) >/dev/null 2>&1
}

# --- T1: fresh clone at a NON-TIP SHA (the case the old code silently botched)
D1="${TMPROOT}/d1"
run_ensure "$D1" "$C2" && [ "$(head_of "$D1")" = "$C2" ] \
  && pass "T1 fresh clone at non-tip SHA lands exactly on it" \
  || fail "T1 fresh SHA: got $(head_of "$D1") want $C2"

# --- T2: update an existing checkout to a DIFFERENT (older) SHA --------------
run_ensure "$D1" "$C1" && [ "$(head_of "$D1")" = "$C1" ] \
  && pass "T2 update to older SHA moves HEAD back to it" \
  || fail "T2 update SHA: got $(head_of "$D1") want $C1"

# --- T3: fresh clone at a TAG ----------------------------------------------
D3="${TMPROOT}/d3"
run_ensure "$D3" "v1" && [ "$(head_of "$D3")" = "$C2" ] \
  && pass "T3 fresh clone at tag resolves to the tagged commit" \
  || fail "T3 tag: got $(head_of "$D3") want $C2"

# --- T4: fresh clone at a BRANCH tip (regression guard) ---------------------
D4="${TMPROOT}/d4"
run_ensure "$D4" "feature" && [ "$(head_of "$D4")" = "$FEAT" ] \
  && pass "T4 fresh clone at branch resolves to branch tip" \
  || fail "T4 branch: got $(head_of "$D4") want $FEAT"

# --- T5: default 'main' still tracks the branch tip -------------------------
D5="${TMPROOT}/d5"
run_ensure "$D5" "main" && [ "$(head_of "$D5")" = "$C3" ] \
  && pass "T5 ref=main lands on main tip" \
  || fail "T5 main: got $(head_of "$D5") want $C3"

# --- T6: a bad/nonexistent ref FAILS LOUD (fatal, non-zero) -----------------
# ensure_repo must return non-zero so main() aborts BEFORE build_and_run — the
# engine never builds/runs on an unintended ref. (On a fresh host the default
# branch is cloned to disk before update_repo fatals on the bad-ref fetch; that
# leftover checkout is a don't-care because the loud non-zero exit prevents any
# build. The old code, by contrast, would clone the default branch and SUCCEED.)
D6="${TMPROOT}/d6"
if run_ensure "$D6" "0000000000000000000000000000000000000000"; then
  fail "T6 bad ref: ensure_repo returned 0 (should fatal loud, aborting before build)"
else
  pass "T6 bad ref fails loud (non-zero) — main() aborts before build_and_run"
fi

# --- T7: update path preserves the reset semantics on a dirty tree ----------
# (dirty tracked file is discarded by reset --hard FETCH_HEAD; a recovery patch
#  is written). Proves the FETCH_HEAD reset still converges from a dirty state.
echo LOCAL_EDIT > "${D1}/f"
run_ensure "$D1" "$C2" && [ "$(head_of "$D1")" = "$C2" ] \
  && [ "$(cat "${D1}/f")" = "two" ] \
  && pass "T7 dirty tree reset to ref via FETCH_HEAD (edit discarded)" \
  || fail "T7 dirty update: HEAD $(head_of "$D1") / f='$(cat "${D1}/f" 2>/dev/null)'"

printf '\n%d passed, %d failed\n' "${PASS}" "${FAIL}"
[ "${FAIL}" -eq 0 ]
