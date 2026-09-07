package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Layered config merge (ADR 0028 §D10 / client-deploy-model Phase 5).
//
// Config *values* layer with precedence estate-default → baseline → overlays
// (declared order) → host, no lock: a later layer always wins, so every setting is
// host-overridable. estate-default is the config.yml chain (base ← content overlay,
// see Load); baseline+overlays are the `config:` blocks on profile files, folded in
// resolved `extends` order (LoadProfileResolved); host is a host-owned file the
// reconcile only ever reads (LoadHostConfig). The whole fold operates on the
// free-form settings bag (Config.Modules) — identity (user/secrets) is out of the
// layered path by design (fenced to modules.*).
//
// Merge semantics (signed off, phase-5-design.md §D-b):
//   - scalars   — presence-wins: a key present in a later layer overrides, even to a
//                 zero value ("" / false / 0), so a host can turn a standardized
//                 setting off (diverges from mergeConfig's non-empty-wins);
//   - maps      — recurse: later keys override/add, earlier-only keys survive, so a
//                 host overrides one nested setting without redefining the subtree;
//   - lists     — replace: a later layer's list wholly replaces the earlier list
//                 (not append), so a lower-layer entry can be removed;
//   - type      — last-wins replacement: a later scalar/list/map simply replaces a
//                 mismatched earlier value;
//   - null      — a value that sets null, NOT a delete-tombstone (no lock, no removal
//                 of a key a lower layer set).

// layerConfig is the shape of a single config layer's contribution — the `config:`
// block on a profile file, and the whole of a host-owned config file. Only `modules:`
// is accepted: with the KnownFields(true) decoders below, an identity field
// (`user:`/`secrets:`) in a profile `config:` block or a host file is REJECTED,
// structurally fencing the layered/host path to modules.* (§D-c).
type layerConfig struct {
	Modules map[string]map[string]any `yaml:"modules"`
}

// deepMergeMap merges src into dst in place, last-wins with recursion into nested
// maps (see semantics above). src values are cloned on assignment, so the result
// never aliases an input layer. Iterating src's keys and assigning is exactly
// presence-wins; the only non-replacement case is two maps at the same key, which
// recurse.
func deepMergeMap(dst, src map[string]any) {
	for k, sv := range src {
		if dv, ok := dst[k]; ok {
			if dm, ok := dv.(map[string]any); ok {
				if sm, ok := sv.(map[string]any); ok {
					deepMergeMap(dm, sm)
					continue
				}
			}
		}
		dst[k] = cloneValue(sv)
	}
}

// cloneValue deep-copies a decoded YAML value so a merged result owns its maps and
// slices outright. Scalars are immutable and returned as-is. yaml.v3 decodes mappings
// into map[string]any and sequences into []any, so those two cases cover every
// composite a config layer can hold.
func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, vv := range t {
			m[k] = cloneValue(vv)
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i, vv := range t {
			s[i] = cloneValue(vv)
		}
		return s
	default:
		return v
	}
}

// cloneModules deep-copies the module settings bag so ComposeModules can fold layers
// into it without mutating any caller's config.
func cloneModules(m map[string]map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(m))
	for mod, settings := range m {
		cs := make(map[string]any, len(settings))
		for k, v := range settings {
			cs[k] = cloneValue(v)
		}
		out[mod] = cs
	}
	return out
}

// ComposeModules folds the layered config bags in precedence order and returns the
// merged result, leaving every input untouched. base is the estate-default
// (config.yml chain); layers are the baseline+overlay `config:` blocks in resolved
// extends order; host is the host-owned layer (nil when absent), applied last so it
// wins over everything.
func ComposeModules(base map[string]map[string]any, layers []map[string]map[string]any, host map[string]map[string]any) map[string]map[string]any {
	result := cloneModules(base)
	apply := func(layer map[string]map[string]any) {
		for mod, settings := range layer {
			if result[mod] == nil {
				result[mod] = make(map[string]any, len(settings))
			}
			deepMergeMap(result[mod], settings)
		}
	}
	for _, l := range layers {
		apply(l)
	}
	apply(host)
	return result
}

// LoadHostConfig reads the host-owned config layer — the settings a host overrides or
// adds locally, which the reconcile only ever reads (ADR 0028 §D10 / §D-c). The engine
// is generic: the path is configurable (--host-config / DOTFILES_HOST_CONFIG); the
// estate points it at /etc/gnet/local-config.yml. An empty path or an absent file
// means "no host overrides" (nil bag). A present-but-MALFORMED file — including one
// carrying a fenced-out identity field (user:/secrets:), rejected by KnownFields — is
// a HARD error: silently treating it as empty would drop the very overrides it exists
// to apply (T2 discipline, mirroring loadAdditionsManifest).
func LoadHostConfig(path string) (map[string]map[string]any, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // absent = no host overrides
		}
		return nil, fmt.Errorf("reading host config %s: %w", path, err)
	}

	var lc layerConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // reject typos and fenced-out identity fields (user:/secrets:)
	if err := dec.Decode(&lc); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parsing host config %s: %w", path, err)
	}
	if err := ErrIfSecondYAMLDoc(dec, fmt.Sprintf("host config %s", path)); err != nil {
		return nil, err
	}
	return lc.Modules, nil
}

// ErrIfSecondYAMLDoc returns an error when dec — a decoder that has already read one
// document — has a further document remaining. Host-owned files (LoadHostConfig here,
// loadAdditionsManifest in the prune command) are single-document by contract: a stray
// `---` would leave every override/protection after the first silently unapplied, the
// exact T2 violation both readers exist to prevent. Shared so the guard can never
// diverge between the two readers. A malformed second document is likewise a hard
// error (its non-EOF decode error trips the guard).
func ErrIfSecondYAMLDoc(dec *yaml.Decoder, what string) error {
	if err := dec.Decode(new(map[string]any)); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s must be a single YAML document", what)
	}
	return nil
}
