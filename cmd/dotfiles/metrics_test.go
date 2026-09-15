package dotfiles

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/garygentry/dotfiles/internal/ui"
)

func TestBoolToExit(t *testing.T) {
	if boolToExit(false) != 0 {
		t.Fatal("no error must be exit 0")
	}
	if boolToExit(true) != 1 {
		t.Fatal("an error must be exit 1")
	}
}

func TestEscapeLabel(t *testing.T) {
	got := escapeLabel(`a"b\c` + "\n" + "d")
	want := `a\"b\\c\nd`
	if got != want {
		t.Fatalf("escapeLabel: got %q want %q", got, want)
	}
}

func TestEnginePinSHA(t *testing.T) {
	// Empty dir → unknown (early-return path before sys is detected).
	if got := enginePinSHA(""); got != "unknown" {
		t.Fatalf("empty dir: got %q want unknown", got)
	}
	// A non-git dir → unknown, never a crash.
	if got := enginePinSHA(t.TempDir()); got != "unknown" {
		t.Fatalf("non-git dir: got %q want unknown", got)
	}
	// A real git repo → the short HEAD sha.
	dir := t.TempDir()
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("commit", "-q", "--allow-empty", "-m", "x")
	got := enginePinSHA(dir)
	if got == "unknown" || len(got) < 4 {
		t.Fatalf("git repo: got %q, want a short sha", got)
	}
}

// The emitter writes a well-formed exposition file: every metric present, each
// # TYPE declared exactly once, the info metric carrying the pin/profile labels,
// and no leftover .tmp (atomic rename).
func TestWriteReconcileMetrics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gnet_reconcile.prom")
	u := ui.New(false)

	writeReconcileMetrics(u, path, reconcileMetrics{
		start:       time.Unix(1_700_000_000, 0),
		exitStatus:  1,
		driftCount:  3,
		pruneArmed:  true,
		dotfilesDir: "", // → pin_sha "unknown"
		profile:     "workstation",
	})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("output not written: %v", err)
	}
	body := string(raw)

	wantLines := []string{
		"gnet_reconcile_last_run_timestamp_seconds 1700000000",
		"gnet_reconcile_exit_status 1",
		"gnet_reconcile_drift_count 3",
		"gnet_reconcile_prune_armed 1",
		`gnet_reconcile_info{pin_sha="unknown",profile="workstation"} 1`,
	}
	for _, l := range wantLines {
		if !strings.Contains(body, l) {
			t.Errorf("missing line %q in:\n%s", l, body)
		}
	}

	// Each metric declares its # TYPE exactly once (a duplicate TYPE across the
	// textfile dir makes node_exporter drop the whole file).
	for _, m := range []string{
		"gnet_reconcile_last_run_timestamp_seconds",
		"gnet_reconcile_exit_status",
		"gnet_reconcile_drift_count",
		"gnet_reconcile_prune_armed",
		"gnet_reconcile_info",
	} {
		if n := strings.Count(body, "# TYPE "+m+" "); n != 1 {
			t.Errorf("metric %s declared TYPE %d times, want 1", m, n)
		}
	}

	// Atomic write leaves no temp file behind.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf(".tmp file should not survive a successful write: %v", err)
	}
}

// prune_armed 0 and an empty profile → "default"; the clean/opted-out shape.
func TestWriteReconcileMetricsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gnet_reconcile.prom")
	writeReconcileMetrics(ui.New(false), path, reconcileMetrics{
		start:      time.Unix(1, 0),
		exitStatus: 0,
		driftCount: 0,
		pruneArmed: false,
		profile:    "",
	})
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "gnet_reconcile_prune_armed 0") {
		t.Error("prune_armed should be 0 when not armed")
	}
	if !strings.Contains(string(body), `profile="default"`) {
		t.Error("empty profile should render as default")
	}
}

// Writing into a directory that does not exist fails softly — logged, no panic,
// and never a partial file at the target path.
func TestWriteReconcileMetricsMissingDirIsSoft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "gnet_reconcile.prom")
	writeReconcileMetrics(ui.New(false), path, reconcileMetrics{start: time.Now()})
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("no file should exist when the parent dir is missing: %v", err)
	}
}
