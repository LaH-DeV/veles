package driver

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// std/log's own tests (formatting, levels) run as part of the suite.
func TestStdLogUnitTests(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	if code := Run(Options{Path: filepath.Join("..", "std", "log"), Mode: "test"}); code != 0 {
		t.Fatalf("veles test std/log: exit %d", code)
	}
}

// std/log (D91) in a program: JSON lines when standard error is not a
// terminal, the lazy message evaluated only when its level is on, fields
// bound with withFields carried by every line of the work — child tasks
// included — VELES_LOG setting the starting level, and lines from many
// threads never interleaved.
func TestLogInAProgram(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io { println }
use log { field }

val built = Mutex(value: 0)

fun costly(): string {
  built.withLock(n => *n += 1)
  "costly"
}

fun main() {
  log.info("start", field("path", "/x"), field("ms", 3), field("ok", true))
  log.debug("hidden ${costly()}")
  println("built after a hidden debug: ${built.get()}")
  log.setLevel(log.Level.Debug)
  log.debug("shown ${costly()}")
  println("built after a shown debug: ${built.get()}")
  log.withFields([field("id", "abc")], () => {
    log.warn("in scope")
    scope {
      val t = async childLine()
      await t
    }
  })
  log.error("say \"hi\"")
  scope {
    loop (i in 0..<8) {
      val _ = async worker(i)
    }
  }
  println("enabled: ${log.enabled(log.Level.Debug)}")
}

fun childLine() {
  log.info("from a child task")
}

fun worker(n: i64) {
  loop (i in 0..<50) {
    log.info("tick", field("worker", n), field("i", i))
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "logprog.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	runWith := func(env ...string) (string, string) {
		var stdout, stderr strings.Builder
		cmd := exec.Command(exe)
		cmd.Env = append(os.Environ(), env...)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("run: %v\n%s", err, stderr.String())
		}
		return strings.ReplaceAll(stdout.String(), "\r\n", "\n"), strings.ReplaceAll(stderr.String(), "\r\n", "\n")
	}
	// an unknown VELES_LOG says so once and logs at info
	_, stderr := runWith("VELES_LOG=loud")
	if !strings.Contains(stderr, "veles: VELES_LOG=loud is not debug, info, warn, error or off; logging at info") {
		t.Errorf("unknown level:\n%s", stderr)
	}
	stdout, stderr := runWith("VELES_LOG=warn")
	if want := "built after a hidden debug: 0\nbuilt after a shown debug: 1\nenabled: true\n"; stdout != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
	}
	// the level starts at warn, so `start` (info) is not logged, and the
	// warn and error lines are; setLevel(Debug) then let `shown` through
	type line map[string]any
	var lines []line
	for _, l := range strings.Split(strings.TrimSpace(stderr), "\n") {
		var v line
		if err := json.Unmarshal([]byte(l), &v); err != nil {
			t.Fatalf("not one JSON object per line: %v\n%q", err, l)
		}
		lines = append(lines, v)
	}
	stamp := regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d{3,6})?Z$`)
	var msgs []string
	ticks := map[float64]int{}
	for _, v := range lines {
		if s, _ := v["time"].(string); !stamp.MatchString(s) {
			t.Errorf("time %q is not RFC 3339 UTC", s)
		}
		msg, _ := v["msg"].(string)
		msgs = append(msgs, msg)
		switch msg {
		case "in scope":
			if v["id"] != "abc" || v["level"] != "warn" {
				t.Errorf("withFields line: %v", v)
			}
		case "from a child task":
			if v["id"] != "abc" {
				t.Errorf("the child task lost the fields: %v", v)
			}
		case "say \"hi\"":
			if _, has := v["id"]; has || v["level"] != "error" {
				t.Errorf("a line after the scope kept its fields, or lost its level: %v", v)
			}
		case "tick":
			ticks[v["worker"].(float64)]++
		}
	}
	joined := strings.Join(msgs, "|")
	if strings.Contains(joined, "start") || !strings.Contains(joined, "shown costly") {
		t.Errorf("levels: %s", joined)
	}
	for w := 0.0; w < 8; w++ {
		if ticks[w] != 50 {
			t.Errorf("worker %v logged %d ticks, want 50 (setLevel(Debug) lets info through)", w, ticks[w])
		}
	}
	// at the default level the info lines come out, typed
	_, stderr = runWith("VELES_LOG=info")
	first := strings.SplitN(strings.TrimSpace(stderr), "\n", 2)[0]
	var v line
	if err := json.Unmarshal([]byte(first), &v); err != nil {
		t.Fatalf("first line: %v\n%q", err, first)
	}
	if v["msg"] != "start" || v["path"] != "/x" || v["ms"] != 3.0 || v["ok"] != true || v["level"] != "info" {
		t.Errorf("start line has wrong types or values: %v", v)
	}
	// 8 workers x 50 lines, interleaved by scheduling but each line whole
	n := 0
	for _, l := range strings.Split(strings.TrimSpace(stderr), "\n") {
		var w line
		if err := json.Unmarshal([]byte(l), &w); err != nil {
			t.Fatalf("interleaved or broken line: %v\n%q", err, l)
		}
		if w["msg"] == "tick" {
			n++
		}
	}
	if n != 400 {
		t.Errorf("%d tick lines, want 400", n)
	}
}
