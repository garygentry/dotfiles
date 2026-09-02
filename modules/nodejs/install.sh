#!/usr/bin/env bash
# nodejs/install.sh - Install Node.js (sudo-free, official prebuilt tarball)
#
# apt-based node needs sudo, so on a no-sudo host it degrades to installing
# nothing. This installs the official Node.js prebuilt tarball into ~/.local
# (the same deterministic, sudo-free pattern as Go and claude-code),
# checksum-verified against nodejs.org's SHASUMS256.txt. Covers Linux (glibc)
# and macOS; there is no official musl prebuilt (Alpine — see the guard below).
# Pin the LTS line with DOTFILES_NODE_MAJOR (default 22).
#
# gnet-lg ADR 0025 Decision 6 — runtime + deps hardening (generic; no consumer
# specifics — the seam holds):
#   * Stable global npm prefix. Pin npm's global prefix to ~/.local so `npm i -g`
#     bins land in ~/.local/bin (already on PATH) and modules in ~/.local/lib —
#     a location that SURVIVES a Node version switch (the NVM-prefix hazard:
#     global installs vanish from PATH when the active node dir changes) and
#     needs no sudo. Enforced even when node itself is system-provided.
#   * Node floor. An optional minimum version (modules.nodejs.min_version in the
#     overlay BOM -> DOTFILES_SETTING_MIN_VERSION) is a HARD requirement: a found
#     node below it triggers the sudo-free tarball install, and a floor the
#     pinned LTS line cannot satisfy is a loud failure (never a hollow success).
#     The version fact lives in the BOM, never hard-coded here.
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_share="${_home}/.local/share/node"
_min="${DOTFILES_SETTING_MIN_VERSION:-}"      # optional node floor (overlay BOM)

# Is $1 >= $2? Numeric major.minor.patch compare, tolerant of a floor spelled with
# fewer fields (e.g. min "22" or "22.19"): both operands are padded to exactly a.b.c
# so a numeric tie means EQUAL, and equal satisfies >=. Printing the padded $2 first
# makes the tie resolve to $2 (== "equal is >="), fixing the equality-boundary case
# where the old form (unpadded, $1 first) wrongly reported equal as "below".
_ver_ge() {
    local a b
    a="$(printf '%s.0.0' "$1" | cut -d. -f1-3)"
    b="$(printf '%s.0.0' "$2" | cut -d. -f1-3)"
    [[ "$(printf '%s\n%s\n' "$b" "$a" | sort -t. -k1,1n -k2,2n -k3,3n | head -1)" == "$b" ]]
}

# Extract a bare a.b.c from `node --version` output (v22.22.0 -> 22.22.0).
_node_ver_of() { "$1" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true; }

# Pin npm's global prefix to ~/.local, idempotently, preserving any other user
# ~/.npmrc settings. Written unconditionally (a host on a system node still gets
# a deterministic, node-switch-safe global prefix) — except under dry-run.
_ensure_npm_prefix() {
    local npmrc="${_home}/.npmrc" want="${_home}/.local"
    if is_dry_run; then
        log_info "[dry-run] Would pin npm global prefix -> ${want} (${npmrc})"
        return 0
    fi
    if [[ -f "$npmrc" ]]; then
        # Only the exact resolved path counts as satisfied: npm does not expand a
        # literal '~' in .npmrc, so a '~/.local' spelling is a broken prefix we must
        # rewrite, not accept. (Reading the value to COMPARE is safe; we never re-emit
        # it — the rewrite below drops+appends rather than sed-substituting the path,
        # so a HOME with sed metacharacters can't break or inject the write.)
        local cur
        cur="$(grep -E '^[[:space:]]*prefix[[:space:]]*=' "$npmrc" | tail -1 \
               | sed -E 's/^[[:space:]]*prefix[[:space:]]*=[[:space:]]*//; s/[[:space:]]*$//')"
        [[ "$cur" == "$want" ]] && return 0
    fi
    local tmp; tmp="$(mktemp)"
    # Preserve every other setting: drop any existing prefix line(s), then append ours.
    if [[ -f "$npmrc" ]]; then
        grep -vE '^[[:space:]]*prefix[[:space:]]*=' "$npmrc" > "$tmp" 2>/dev/null || true
    fi
    printf 'prefix=%s\n' "$want" >> "$tmp"
    mv "$tmp" "$npmrc"
    log_info "npm: pinned global prefix -> ${want} (${npmrc})"
}

# A stable global prefix is the whole point (the NVM-prefix hazard) — do it first,
# so even the already-installed fast path below leaves npm deterministic.
_ensure_npm_prefix

