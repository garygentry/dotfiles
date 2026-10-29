package module

import (
	"fmt"
	"strings"
)

// AnswerSource names where a resolved prompt answer came from.
type AnswerSource string

const (
	AnswerFromConfig  AnswerSource = "config"  // modules.<module>.<prompt_key> in a config layer
	AnswerFromSaved   AnswerSource = "saved"   // an earlier interactive answer on this machine
	AnswerFromAsked   AnswerSource = "asked"   // asked on this run
	AnswerFromDefault AnswerSource = "default" // the module's declared default
	AnswerWouldAsk    AnswerSource = "ask"     // preview only: a real run would ask
	AnswerUnused      AnswerSource = "unused"  // depends_on not met: answer is empty
)

// ResolvedAnswer is one prompt's answer and where it came from.
type ResolvedAnswer struct {
	Key    string
	Value  string
	Source AnswerSource
}

// askFunc asks a prompt interactively, offering def as the default answer.
type askFunc func(p Prompt, def string) (string, error)

// resolvePrompts resolves each of mod's prompts to the first of:
//
//  1. config: modules.<module>.<prompt_key> from the composed config layers
//     (config.yml, profiles, host). Config always wins and is never asked.
//  2. a saved answer from an earlier interactive run on this machine (skipped
//     under --interactive, which re-asks with the saved value as the default).
//  3. asking, only when the run is interactive and depends_on, skip_if_file and
//     show_when let the prompt be shown.
//  4. the module's default.
//
// With ask == nil nothing is asked: a prompt that would be asked resolves to the
// value it would offer, with source AnswerWouldAsk (dry-run preview, and the
// silent verify before skipping an up-to-date module).
//
// Unattended runs behave as before this resolution existed, apart from steps 1
// and 2: depends_on is not applied, so a dependent prompt gets its default
// rather than an empty answer.
func resolvePrompts(cfg *RunConfig, mod *Module, ask askFunc) ([]ResolvedAnswer, error) {
	isExplicit := cfg.ExplicitModules != nil && cfg.ExplicitModules[mod.Name]
	settings := cfg.Config.Modules[mod.Name]

	resolved := make([]ResolvedAnswer, 0, len(mod.Prompts))
	values := make(map[string]string, len(mod.Prompts))
	set := func(p Prompt, v string, src AnswerSource) {
		values[p.Key] = v
		resolved = append(resolved, ResolvedAnswer{Key: p.Key, Value: v, Source: src})
	}

	for _, p := range mod.Prompts {
		if raw, ok := settings[p.Key]; ok {
			v, err := promptValueFromConfig(p, raw)
			if err != nil {
				return nil, fmt.Errorf("modules.%s.%s: %w", mod.Name, p.Key, err)
			}
			set(p, v, AnswerFromConfig)
			continue
		}

		saved, hasSaved := "", false
		if cfg.Answers != nil {
			saved, hasSaved = cfg.Answers.Get(mod.Name, p.Key)
		}
		fallback := func() {
			if hasSaved {
				set(p, saved, AnswerFromSaved)
			} else {
				set(p, p.Default, AnswerFromDefault)
			}
		}

		if cfg.Unattended {
			fallback()
			continue
		}

		// Skip prompts whose dependency condition is not met; use empty string
		// so the template can fall back to its own default (via the `default`
		// function). Only prompts resolved earlier in the same loop can be
		// referenced, so prompt order in module.yml matters.
		if p.DependsOn != nil && values[p.DependsOn.Key] != p.DependsOn.Value {
			set(p, "", AnswerUnused)
			continue
		}

		// Skip prompts made moot because a listed file already exists (e.g. an
		// existing SSH key makes the key-type prompt noise — install.sh reuses
		// the key regardless of the answer).
		if promptSkippedByFile(p, cfg.SysInfo) {
			fallback()
			if cfg.Verbose {
				cfg.UI.Debug(fmt.Sprintf("Skipping prompt %s.%s: a skip_if_file path already exists", mod.Name, p.Key))
			}
			continue
		}

		if hasSaved && !cfg.Interactive {
			set(p, saved, AnswerFromSaved)
			continue
		}

		if !shouldShowPrompt(p, cfg, mod, isExplicit) {
			fallback()
			// Log for transparency when not asking for auto-included modules
			if !isExplicit && cfg.Verbose {
				cfg.UI.Debug(fmt.Sprintf("Using %s for %s.%s: %s (auto-included dependency)", resolved[len(resolved)-1].Source, mod.Name, p.Key, values[p.Key]))
			}
			continue
		}

		def := p.Default
		if hasSaved {
			def = saved
		}
		if ask == nil {
			set(p, def, AnswerWouldAsk)
			continue
		}
		answer, err := ask(p, def)
		if err != nil {
			return nil, fmt.Errorf("prompt %q: %w", p.Key, err)
		}
		set(p, answer, AnswerFromAsked)
	}

	return resolved, nil
}

