package dotfiles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/module"
	"github.com/garygentry/dotfiles/internal/state"
	"gopkg.in/yaml.v3"
)

// Prune reconcile (ADR 0028 §D6 / client-deploy-model Phase 4, D8).
//
// A full profile reconcile is declarative: a module the engine recorded
// installing but that is absent from the resolved effective set (profile +
// deps) AND from the host-owned additions manifest is removed. Prune is gated
// hard — it runs only in a no-args profile reconcile, only when the host has
// opted in (--allow-prune, recorded per host), and reuses the recorded-operation
// rollback so it can only ever undo what the engine itself did.

// prune-related flags on `install`.
var (
	allowPrune        bool
	noPrune           bool
	additionsManifest string
)

// additionsFile is the host-owned "protect" manifest schema: module names that
// must survive prune even when absent from the profile.
type additionsFile struct {
	Modules []string `yaml:"modules"`
}

// loadAdditionsManifest reads the host-owned additions manifest — the modules to
// protect from prune (ADR 0028 §D6). The engine is generic: the path is
// configurable (--additions-manifest / DOTFILES_ADDITIONS_MANIFEST); the estate
// points it at /etc/gnet/local-additions.yml. An empty path or an absent file
// means "no additions" (empty allowlist, nothing protected). A present-but-
// MALFORMED file is a HARD error: silently treating it as empty would prune the
// very modules it exists to protect (T2 discipline).
func loadAdditionsManifest(path string) (map[string]bool, error) {
	protect := map[string]bool{}
	if path == "" {
		return protect, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return protect, nil // absent = empty allowlist
		}
		return nil, fmt.Errorf("reading additions manifest %s: %w", path, err)
	}

	var af additionsFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // reject typos (e.g. `module:`), matching the engine's config decoder
	if err := dec.Decode(&af); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parsing additions manifest %s: %w", path, err)
	}
	// Single document only (shared T2 guard): a stray `---` would silently drop the
	// protections in every document after the first, wrongly pruning those modules.
	if err := config.ErrIfSecondYAMLDoc(dec, fmt.Sprintf("additions manifest %s", path)); err != nil {
		return nil, err
	}
	for _, m := range af.Modules {
		if m != "" {
			protect[m] = true
		}
	}
	return protect, nil
}

// computePruneCandidates returns installed modules (per the state store) that are
// absent from both the effective set and the protect set, sorted for stable
// output. The state store is the source of truth: prune never reasons about
// modules the engine did not install.
func computePruneCandidates(store *state.Store, effective, protect map[string]bool) ([]string, error) {
	all, err := store.GetAll()
	if err != nil {
		return nil, err
	}
	var cands []string
	for _, ms := range all {
		if ms.Status != "installed" {
			continue
		}
		if effective[ms.Name] || protect[ms.Name] {
			continue
		}
		cands = append(cands, ms.Name)
	}
	sort.Strings(cands)
	return cands, nil
}

// pruneOptInPath is the per-host opt-in marker inside the state dir. It lives
// with the state the reconcile already owns and is user-writable (reconcile runs
// as the user; /etc/gnet is root-owned). This makes the opt-in per-(user,host),
// which equals per-host for today's single-operator hosts.
func pruneOptInPath(dotfilesDir string) string {
	return filepath.Join(dotfilesDir, ".state", ".prune-opt-in")
}

func hasPruneOptIn(dotfilesDir string) bool {
	_, err := os.Stat(pruneOptInPath(dotfilesDir))
	return err == nil
}

func recordPruneOptIn(dotfilesDir string) error {
	p := pruneOptInPath(dotfilesDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte("opted-in\n"), 0o644)
}

// pathOwners maps every path an installed module other than `except` deploys
// to that module's name, from its file states and recorded file_deploy ops.
// Prune runs after the reconcile's installs, so a path a pruned module once
// deployed may now belong to a module that is still in the profile.
func pathOwners(store *state.Store, except string) (map[string]string, error) {
	all, err := store.GetAll()
	if err != nil {
		return nil, err
	}
	owners := make(map[string]string)
	for _, ms := range all {
		if ms.Name == except || ms.Status != "installed" {
			continue
		}
		for _, fs := range ms.FileStates {
			owners[filepath.Clean(fs.Dest)] = ms.Name
		}
		for _, op := range ms.Operations {
			if op.Type == "file_deploy" {
				owners[filepath.Clean(op.Path)] = ms.Name
			}
		}
	}
	return owners, nil
}

// pruneKeepReason reports why prune must leave a file_deploy op's path in
// place, or "" when the path is still what ms deployed and rolling it back is
// safe: another installed module now deploys it, the symlink no longer points
// at the recorded source, or the file's content differs from what was deployed.
func pruneKeepReason(op state.Operation, ms *state.ModuleState, owners map[string]string) string {
	path := filepath.Clean(op.Path)
	if owner, ok := owners[path]; ok {
		return "now deployed by " + owner
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return "" // absent: rollback already treats that as removed
	}
	if op.Action == "symlinked" {
		if fi.Mode()&os.ModeSymlink == 0 {
			return "no longer a symlink"
		}
		if src := op.Metadata["source"]; src != "" {
			if target, err := os.Readlink(path); err != nil || filepath.Clean(target) != filepath.Clean(src) {
				return "symlink points elsewhere"
			}
		}
		return ""
	}
	var deployed string
	for _, fs := range ms.FileStates {
		if filepath.Clean(fs.Dest) == path {
			deployed = fs.DeployedHash
		}
	}
	if deployed == "" || !fi.Mode().IsRegular() {
		return ""
	}
	if current, err := module.ComputeFileHash(path); err != nil || current != deployed {
		return "changed since it was deployed"
	}
	return ""
}

// leftInstalled reports whether ms ran an install script or installed a
// package. Prune can't undo either, so their software stays on the machine.
func leftInstalled(ms *state.ModuleState) bool {
	for _, op := range ms.Operations {
		if op.Type == "script_run" || op.Type == "package_install" {
			return true
		}
	}
	return false
}
