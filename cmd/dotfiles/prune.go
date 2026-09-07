package dotfiles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

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
