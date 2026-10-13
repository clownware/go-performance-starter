// Package stopgate tests the Stop hook, .claude/hooks/stop-gate.sh (ADR-033).
// The hook is a shell script; the test runs it with a stub `go` on PATH.
package stopgate

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const stubGo = `#!/bin/sh
case "$1:$STUB_FAIL" in
  test:test) echo "--- FAIL: TestSomething"; exit 1 ;;
  run:check) echo "BLOCKER  some-check"; exit 1 ;;
esac
exit 0
`

type result struct {
	exit   int
	stdout string
	stderr string
}

// stop runs the hook once, as one attempt by an agent to finish its turn.
func stop(t *testing.T, tmp, fail string, env ...string) result {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("..", "..", ".claude", "hooks", "stop-gate.sh"))
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(stubGo), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", script)
	cmd.Dir = tmp
	cmd.Stdin = strings.NewReader(`{"session_id":"abc-123","hook_event_name":"Stop","stop_hook_active":true}`)
	cmd.Env = append([]string{"PATH=" + bin + ":/usr/bin:/bin", "TMPDIR=" + tmp, "STUB_FAIL=" + fail}, env...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running hook: %v", err)
		}
		code = ee.ExitCode()
	}
	return result{code, out.String(), errb.String()}
}

func TestStopGate(t *testing.T) {
	tests := []struct {
		name string
		// fail names what is broken at each successive stop of one session:
		// "test", "check" or "" for a clean tree.
		fail []string
		env  []string
		// want is the exit code at each stop; warn says whether the last
		// stop hands the operator a warning instead of blocking.
		want []int
		warn bool
	}{
		{name: "clean tree finishes", fail: []string{""}, want: []int{0}},
		{name: "failing tests block the first stop", fail: []string{"test"}, want: []int{2}},
		{name: "a block-status check blocks the first stop", fail: []string{"check"}, want: []int{2}},
		{name: "still failing blocks a second time", fail: []string{"test", "test"}, want: []int{2, 2}},
		{name: "third failing stop is let through with a warning to the operator", fail: []string{"test", "test", "test"}, want: []int{2, 2, 0}, warn: true},
		{name: "fixed after a block finishes without a warning", fail: []string{"test", ""}, want: []int{2, 0}},
		{name: "kill-switch disables the gate", fail: []string{"test"}, env: []string{"STOP_GATE_OFF=1"}, want: []int{0}},
		{name: "block limit is configurable", fail: []string{"check", "check"}, env: []string{"STOP_GATE_MAX_BLOCKS=1"}, want: []int{2, 0}, warn: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			var last result
			for i, f := range tt.fail {
				last = stop(t, tmp, f, tt.env...)
				if last.exit != tt.want[i] {
					t.Fatalf("stop %d: exit %d, want %d (stderr %q)", i+1, last.exit, tt.want[i], last.stderr)
				}
				if last.exit == 2 && !strings.Contains(last.stderr, "Stop-gate BLOCKER") {
					t.Errorf("stop %d: a block must say why on stderr, got %q", i+1, last.stderr)
				}
			}
			warned := strings.Contains(last.stdout, `"systemMessage"`) && strings.Contains(last.stdout, "Stop-gate")
			if warned != tt.warn {
				t.Errorf("operator warning = %v, want %v (stdout %q)", warned, tt.warn, last.stdout)
			}
		})
	}
}

func TestStopGateSweepsStaleCounters(t *testing.T) {
	tmp := t.TempDir()
	stale := filepath.Join(tmp, "stop-gate-dead-session.count")
	fresh := filepath.Join(tmp, "stop-gate-live-session.count")
	other := filepath.Join(tmp, "unrelated.count")
	for _, f := range []string{stale, fresh, other} {
		if err := os.WriteFile(f, []byte("1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-72 * time.Hour)
	for _, f := range []string{stale, other} {
		if err := os.Chtimes(f, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if got := stop(t, tmp, ""); got.exit != 0 {
		t.Fatalf("clean stop exited %d", got.exit)
	}
	tests := []struct {
		name   string
		path   string
		exists bool
	}{
		{"a counter left by a session that died days ago is removed", stale, false},
		{"another session's recent counter is kept", fresh, true},
		{"a file that is not a counter is never touched", other, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := os.Stat(tt.path)
			if (err == nil) != tt.exists {
				t.Errorf("exists = %v, want %v", err == nil, tt.exists)
			}
		})
	}
}
