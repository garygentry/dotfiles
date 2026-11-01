package dotfiles

import (
	"bytes"
	"testing"

	"github.com/garygentry/dotfiles/internal/module"
)

func TestPrintNotes(t *testing.T) {
	mods := []*module.Module{
		{Name: "zsh", Notes: []string{"exec zsh"}},
		{Name: "git"},
		{Name: "mise", Notes: []string{"pin it", "shims on PATH"}},
	}

	var all bytes.Buffer
	if err := printNotes(&all, mods, nil); err != nil {
		t.Fatal(err)
	}
	if want := "mise:\n  - pin it\n  - shims on PATH\n\nzsh:\n  - exec zsh\n"; all.String() != want {
		t.Errorf("all modules:\n%s\nwant:\n%s", all.String(), want)
	}

	var named bytes.Buffer
	if err := printNotes(&named, mods, []string{"git"}); err != nil {
		t.Fatal(err)
	}
	if want := "git: no notes\n"; named.String() != want {
		t.Errorf("named = %q, want %q", named.String(), want)
	}

	if err := printNotes(&bytes.Buffer{}, mods, []string{"nope"}); err == nil {
		t.Error("unknown module: want an error")
	}
}