// PreviewPromptAnswers resolves mod's prompts without asking anything, for
// display (install --dry-run). A prompt a real run would ask is reported with
// source AnswerWouldAsk and the value it would offer as the default.
func PreviewPromptAnswers(cfg *RunConfig, mod *Module) ([]ResolvedAnswer, error) {
	return resolvePrompts(cfg, mod, nil)
}

// quietPromptAnswers resolves mod's prompts without asking anything, keyed by
// prompt key: the answers a run would use, short of a new interactive answer.
func quietPromptAnswers(cfg *RunConfig, mod *Module) (map[string]string, error) {
	resolved, err := resolvePrompts(cfg, mod, nil)
	if err != nil {
		return nil, err
	}
	return answerValues(resolved), nil
}

func answerValues(resolved []ResolvedAnswer) map[string]string {
	values := make(map[string]string, len(resolved))
	for _, r := range resolved {
		values[r.Key] = r.Value
	}
	return values
}

// promptValueFromConfig turns a config value pinned for prompt p into the
// answer string scripts see as DOTFILES_PROMPT_<KEY>. It must be a single
// value; a choice must be one of the prompt's options, and a confirm a yes/no
// (normalized to "true"/"false", as an interactive confirm answers).
func promptValueFromConfig(p Prompt, raw any) (string, error) {
	switch raw.(type) {
	case nil:
		return "", fmt.Errorf("pins prompt %q but has no value", p.Key)
	case []any, map[string]any:
		return "", fmt.Errorf("pins prompt %q and must be a single value, not a list or map", p.Key)
	}
	v := scalarToStr(raw)

	switch p.Type {
	case "choice":
		if len(p.Options) == 0 {
			return v, nil
		}
		for _, opt := range p.Options {
			if v == opt {
				return v, nil
			}
		}
		return "", fmt.Errorf("%q is not a valid answer for prompt %q (one of: %s)", v, p.Key, strings.Join(p.Options, ", "))
	case "confirm":
		switch strings.ToLower(v) {
		case "true", "yes", "y":
			return "true", nil
		case "false", "no", "n":
			return "false", nil
		}
		return "", fmt.Errorf("%q is not a valid answer for yes/no prompt %q (true/false)", v, p.Key)
	default:
		return v, nil
	}
}

// ValidatePromptConfig checks every config value that pins one of mod's
// prompts (settings is the composed modules.<module> map) and returns one
// message per invalid value.
func ValidatePromptConfig(mod *Module, settings map[string]any) []string {
	var errs []string
	for _, p := range mod.Prompts {
		raw, ok := settings[p.Key]
		if !ok {
			continue
		}
		if _, err := promptValueFromConfig(p, raw); err != nil {
			errs = append(errs, fmt.Sprintf("modules.%s.%s: %v", mod.Name, p.Key, err))
		}
	}
	return errs
}
