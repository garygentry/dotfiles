package module

import (
	"strings"
	"testing"

	"github.com/garygentry/dotfiles/internal/state"
)

// recordingUI answers each choice prompt from a queue and records the default
// it was offered.
type recordingUI struct {
	testUI
	answers  []string
	asked    []string
	defaults []string
}

func (r *recordingUI) PromptChoice(msg string, _ []string, def string) (string, error) {
	r.asked = append(r.asked, msg)
	r.defaults = append(r.defaults, def)
	if len(r.answers) == 0 {
		return def, nil
	}
	a := r.answers[0]
	r.answers = r.answers[1:]
	return a, nil
}

func zshPromptModule() *Module {
	return &Module{
		Name: "zsh",
		Prompts: []Prompt{
			{Key: "zsh_framework", Message: "Zsh plugin framework", Default: "zinit", Type: "choice", Options: []string{"zinit", "ohmyzsh"}, ShowWhen: "explicit_install"},
			{Key: "zsh_omz_plugins", Message: "Oh My Zsh plugin preset", Default: "standard", Type: "choice", Options: []string{"minimal", "standard", "full"}, ShowWhen: "explicit_install", DependsOn: &PromptDependency{Key: "zsh_framework", Value: "ohmyzsh"}},
		},
	}
}

// interactiveCfg is an interactive run with zsh explicitly selected and an
// empty answers store in a temp state dir.
func interactiveCfg(t *testing.T, ui RunnerUI) *RunConfig {
	t.Helper()
	cfg := newTestRunConfig(t)
	cfg.Unattended = false
	cfg.UI = ui
	cfg.ExplicitModules = map[string]bool{"zsh": true}
	cfg.Answers = state.NewAnswers(cfg.State.Dir)
	return cfg
}

func TestConfigPinSuppressesPromptAndIsExported(t *testing.T) {
	ui := &recordingUI{answers: []string{"zinit"}}
	cfg := interactiveCfg(t, ui)
	cfg.Config.Modules["zsh"] = map[string]any{"zsh_framework": "ohmyzsh"}
	mod := zshPromptModule()

	answers, err := handlePrompts(cfg, mod)
	if err != nil {
		t.Fatalf("handlePrompts: %v", err)
	}
	// Only the dependent prompt (now applicable) is asked; the pinned one is not.
	if len(ui.asked) != 1 || ui.asked[0] != "Oh My Zsh plugin preset" {
		t.Errorf("asked = %v, want only the OMZ preset", ui.asked)
	}
	if answers["zsh_framework"] != "ohmyzsh" {
		t.Errorf("zsh_framework = %q, want ohmyzsh (config)", answers["zsh_framework"])
	}
	env := buildEnvVars(cfg, mod, answers)
	if env["DOTFILES_PROMPT_ZSH_FRAMEWORK"] != "ohmyzsh" {
		t.Errorf("DOTFILES_PROMPT_ZSH_FRAMEWORK = %q, want ohmyzsh", env["DOTFILES_PROMPT_ZSH_FRAMEWORK"])
	}
	// A config answer is not a saved answer.
	if _, ok := cfg.Answers.Get("zsh", "zsh_framework"); ok {
		t.Error("config-pinned answer must not be saved")
	}
}

func TestConfigPinAppliesUnattended(t *testing.T) {
	cfg := newTestRunConfig(t) // unattended
	cfg.Config.Modules["zsh"] = map[string]any{"zsh_framework": "ohmyzsh"}
	answers, err := handlePrompts(cfg, zshPromptModule())
	if err != nil {
		t.Fatalf("handlePrompts: %v", err)
	}
	if answers["zsh_framework"] != "ohmyzsh" {
		t.Errorf("zsh_framework = %q, want ohmyzsh (config beats default)", answers["zsh_framework"])
	}
}

