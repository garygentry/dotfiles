#!/usr/bin/env bash
# Integration test for uninstall command

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOTFILES_BIN="$PROJECT_ROOT/bin/dotfiles"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[TEST]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# Setup test environment
TEST_HOME=$(mktemp -d)
TEST_STATE_DIR=$(mktemp -d)

log_info "Test home: $TEST_HOME"
log_info "Test state: $TEST_STATE_DIR"

# Cleanup on exit
cleanup() {
    log_info "Cleaning up test environment..."
    rm -rf "$TEST_HOME"
    rm -rf "$TEST_STATE_DIR"
}
trap cleanup EXIT

# Create a minimal test module directly in project
setup_test_module() {
    local module_dir="$PROJECT_ROOT/modules/test-rollback"
    mkdir -p "$module_dir"

    cat > "$module_dir/module.yml" <<EOF
name: test-rollback
version: 1.0.0
description: Minimal test module for rollback testing
priority: 99
os: [ubuntu, darwin, arch]

files:
  - source: test.conf
    dest: ~/.testrollback/test.conf
    type: copy
EOF

    echo "test content" > "$module_dir/test.conf"

    cat > "$module_dir/install.sh" <<'EOF'
#!/usr/bin/env bash
echo "Test module installed"
EOF
    chmod +x "$module_dir/install.sh"

    log_info "Created test module at $module_dir"
}

# Remove test module
cleanup_test_module() {
    rm -rf "$PROJECT_ROOT/modules/test-rollback"
    log_info "Removed test module"
}

# Test: Install then uninstall
test_install_uninstall_cycle() {
    log_info "Test: Install-Uninstall cycle"

    setup_test_module

    # Install
    log_info "Installing test-rollback..."
    HOME="$TEST_HOME" DOTFILES_DIR="$PROJECT_ROOT" \
        "$DOTFILES_BIN" install test-rollback --unattended

    # Check state was recorded
    if [[ ! -f "$PROJECT_ROOT/.state/test-rollback.json" ]]; then
        log_error "State file was not created"
        cleanup_test_module
        return 1
    fi

    log_info "State file created"

    # Check deployed file exists
    if [[ ! -f "$TEST_HOME/.testrollback/test.conf" ]]; then
        log_error "Deployed file not found"
        cleanup_test_module
        return 1
    fi

    log_info "File deployed successfully"

    # Uninstall with force flag
    log_info "Uninstalling test-rollback..."
    HOME="$TEST_HOME" DOTFILES_DIR="$PROJECT_ROOT" "$DOTFILES_BIN" uninstall test-rollback --force

    # Check file was removed
    if [[ -f "$TEST_HOME/.testrollback/test.conf" ]]; then
        log_error "File was not removed by uninstall"
        cleanup_test_module
        return 1
    fi

    log_info "File removed successfully"

    # Check state was removed
    if [[ -f "$PROJECT_ROOT/.state/test-rollback.json" ]]; then
        log_error "State file was not removed"
        cleanup_test_module
        return 1
    fi

    log_info "State removed successfully"

    cleanup_test_module
    log_info "✓ Install-Uninstall cycle test passed"
    return 0
}

# Test: Dry-run doesn't modify anything
test_dryrun_uninstall() {
    log_info "Test: Dry-run uninstall"

    setup_test_module

    # Install
    HOME="$TEST_HOME" DOTFILES_DIR="$PROJECT_ROOT" \
        "$DOTFILES_BIN" install test-rollback --unattended

    # Dry-run uninstall
    log_info "Dry-run uninstall..."
    HOME="$TEST_HOME" DOTFILES_DIR="$PROJECT_ROOT" \
        "$DOTFILES_BIN" uninstall test-rollback --dry-run

    # File should still exist
    if [[ ! -f "$TEST_HOME/.testrollback/test.conf" ]]; then
        log_error "File was removed during dry-run"
        cleanup_test_module
        return 1
    fi

    # State should still exist
    if [[ ! -f "$PROJECT_ROOT/.state/test-rollback.json" ]]; then
        log_error "State was removed during dry-run"
        cleanup_test_module
        return 1
    fi

    log_info "Files preserved during dry-run"

    # Clean up
    HOME="$TEST_HOME" DOTFILES_DIR="$PROJECT_ROOT" "$DOTFILES_BIN" uninstall test-rollback --force

    cleanup_test_module
    log_info "✓ Dry-run test passed"
    return 0
}

# Test: Uninstall non-existent module
test_uninstall_nonexistent() {
    log_info "Test: Uninstall non-existent module"

    # Try to uninstall module that was never installed
    HOME="$TEST_HOME" DOTFILES_DIR="$PROJECT_ROOT" \
        "$DOTFILES_BIN" uninstall nonexistent-module 2>&1 | grep -q "not installed"

    if [[ $? -eq 0 ]]; then
        log_info "✓ Non-existent module test passed"
        return 0
    else
        log_error "Expected 'not installed' message"
        return 1
    fi
}

# Run all tests
main() {
    log_info "Starting uninstall integration tests..."
    log_info "====================================="

    local failed=0

    if ! test_install_uninstall_cycle; then
        failed=$((failed + 1))
    fi

    if ! test_dryrun_uninstall; then
        failed=$((failed + 1))
    fi

    if ! test_uninstall_nonexistent; then
        failed=$((failed + 1))
    fi

    log_info "====================================="
    if [[ $failed -eq 0 ]]; then
        log_info "All tests passed!"
        exit 0
    else
        log_error "$failed test(s) failed"
        exit 1
    fi
}

main "$@"
