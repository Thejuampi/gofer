package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// testEnv holds paths to pre-built binaries and the fakeamps address.
type testEnv struct {
	goferBin    string
	fakeampsBin string
	addr        string
	uri         string
	cmd         *exec.Cmd
}

// setupTestEnv builds both binaries and starts fakeamps on a random port.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tmpDir := t.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	ampsRoot := ampsClientGoRoot(t)

	// Build fakeamps from the amps-client-go sibling repo.
	fakeampsBin := filepath.Join(tmpDir, "fakeamps"+ext)
	buildFakeamps := exec.Command("go", "build", "-o", fakeampsBin, "./tools/fakeamps")
	buildFakeamps.Dir = ampsRoot
	if out, err := buildFakeamps.CombinedOutput(); err != nil {
		t.Fatalf("build fakeamps: %v\n%s", err, out)
	}

	// Build gofer from this module's root.
	goferBin := filepath.Join(tmpDir, "gofer"+ext)
	buildGofer := exec.Command("go", "build", "-o", goferBin, ".")
	buildGofer.Dir = repoRoot(t)
	if out, err := buildGofer.CombinedOutput(); err != nil {
		t.Fatalf("build gofer: %v\n%s", err, out)
	}

	// Find a free port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	// Start fakeamps.
	cmd := exec.Command(fakeampsBin, "-addr", addr)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fakeamps: %v", err)
	}

	// Wait for fakeamps to accept connections.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	env := &testEnv{
		goferBin:    goferBin,
		fakeampsBin: fakeampsBin,
		addr:        addr,
		uri:         fmt.Sprintf("tcp://%s/amps/json", addr),
		cmd:         cmd,
	}

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	return env
}

// runGofer executes the gofer binary with the given args and returns stdout+stderr.
func (e *testEnv) runGofer(t *testing.T, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(e.goferBin, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec gofer: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// repoRoot walks up from the working directory to find this module's go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root (go.mod) from %s", dir)
		}
		dir = parent
	}
}