func TestInteractiveAnswerPersistsForUnattendedRerun(t *testing.T) {
	ui := &recordingUI{answers: []string{"ohmyzsh", "full"}}
	cfg := interactiveCfg(t, ui)
	if _, err := handlePrompts(cfg, zshPromptModule()); err != nil {
		t.Fatalf("handlePrompts: %v", err)
	}

	// A later unattended run loads the saved answers from disk.
	loaded, err := state.LoadAnswers(cfg.State.Dir)
	if err != nil {
		t.Fatalf("LoadAnswers: %v", err)
	}
	cfg2 := newTestRunConfig(t)
	cfg2.Answers = loaded
	answers, err := handlePrompts(cfg2, zshPromptModule())
	if err != nil {
		t.Fatalf("handlePrompts: %v", err)
	}
	if answers["zsh_framework"] != "ohmyzsh" || answers["zsh_omz_plugins"] != "full" {
		t.Errorf("answers = %v, want the saved ohmyzsh/full, not defaults", answers)
	}
}

func TestSavedAnswerIsNotAskedAgain(t *testing.T) {
	ui := &recordingUI{}
	cfg := interactiveCfg(t, ui)
	cfg.Answers.Set("zsh", "zsh_framework", "zinit")

	answers, err := handlePrompts(cfg, zshPromptModule())
	if err != nil {
		t.Fatalf("handlePrompts: %v", err)
	}
	if len(ui.asked) != 0 {
		t.Errorf("asked = %v, want nothing (saved answer)", ui.asked)
	}
	if answers["zsh_framework"] != "zinit" {
		t.Errorf("zsh_framework = %q, want zinit", answers["zsh_framework"])
	}
}

func TestInteractiveFlagReasksWithSavedAsDefault(t *testing.T) {
	ui := &recordingUI{answers: []string{"zinit"}}
	cfg := interactiveCfg(t, ui)
	cfg.Interactive = true
	cfg.Answers.Set("zsh", "zsh_framework", "ohmyzsh")
	cfg.Answers.Set("zsh", "zsh_omz_plugins", "full")

	answers, err := handlePrompts(cfg, zshPromptModule())
	if err != nil {
		t.Fatalf("handlePrompts: %v", err)
	}
	if len(ui.asked) != 1 || ui.defaults[0] != "ohmyzsh" {
		t.Errorf("asked = %v defaults = %v, want the framework asked with ohmyzsh offered", ui.asked, ui.defaults)
	}
	if v, _ := cfg.Answers.Get("zsh", "zsh_framework"); v != "zinit" {
		t.Errorf("saved zsh_framework = %q, want the new answer zinit", v)
	}
	if answers["zsh_omz_plugins"] != "" {
		t.Errorf("zsh_omz_plugins = %q, want empty (dependency no longer met)", answers["zsh_omz_plugins"])
	}
}

func TestInteractiveFlagNeverAsksConfigPin(t *testing.T) {
	ui := &recordingUI{}
	cfg := interactiveCfg(t, ui)
	cfg.Interactive = true
	cfg.Config.Modules["zsh"] = map[string]any{"zsh_framework": "zinit"}
	if _, err := handlePrompts(cfg, zshPromptModule()); err != nil {
		t.Fatalf("handlePrompts: %v", err)
	}
	if len(ui.asked) != 0 {
		t.Errorf("asked = %v, want nothing (config wins)", ui.asked)
	}
}

