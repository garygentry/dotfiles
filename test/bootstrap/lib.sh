#!/usr/bin/env bash
# Shared by the bootstrap tests. Not a test itself: `make test-bootstrap` runs *_test.sh.

# run_errexit <command...>: run the command in a subshell with errexit on, the way the
# real scripts run, and put its exit status in ERREXIT_RC. Returns 0 either way.
#
# Bash ignores set -e all the way down under an if, &&, || or !, even a set -e set inside.
# Running the code under test there hides any failing command that isn't its last one
# (#83), so a call from there stops the test script instead of passing quietly. Callers
# redirect stderr and may be in $(...), so the guard uses the script's own stderr (fd 9)
# and signals the script itself.
exec 9>&2
run_errexit() {
    set +e
    ( set -e; false; exit 0 )
    if [ "$?" -eq 0 ]; then
        printf 'test bug: run_errexit called from a condition, errexit is off: %s\n' "$*" >&9
        kill "$$"
        exit 2
    fi
    ( set -e; "$@" )
    ERREXIT_RC=$?
    set -e
}
