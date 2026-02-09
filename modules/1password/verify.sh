# 1password/verify.sh - Verify 1Password CLI installation

if command -v op &>/dev/null; then
    _op_version="$(op --version 2>/dev/null || true)"
    if [[ -n "$_op_version" ]]; then
        log_success "1Password CLI verified: v${_op_version}"
    else
        log_warn "1Password CLI found but --version returned empty output"
    fi
else
    log_error "1Password CLI (op) is not installed"
    return 1
fi
