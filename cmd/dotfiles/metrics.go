package dotfiles

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/garygentry/dotfiles/internal/ui"
)

// reconcileMetrics is the end-of-run snapshot the observability contract (Phase 7
// / D13) publishes as a node_exporter textfile. It is populated across the
// reconcile and read once by the deferred emitter, so every field must be safe to
// read even when the run returned early (zero values are meaningful).
type reconcileMetrics struct {
	start       time.Time // run start; emitted as last_run_timestamp
	exitStatus  int       // 0 success, 1 the reconcile returned an error
	driftCount  int       // modules that would prune (state ∖ plan ∖ manifest)
	pruneArmed  bool      // this run computed drift AND the host opted into prune
	dotfilesDir string    // engine checkout dir, for the pin_sha git read
	profile     string    // effective profile name (informational label)
}

// boolToExit maps a run error into the node_exporter exit_status value, mirroring
// main.go's os.Exit(1)-on-error contract.
func boolToExit(failed bool) int {
	if failed {
		return 1
	}
	return 0
}

// enginePinSHA returns the short git SHA the engine checkout is currently at — the
// runtime truth of the Phase 3 pin (DOTFILES_REF is bootstrap-only and invisible
// here). Best-effort: any failure yields "unknown" rather than aborting emission,
// because a missing pin label must never cost us the exit_status/drift signal.
func enginePinSHA(dir string) string {
	if dir == "" {
		return "unknown"
	}
	// A timeout so a wedged/hung mount under dir can't block the exit-time defer
	// forever. rev-parse is local + non-interactive, so 5s is generous.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	sha := strings.TrimSpace(string(out))
	if sha == "" {
		return "unknown"
	}
	return sha
}

// writeReconcileMetrics renders the reconcile metrics to a node_exporter textfile
// at path, written atomically (tmp + rename) so a concurrent scrape never sees a
// torn file. The engine stays generic: the path is supplied by the estate via
// --metrics-textfile / DOTFILES_METRICS_TEXTFILE (empty = feature off, handled by
// the caller). Emission failure is logged, never fatal — publishing the run's
// health must not change the run's own outcome.
func writeReconcileMetrics(u *ui.UI, path string, m reconcileMetrics) {
	armed := 0
	if m.pruneArmed {
		armed = 1
	}
	// escapeLabel keeps a stray value from breaking the exposition format; the SHA
	// and profile name are tame, but be defensive.
	pin := escapeLabel(enginePinSHA(m.dotfilesDir))
	prof := escapeLabel(m.profile)
	if prof == "" {
		prof = "default"
	}

	// Each # TYPE is declared exactly once. gnet_reconcile.prom is the only file
	// declaring these metrics in the textfile dir, so there is no cross-file TYPE
	// collision (which would make node_exporter drop the whole file).
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP gnet_reconcile_last_run_timestamp_seconds Unix time of the last reconcile run.\n")
	fmt.Fprintf(&b, "# TYPE gnet_reconcile_last_run_timestamp_seconds gauge\n")
	fmt.Fprintf(&b, "gnet_reconcile_last_run_timestamp_seconds %d\n", m.start.Unix())
	fmt.Fprintf(&b, "# HELP gnet_reconcile_exit_status 0 = success, 1 = the reconcile returned an error.\n")
	fmt.Fprintf(&b, "# TYPE gnet_reconcile_exit_status gauge\n")
	fmt.Fprintf(&b, "gnet_reconcile_exit_status %d\n", m.exitStatus)
	fmt.Fprintf(&b, "# HELP gnet_reconcile_drift_count Installed modules absent from the profile (state ∖ plan ∖ manifest).\n")
	fmt.Fprintf(&b, "# TYPE gnet_reconcile_drift_count gauge\n")
	fmt.Fprintf(&b, "gnet_reconcile_drift_count %d\n", m.driftCount)
	fmt.Fprintf(&b, "# HELP gnet_reconcile_prune_armed 1 if this run computed drift and the host has opted into prune.\n")
	fmt.Fprintf(&b, "# TYPE gnet_reconcile_prune_armed gauge\n")
	fmt.Fprintf(&b, "gnet_reconcile_prune_armed %d\n", armed)
	fmt.Fprintf(&b, "# HELP gnet_reconcile_info Static info; pin_sha carries the engine git HEAD.\n")
	fmt.Fprintf(&b, "# TYPE gnet_reconcile_info gauge\n")
	fmt.Fprintf(&b, "gnet_reconcile_info{pin_sha=\"%s\",profile=\"%s\"} 1\n", pin, prof)

	// Write to a UNIQUE temp file in the same dir, then rename — so two concurrent
	// reconciles (e.g. a scheduled run overlapping a manual one) can never truncate
	// each other's tmp and publish a torn .prom (which node_exporter would drop
	// wholesale). The *.tmp suffix keeps the in-progress file out of the *.prom
	// scrape glob. os.CreateTemp gives the per-write unique name.
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, "gnet_reconcile.*.prom.tmp")
	if err != nil {
		u.Warn(fmt.Sprintf("reconcile metrics: could not create temp in %s: %v (metrics not published)", dir, err))
		return
	}
	tmp := f.Name()
	// os.CreateTemp makes the file 0600; node_exporter reads it as an UNPRIVILEGED
	// user (the upstream image runs as `nobody`, and the native unit as the
	// `node_exporter` user), so the published file MUST be world-readable or the
	// whole gnet_reconcile_* family silently vanishes. Chmod before the rename.
	if err := f.Chmod(0o644); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		u.Warn(fmt.Sprintf("reconcile metrics: could not chmod %s: %v (metrics not published)", tmp, err))
		return
	}
	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		u.Warn(fmt.Sprintf("reconcile metrics: could not write %s: %v (metrics not published)", tmp, err))
		return
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		u.Warn(fmt.Sprintf("reconcile metrics: could not close %s: %v (metrics not published)", tmp, err))
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		u.Warn(fmt.Sprintf("reconcile metrics: could not publish %s: %v", path, err))
		_ = os.Remove(tmp)
		return
	}
	if verbose {
		u.Info(fmt.Sprintf("reconcile metrics published to %s", filepath.Clean(path)))
	}
}

// escapeLabel escapes the characters the Prometheus exposition format reserves in a
// label value: backslash, double-quote, and newline.
func escapeLabel(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(s)
}