# Already installed? Match PATH (ignoring a Windows node under /mnt on WSL) OR
# our own ~/.local/bin/node, which may not be on PATH yet on a fresh host.
_node_path="$(command -v node 2>/dev/null || true)"
[[ -n "$_node_path" && "$_node_path" == /mnt/* ]] && _node_path=""
[[ -z "$_node_path" && -x "${_bin}/node" ]] && _node_path="${_bin}/node"
if [[ -n "$_node_path" ]]; then
    _cur_ver="$(_node_ver_of "$_node_path")"
    if [[ -z "$_cur_ver" ]]; then
        # Present but won't run (e.g. a glibc build on musl, a truncated binary) — do
        # not report a hollow "already installed"; fall through and install a real one.
        log_warn "Node.js at $_node_path does not run — installing a working Node.js into ${_share}"
    else
        # Honor the floor: a found node BELOW the minimum is not "already installed"
        # for our purposes — fall through and install a compliant node into ~/.local.
        _floor_ok=true
        if [[ -n "$_min" ]] && ! _ver_ge "$_cur_ver" "$_min"; then
            _floor_ok=false
        fi
        if [[ "$_floor_ok" == true ]]; then
            log_info "Node.js already installed: v${_cur_ver} ($_node_path)"
            exit 0
        fi
        log_warn "Node.js ${_cur_ver} ($_node_path) is below the required floor ${_min} — installing a compliant Node.js into ${_share}"
    fi
fi

if is_dry_run; then
    log_info "[dry-run] Would install Node.js (sudo-free tarball) to ${_share}${_min:+ (floor >=${_min})}"
    exit 0
fi

# Platform (node's naming: linux/darwin, x64/arm64).
case "$(uname -s)" in
    Linux)  _os="linux" ;;
    Darwin) _os="darwin" ;;
    *) log_error "Node.js: unsupported OS $(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64)  _arch="x64" ;;
    arm64|aarch64) _arch="arm64" ;;
    *) log_error "Node.js: unsupported architecture $(uname -m)"; exit 1 ;;
esac
_platform="${_os}-${_arch}"

# nodejs.org publishes no musl prebuilt — a glibc tarball won't run on Alpine.
# Fail loudly with guidance rather than install a broken binary.
if [[ "$_os" == "linux" ]] && { [[ -e /lib/libc.musl-x86_64.so.1 ]] || [[ -e /lib/libc.musl-aarch64.so.1 ]] || ldd /bin/ls 2>&1 | grep -q musl; }; then
    log_error "Node.js: no official prebuilt for musl/Alpine — install it with your package manager (e.g. 'apk add nodejs npm')"
    exit 1
fi

_major="${DOTFILES_NODE_MAJOR:-${DOTFILES_SETTING_NODE_MAJOR:-22}}"
_base="https://nodejs.org/dist/latest-v${_major}.x"

# SHASUMS256.txt gives us both the exact filename (newest patch of this LTS line)
# and its checksum, so we never hardcode a patch version.
_shasums="$(curl -fsSL "${_base}/SHASUMS256.txt" 2>/dev/null || true)"
if [[ -z "$_shasums" ]]; then
    log_error "Node.js: could not reach ${_base}/SHASUMS256.txt (network or bad DOTFILES_NODE_MAJOR=${_major}?)"
    exit 1
fi
_file="$(printf '%s\n' "$_shasums" | grep -oE "node-v[0-9]+\.[0-9]+\.[0-9]+-${_platform}\.tar\.gz" | head -1 || true)"
if [[ -z "$_file" ]]; then
    log_error "Node.js: no ${_platform} tarball in latest-v${_major}.x"
    exit 1
fi
_sum="$(printf '%s\n' "$_shasums" | awk -v f="$_file" '$2==f{print $1}' | head -1 || true)"
_ver="$(printf '%s' "$_file" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || true)"
if [[ ! "$_sum" =~ ^[a-f0-9]{64}$ || -z "$_ver" ]]; then
    log_error "Node.js: could not resolve checksum/version for ${_file}"
    exit 1
fi

# The floor must be satisfiable by the pinned LTS line, or the pin is wrong: fail
# loudly (a too-low DOTFILES_NODE_MAJOR against a higher min_version) rather than
# install a node we already know is below the floor.
_verbare="${_ver#v}"
if [[ -n "$_min" ]] && ! _ver_ge "$_verbare" "$_min"; then
    log_error "Node.js: latest-v${_major}.x is ${_ver}, below the required floor ${_min} — raise DOTFILES_NODE_MAJOR to a line that meets it"
    exit 1
fi

_work="$(mktemp -d)"
trap 'rm -rf "$_work"' EXIT

log_info "Installing Node.js ${_ver} (${_platform}, sudo-free)${_min:+ (floor >=${_min})}..."
# download_file does portable checksum (sha256sum/shasum) + curl/wget + abort.
if ! download_file "${_base}/${_file}" "${_work}/node.tar.gz" "$_sum"; then
    log_error "Node.js: download/checksum verification failed"
    exit 1
fi

_dir="node-${_ver}-${_platform}"   # the tarball's top-level directory
mkdir -p "$_share" "$_bin"
rm -rf "${_share:?}/${_dir}"
tar -xzf "${_work}/node.tar.gz" -C "$_share"

# Symlink the user-facing binaries into ~/.local/bin. npm/npx inside the tarball
# are relative symlinks into lib/node_modules, so they resolve through our link.
for _b in node npm npx; do
    if [[ -e "${_share}/${_dir}/bin/${_b}" ]]; then
        ln -sf "${_share}/${_dir}/bin/${_b}" "${_bin}/${_b}"
    fi
done

log_success "Node.js installed: $("${_bin}/node" --version 2>/dev/null) (npm $("${_bin}/npm" --version 2>/dev/null))"