func TestPreviewPromptAnswersSources(t *testing.T) {
	cfg := interactiveCfg(t, &recordingUI{})
	cfg.Config.Modules["zsh"] = map[string]any{"zsh_framework": "ohmyzsh"}
	mod := zshPromptModule()
	mod.Prompts = append(mod.Prompts,
		Prompt{Key: "zsh_prompt", Message: "Prompt theme", Default: "starship", Type: "choice", Options: []string{"starship", "agnoster"}, ShowWhen: "explicit_install"},
		Prompt{Key: "zsh_extra", Message: "Extra", Default: "x", Type: "input", ShowWhen: "explicit_install"})
	cfg.Answers.Set("zsh", "zsh_prompt", "agnoster")

	got, err := PreviewPromptAnswers(cfg, mod)
	if err != nil {
		t.Fatalf("PreviewPromptAnswers: %v", err)
	}
	want := []ResolvedAnswer{
		{"zsh_framework", "ohmyzsh", AnswerFromConfig},
		{"zsh_omz_plugins", "standard", AnswerWouldAsk},
		{"zsh_prompt", "agnoster", AnswerFromSaved},
		{"zsh_extra", "x", AnswerWouldAsk},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	// Unattended: nothing would be asked; unanswered prompts take the default.
	cfg.Unattended = true
	got, err = PreviewPromptAnswers(cfg, mod)
	if err != nil {
		t.Fatalf("PreviewPromptAnswers: %v", err)
	}
	if got[3].Source != AnswerFromDefault {
		t.Errorf("unattended zsh_extra source = %s, want default", got[3].Source)
	}
}

func TestPromptValueFromConfig(t *testing.T) {
	choice := Prompt{Key: "zsh_framework", Type: "choice", Options: []string{"zinit", "ohmyzsh"}}
	confirm := Prompt{Key: "enable", Type: "confirm"}
	input := Prompt{Key: "name", Type: "input"}
	tests := []struct {
		name    string
		p       Prompt
		raw     any
		want    string
		wantErr string
	}{
		{"valid choice", choice, "ohmyzsh", "ohmyzsh", ""},
		{"invalid choice", choice, "zim", "", "one of: zinit, ohmyzsh"},
		{"confirm bool", confirm, true, "true", ""},
		{"confirm yes", confirm, "Yes", "true", ""},
		{"confirm no", confirm, "no", "false", ""},
		{"confirm junk", confirm, "maybe", "", "yes/no"},
		{"input number", input, 42, "42", ""},
		{"null", input, nil, "", "no value"},
		{"list", input, []any{"a"}, "", "single value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := promptValueFromConfig(tt.p, tt.raw)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("= %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestInvalidConfigPinFailsPrompts(t *testing.T) {
	cfg := newTestRunConfig(t)
	cfg.Config.Modules["zsh"] = map[string]any{"zsh_framework": "zim"}
	if _, err := handlePrompts(cfg, zshPromptModule()); err == nil || !strings.Contains(err.Error(), "modules.zsh.zsh_framework") {
		t.Errorf("err = %v, want one naming modules.zsh.zsh_framework", err)
	}
	errs := ValidatePromptConfig(zshPromptModule(), cfg.Config.Modules["zsh"])
	if len(errs) != 1 || !strings.Contains(errs[0], `"zim"`) {
		t.Errorf("ValidatePromptConfig = %v", errs)
	}
	// Settings that are not prompt keys are not checked.
	if errs := ValidatePromptConfig(zshPromptModule(), map[string]any{"startup_budget_ms": "fast"}); len(errs) != 0 {
		t.Errorf("non-prompt setting flagged: %v", errs)
	}
}

// --interactive reviews an up-to-date module: an unchanged answer keeps it
// skipped, a changed one re-runs it.
func TestInteractiveFlagOnUpToDateModule(t *testing.T) {
	for _, tc := range []struct {
		answer   string
		wantSkip bool
	}{
		{"ohmyzsh", true},
		{"zinit", false},
	} {
		t.Run(tc.answer, func(t *testing.T) {
			ui := &recordingUI{answers: []string{tc.answer, "standard"}}
			cfg := interactiveCfg(t, ui)
			cfg.Interactive = true
			cfg.Answers.Set("zsh", "zsh_framework", "ohmyzsh")
			cfg.Answers.Set("zsh", "zsh_omz_plugins", "standard")

			mod := zshPromptModule()
			mod.Version = "1.0.0"
			mod.Dir = t.TempDir()
			checksum, _ := ComputeModuleChecksum(mod)
			if err := cfg.State.Set(&state.ModuleState{
				Name: mod.Name, Version: mod.Version, Status: "installed",
				Checksum: checksum, ConfigHash: ComputeConfigHash(mod, cfg.Config),
			}); err != nil {
				t.Fatal(err)
			}

			res := runModule(cfg, mod)
			if res.Skipped != tc.wantSkip {
				t.Errorf("Skipped = %v, want %v (err %v)", res.Skipped, tc.wantSkip, res.Error)
			}
			if len(ui.asked) == 0 || ui.defaults[0] != "ohmyzsh" {
				t.Errorf("asked = %v defaults = %v, want framework asked once with ohmyzsh offered", ui.asked, ui.defaults)
			}
			if !tc.wantSkip && len(ui.asked) > 2 {
				t.Errorf("asked = %v: prompts must not be asked twice when the module re-runs", ui.asked)
			}
		})
	}
}
