#!/usr/bin/env bash
# mise/install.sh - Install mise (sudo-free, pinned, checksum-verified) and the
# tools declared in config.
#
# Generic by design: the engine ships NO tool list and NO versions. Everything
# comes from modules.mise.* in config.yml (usually a content overlay):
#   version                 mise release to install (e.g. "2026.9.14"); unset = latest (warned)
#   tools                   map tool -> exact version, rendered to ~/.config/mise/conf.d/dotfiles.toml
#   settings                map rendered to [settings] in the same file
#   lockfile                optional mise.lock (relative to the content dir, else DOTFILES_DIR);
#                           when set, the engine owns ~/.config/mise/mise.lock and installs --locked
#   github_token_command    optional command printing a GitHub token, used for this run only
#   activate                "false" = shims only in interactive zsh (no `mise activate`)
#
# One owner per binary: this module manages ~/.local/bin/mise only. A mise that
# some other channel provides (Homebrew, a distro package) is used as-is and never
# replaced; if its version differs from the pin, a warning is logged.
set -euo pipefail

_home="${DOTFILES_HOME:-$HOME}"
_bin="${_home}/.local/bin"
_mise="${_bin}/mise"
_cfg_dir="${DOTFILES_XDG_CONFIG_HOME:-${_home}/.config}/mise"
_conf="${_cfg_dir}/conf.d/dotfiles.toml"
_want="${DOTFILES_SETTING_VERSION:-}"
_want="${_want#v}"

_mise_ver_of() { "$1" --version 2>/dev/null | awk '{print $1}' || true; }

# --- 1. The mise binary --------------------------------------------------------
_foreign="$(command -v mise 2>/dev/null || true)"
if [[ -n "$_foreign" && "$_foreign" != "$_mise" ]]; then
    _cur="$(_mise_ver_of "$_foreign")"
    log_info "mise provided outside dotfiles: ${_foreign} (${_cur:-unknown version}); using it as-is"
    if [[ -n "$_want" && "$_cur" != "$_want" ]]; then
        log_warn "mise ${_cur} differs from the pinned ${_want}; the pin only applies to ${_mise}"
    fi
    _mise="$_foreign"
elif [[ -x "$_mise" && ( -z "$_want" || "$(_mise_ver_of "$_mise")" == "$_want" ) ]]; then
    log_info "mise already installed: $(_mise_ver_of "$_mise") (${_mise})"
elif is_dry_run; then
    log_info "[dry-run] Would install mise ${_want:-latest} to ${_mise}"
else
    case "$(uname -s)" in
        Linux)  _os="linux" ;;
        Darwin) _os="macos" ;;
        *) log_error "mise: unsupported OS $(uname -s)"; exit 1 ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64)  _arch="x64" ;;
        arm64|aarch64) _arch="arm64" ;;
        *) log_error "mise: unsupported architecture $(uname -m)"; exit 1 ;;
    esac
    _target="${_os}-${_arch}"
    if [[ "$_os" == "linux" ]] && { [[ -e /lib/libc.musl-x86_64.so.1 ]] || [[ -e /lib/libc.musl-aarch64.so.1 ]] || ldd /bin/ls 2>&1 | grep -q musl; }; then
        _target="${_target}-musl"
    fi

    if [[ -n "$_want" ]]; then
        _tag="v${_want}"
    else
        # Latest WITHOUT the GitHub API (a fleet behind one NAT would exhaust 60 req/h):
        # follow the /releases/latest redirect and read the tag from the URL.
        _eff="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/jdx/mise/releases/latest 2>/dev/null || true)"
        _tag="$(printf '%s' "$_eff" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
        [[ -n "$_tag" ]] || { log_error "mise: could not resolve the latest release"; exit 1; }
        log_warn "mise: no modules.mise.version pin; installing latest (${_tag})"
    fi

    _asset="mise-${_tag}-${_target}.tar.gz"
    _rel="https://github.com/jdx/mise/releases/download/${_tag}"
    _work="$(mktemp -d)"
    trap 'rm -rf "$_work"' EXIT

    # Checksum is mandatory: mise publishes SHASUMS256.txt for every release.
    _sum="$(curl -fsSL "${_rel}/SHASUMS256.txt" 2>/dev/null \
        | awk -v f="$_asset" '{g=$2; sub(/^\*/,"",g); sub(/^\.\//,"",g)} g==f{print $1}' | head -1 || true)"
    if [[ ! "$_sum" =~ ^[a-f0-9]{64}$ ]]; then
        log_error "mise: no SHA-256 for ${_asset} in ${_rel}/SHASUMS256.txt; refusing to install unverified"
        exit 1
    fi

    log_info "Installing mise ${_tag#v} (${_target}, sudo-free)..."
    if ! download_file "${_rel}/${_asset}" "${_work}/mise.tar.gz" "$_sum"; then
        log_error "mise: download/checksum verification failed (${_asset})"
        exit 1
    fi
    tar -xzf "${_work}/mise.tar.gz" -C "$_work"
    if [[ ! -x "${_work}/mise/bin/mise" ]]; then
        log_error "mise: binary not found inside ${_asset}"
        exit 1
    fi
    # Refuse a wrong-arch or broken binary before it replaces a working one.
    if [[ -z "$(_mise_ver_of "${_work}/mise/bin/mise")" ]]; then
        log_error "mise: downloaded binary does not run on this host (${_target})"
        exit 1
    fi
    mkdir -p "$_bin"
    # Atomic replace: shims are symlinks to this path, so it must never be half-written.
    install -m 0755 "${_work}/mise/bin/mise" "${_mise}.new"
    mv -f "${_mise}.new" "$_mise"
    log_success "mise $(_mise_ver_of "$_mise") installed to ${_mise}"
