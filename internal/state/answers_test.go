package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAnswersAbsentIsEmpty(t *testing.T) {
	a, err := LoadAnswers(t.TempDir())
	if err != nil {
		t.Fatalf("LoadAnswers: %v", err)
	}
	if _, ok := a.Get("zsh", "zsh_framework"); ok {
		t.Error("expected no answer in an absent file")
	}
}

func TestAnswersSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := NewAnswers(dir)
	a.Set("zsh", "zsh_framework", "ohmyzsh")
	a.Set("ssh", "ssh_key_type", "rsa")
	if err := a.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := LoadAnswers(dir)
	if err != nil {
		t.Fatalf("LoadAnswers: %v", err)
	}
	for _, tc := range []struct{ mod, key, want string }{
		{"zsh", "zsh_framework", "ohmyzsh"},
		{"ssh", "ssh_key_type", "rsa"},
	} {
		if v, ok := got.Get(tc.mod, tc.key); !ok || v != tc.want {
			t.Errorf("%s.%s = %q (present %v), want %q", tc.mod, tc.key, v, ok, tc.want)
		}
	}
	if got.Path() != filepath.Join(dir, "answers", "answers.json") {
		t.Errorf("Path = %q", got.Path())
	}
}

func TestLoadAnswersMalformedIsError(t *testing.T) {
	dir := t.TempDir()
	path := AnswersPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAnswers(dir); err == nil {
		t.Error("expected an error for a malformed answers file")
	}
}

// The answers file lives in a subdirectory so Store.GetAll, which parses every
// *.json directly in the state dir as a module state, never sees it.
func TestGetAllIgnoresAnswersFile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Set(&ModuleState{Name: "zsh", Status: "installed"}); err != nil {
		t.Fatal(err)
	}
	a := NewAnswers(dir)
	a.Set("zsh", "zsh_framework", "ohmyzsh")
	if err := a.Save(); err != nil {
		t.Fatal(err)
	}

	states, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(states) != 1 || states[0].Name != "zsh" {
		t.Errorf("GetAll = %v, want only the zsh state", states)
	}
}
