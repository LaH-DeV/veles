package driver

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"
)

// `veles new` makes a package that checks clean and whose test passes;
// it refuses a name other packages could not write in `use`, a template
// that does not exist, and never writes into a directory that has
// something in it.
func TestNew(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "hello")
	if code := New(dir, ""); code != 0 {
		t.Fatalf("new: exit %d", code)
	}
	for _, f := range []string{"veles.toml", "main.vs", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if code := Run(Options{Path: dir, Mode: "check"}); code != 0 {
		t.Errorf("the new package does not check clean")
	}
	if _, err := findClang(); err == nil {
		if code := Run(Options{Path: dir, Mode: "test"}); code != 0 {
			t.Errorf("the new package's test does not pass")
		}
	}
	if code := New(dir, ""); code == 0 {
		t.Errorf("new over a non-empty directory succeeded")
	}
	if code := New(filepath.Join(root, "my-app"), ""); code != 2 {
		t.Errorf("'my-app' accepted as a package name")
	}
	if code := New(filepath.Join(root, "other"), "desktop"); code != 2 {
		t.Errorf("an unknown template was accepted")
	}
}

// `--template server` is an HTTP service that checks clean, passes its
// tests, and serves: started with PORT=0 it says where, and /healthz
// answers.
func TestNewServer(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := filepath.Join(t.TempDir(), "svc")
	if code := New(dir, "server"); code != 0 {
		t.Fatalf("new: exit %d", code)
	}
	if code := Run(Options{Path: dir, Mode: "check"}); code != 0 {
		t.Fatalf("the server template does not check clean")
	}
	if code := Run(Options{Path: dir, Mode: "test"}); code != 0 {
		t.Fatalf("the server template's tests do not pass")
	}
	exe := filepath.Join(dir, "svc")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	if code := Run(Options{Path: dir, Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("the server template does not build")
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "PORT=0")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	// the line must arrive while the server runs: stdout is a pipe here,
	// and a buffered one (Windows' C runtime) kept it until the process
	// ended — a service that logs nothing until it stops
	lines := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(out).ReadString('\n')
		lines <- line
	}()
	var line string
	select {
	case line = <-lines:
	case <-time.After(30 * time.Second):
		t.Fatal("the server's first line did not arrive while it ran: stdout is held in a buffer")
	}
	m := regexp.MustCompile(`listening on (http://\S+/)`).FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("first line: %q", line)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(m[1] + "healthz")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(body) != "ok" || resp.Header.Get("x-request-id") == "" {
		t.Errorf("GET /healthz: %d %q, request id %q", resp.StatusCode, body, resp.Header.Get("x-request-id"))
	}
}