// ampsClientGoRoot returns the root of the sibling amps-client-go repository,
// resolved from the replace directive in this module's go.mod
// (i.e. "../amps-client-go" relative to this module root).
func ampsClientGoRoot(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	candidate := filepath.Join(root, "..", "amps-client-go")
	abs, err := filepath.Abs(candidate)
	if err != nil {
		t.Fatalf("resolve amps-client-go root: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(abs, "go.mod")); statErr != nil {
		t.Fatalf("amps-client-go repo not found at %s (expected sibling directory)", abs)
	}
	return abs
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestPing(t *testing.T) {
	env := setupTestEnv(t)

	stdout, stderr, code := env.runGofer(t, "ping", "-server", env.uri)
	if code != 0 {
		t.Fatalf("ping exit code = %d, stderr = %s", code, stderr)
	}
	if !strings.HasPrefix(stdout, "OK") {
		t.Fatalf("ping output = %q, want prefix 'OK'", stdout)
	}
}

func TestPingBadServer(t *testing.T) {
	env := setupTestEnv(t)

	_, _, code := env.runGofer(t, "ping", "-server", "tcp://127.0.0.1:1/amps/json", "-timeout", "1s")
	if code == 0 {
		t.Fatalf("ping to bad server should fail with non-zero exit code")
	}
}

func TestPublishAndSubscribe(t *testing.T) {
	env := setupTestEnv(t)

	// Publish a message.
	_, stderr, code := env.runGofer(t,
		"publish", "-server", env.uri, "-topic", "test.gofer", "-data", `{"hello":"gofer"}`,
	)
	if code != 0 {
		t.Fatalf("publish exit code = %d, stderr = %s", code, stderr)
	}

	// Subscribe with -n 1 in a goroutine, then publish another message.
	type result struct {
		stdout   string
		stderr   string
		exitCode int
	}
	ch := make(chan result, 1)
	go func() {
		out, err, ec := env.runGofer(t,
			"subscribe", "-server", env.uri, "-topic", "test.sub", "-n", "1",
		)
		ch <- result{out, err, ec}
	}()

	// Give the subscriber time to connect (Windows process startup is slow).
	time.Sleep(5 * time.Second)

	_, stderr, code = env.runGofer(t,
		"publish", "-server", env.uri, "-topic", "test.sub", "-data", `{"from":"publisher"}`,
	)
	if code != 0 {
		t.Fatalf("second publish exit code = %d, stderr = %s", code, stderr)
	}

	select {
	case r := <-ch:
		if r.exitCode != 0 {
			t.Fatalf("subscribe exit code = %d, stderr = %s", r.exitCode, r.stderr)
		}
		if !strings.Contains(r.stdout, `"from":"publisher"`) {
			t.Fatalf("subscribe output = %q, want to contain {\"from\":\"publisher\"}", r.stdout)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("subscribe -n 1 timed out")
	}
}

func TestSOW(t *testing.T) {
	env := setupTestEnv(t)

	// Seed SOW data.
	_, stderr, code := env.runGofer(t,
		"publish", "-server", env.uri, "-topic", "sow.gofer", "-data", `{"id":1,"val":"seed"}`,
	)
	if code != 0 {
		t.Fatalf("seed publish exit code = %d, stderr = %s", code, stderr)
	}

	// Query SOW.
	stdout, stderr, code := env.runGofer(t,
		"sow", "-server", env.uri, "-topic", "sow.gofer",
	)
	if code != 0 {
		t.Fatalf("sow exit code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, `"id":1`) {
		t.Fatalf("sow output = %q, want to contain '\"id\":1'", stdout)
	}
}

func TestSOWAndSubscribe(t *testing.T) {
	env := setupTestEnv(t)

	// Seed SOW.
	_, stderr, code := env.runGofer(t,
		"publish", "-server", env.uri, "-topic", "sowsub.gofer", "-data", `{"id":1,"seed":true}`,
	)
	if code != 0 {
		t.Fatalf("seed publish exit code = %d, stderr = %s", code, stderr)
	}

	// sow_and_subscribe with -n 2: should get the SOW record + one live update.
	type result struct {
		stdout   string
		stderr   string
		exitCode int
	}
	ch := make(chan result, 1)
	go func() {
		out, err, ec := env.runGofer(t,
			"sow_and_subscribe", "-server", env.uri, "-topic", "sowsub.gofer", "-n", "2",
		)
		ch <- result{out, err, ec}
	}()

	// Give it time to get SOW, then publish a live update (Windows process startup is slow).
	time.Sleep(5 * time.Second)

	_, stderr, code = env.runGofer(t,
		"publish", "-server", env.uri, "-topic", "sowsub.gofer", "-data", `{"id":2,"live":true}`,
	)
	if code != 0 {
		t.Fatalf("live publish exit code = %d, stderr = %s", code, stderr)
	}

	select {
	case r := <-ch:
		if r.exitCode != 0 {
			t.Fatalf("sow_and_subscribe exit code = %d, stderr = %s", r.exitCode, r.stderr)
		}
		if !strings.Contains(r.stdout, `"id":1`) {
			t.Fatalf("sow_and_subscribe output missing SOW record: %q", r.stdout)
		}
		if !strings.Contains(r.stdout, `"id":2`) {
			t.Fatalf("sow_and_subscribe output missing live record: %q", r.stdout)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("sow_and_subscribe -n 2 timed out")
	}
}

func TestSOWDelete(t *testing.T) {
	env := setupTestEnv(t)

	// Seed.
	_, stderr, code := env.runGofer(t,
		"publish", "-server", env.uri, "-topic", "sowdel.gofer", "-data", `{"id":1,"deleteme":true}`,
	)
	if code != 0 {
		t.Fatalf("seed publish exit code = %d, stderr = %s", code, stderr)
	}

	// Delete.
	stdout, stderr, code := env.runGofer(t,
		"sow_delete", "-server", env.uri, "-topic", "sowdel.gofer", "-filter", "/id = 1",
	)
	if code != 0 {
		t.Fatalf("sow_delete exit code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, "deleted") {
		t.Fatalf("sow_delete output = %q, want to contain 'deleted'", stdout)
	}

	// Verify SOW is now empty.
	stdout, stderr, code = env.runGofer(t,
		"sow", "-server", env.uri, "-topic", "sowdel.gofer",
	)
	if code != 0 {
		t.Fatalf("verification sow exit code = %d, stderr = %s", code, stderr)
	}
	if strings.Contains(stdout, `"deleteme"`) {
		t.Fatalf("sow should be empty after delete, got: %q", stdout)
	}
}

func TestHelpOutput(t *testing.T) {
	env := setupTestEnv(t)

	stdout, _, code := env.runGofer(t, "help")
	if code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(stdout, "gofer") {
		t.Fatalf("help output should mention 'gofer': %q", stdout)
	}
}

func TestUnknownCommand(t *testing.T) {
	env := setupTestEnv(t)

	_, stderr, code := env.runGofer(t, "nonexistent")
	if code == 0 {
		t.Fatalf("unknown command should fail")
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("stderr should mention unknown command: %q", stderr)
	}
}

func TestNoArgs(t *testing.T) {
	env := setupTestEnv(t)

	_, _, code := env.runGofer(t)
	if code == 0 {
		t.Fatalf("no args should fail with non-zero exit code")
	}
}