fi

# --- 2. Declared tools and settings --------------------------------------------
if is_dry_run; then
    log_info "[dry-run] Would render ${_conf} from modules.mise.tools/settings"
else
    mkdir -p "$(dirname "$_conf")"
    render_template "${DOTFILES_MODULE_DIR}/dotfiles.toml.tmpl" "$_conf"
fi

# The declared tools, as tool@version args, read back from the rendered [tools]
# table (this module renders every entry as "name" = "version").
_tools=()
if [[ -f "$_conf" ]]; then
    while IFS= read -r _t; do
        [[ -n "$_t" ]] && _tools+=("$_t")
    done < <(awk '
        /^\[/ { in_tools = ($0 == "[tools]"); next }
        in_tools && /^"/ {
            if (match($0, /^"[^"]+" = "[^"]*"$/)) {
                split($0, kv, /" = "/)
                name = substr(kv[1], 2); ver = kv[2]; sub(/"$/, "", ver)
                print name "@" ver
            }
        }' "$_conf")
fi

# Optional lockfile: the engine owns ~/.config/mise/mise.lock when one is configured.
_locked=()
if [[ -n "${DOTFILES_SETTING_LOCKFILE:-}" ]]; then
    _lock_src="$DOTFILES_SETTING_LOCKFILE"
    if [[ "$_lock_src" != /* ]]; then
        _lock_src="${DOTFILES_CONTENT_DIR:-${DOTFILES_DIR}}/${_lock_src}"
    fi
    if [[ ! -f "$_lock_src" ]]; then
        log_error "mise: modules.mise.lockfile not found: ${_lock_src}"
        exit 1
    fi
    if is_dry_run; then
        log_info "[dry-run] Would install lockfile ${_lock_src} -> ${_cfg_dir}/mise.lock"
    else
        command cp -f "$_lock_src" "${_cfg_dir}/mise.lock"
    fi
    _locked=(--locked)
fi

if [[ ${#_tools[@]} -eq 0 ]]; then
    log_info "mise: no tools declared (modules.mise.tools); nothing to install"
elif is_dry_run; then
    log_info "[dry-run] Would run: mise install ${_locked[*]:-} ${_tools[*]}"
else
    # Token for this invocation only; never persisted.
    _token=""
    if [[ -n "${DOTFILES_SETTING_GITHUB_TOKEN_COMMAND:-}" ]]; then
        _token="$(bash -c "$DOTFILES_SETTING_GITHUB_TOKEN_COMMAND" 2>/dev/null | head -1 || true)"
        [[ -n "$_token" ]] || log_warn "mise: github_token_command produced no token; continuing anonymously"
    fi
    log_info "mise: installing ${#_tools[@]} declared tool(s)${_locked:+ (locked)}..."
    # ${arr[@]+...}: an empty array is "unbound" under set -u in bash 3.2 (macOS).
    # Scoped to the declared tools: a full `mise install --locked` would fail on a
    # user's own global tools that are absent from the managed lockfile.
    if ! MISE_GITHUB_TOKEN="${_token:-${MISE_GITHUB_TOKEN:-}}" MISE_YES=1 \
            "$_mise" install ${_locked[@]+"${_locked[@]}"} "${_tools[@]}"; then
        log_error "mise: tool install failed${_locked:+ (a --locked failure usually means the lockfile lacks an entry: re-run \`mise lock --global\`)}"
        exit 1
    fi
    "$_mise" reshim
    log_success "mise: ${#_tools[@]} tool(s) installed"
fi

# --- 3. Shims on PATH for every shell ------------------------------------------
# Shims resolve per invocation, so they work where rc-file activation never runs:
# `ssh host cmd`, `bash -lc`, scripts, agent tool shells. They must win over stale
# copies in ~/.local/bin. The zsh module's own ~/.zshenv PATH block also puts the
# shims first when they exist, so the order of the two blocks doesn't matter
# (module order follows dependency levels, not priority alone).
# shellcheck disable=SC2016  # kept literal: each shell expands it at startup
_shims='${XDG_DATA_HOME:-$HOME/.local/share}/mise/shims'
_activate_line=""
if [[ "${DOTFILES_SETTING_ACTIVATE:-true}" == "false" ]]; then
    _activate_line=$'\nexport DOTFILES_MISE_ACTIVATE=0   # modules.mise.activate: false (shims only)'
fi

_zshenv="${_home}/.zshenv"
upsert_managed_block "$_zshenv" "mise" "typeset -U path
path=(\"${_shims}\" \$path)${_activate_line}"

if is_macos; then
    _login_profile="${_home}/.bash_profile"
else
    _login_profile="${_home}/.profile"
fi
# POSIX sh: ~/.profile is also read by dash for `sh -l`.
upsert_managed_block "$_login_profile" "mise" "case \":\$PATH:\" in
    *\":${_shims}:\"*) ;;
    *) PATH=\"${_shims}:\$PATH\"; export PATH ;;
esac"
# Interactive non-login bash (e.g. most Linux terminal emulators) reads only ~/.bashrc.
upsert_managed_block "${_home}/.bashrc" "mise" "case \":\$PATH:\" in
    *\":${_shims}:\"*) ;;
    *) PATH=\"${_shims}:\$PATH\"; export PATH ;;
esac"
