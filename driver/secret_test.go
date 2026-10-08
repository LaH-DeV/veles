package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D112: a Secret prints as [redacted], exposes a copy, compares in constant
// time, decodes from JSON, and panics at the caller once closed; under a
// tiny collection threshold the bytes survive collections while the Secret
// is alive. And the collector zeroes a freed Secret's bytes: the runtime's
// probe allocates one object of the wiping descriptor and one ordinary
// object, drops both, collects, and reads the two slots back.
func TestSecret(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io, json

struct Config {
  port: i64 = 8080
  databaseUrl: Secret<string>
  implement Decodable
}

extern "C" {
  fun veles_gc_test_wipe(): i64
}

fun churn(n: i64): i64 {
  var total = 0
  loop (i in 0..<n) {
    with s = Secret.of("key-$i".bytes())
    total += s.expose().len()
  }
  total
}

fun main() throws DecodeError {
  with s = Secret.of("hunter2")
  io.println("pw: $s, ${s.len()} bytes, ${s.expose()}")
  io.println("${s == Secret.of("hunter2")} ${s == Secret.of("hunter3")} ${s == Secret.of("hunter")}")
  val cfg = try json.decode<Config>("{\"databaseUrl\": \"postgres://u:p@h/db\"}")
  io.println("$cfg ${cfg.databaseUrl.expose()}")
  io.println("churn ${churn(2000)} ${s.expose()}")
  // SAFETY: a runtime test hook that takes nothing and returns a number
  io.println("wiped when freed: ${unsafe { veles_gc_test_wipe() }}")
  val k = Secret.of("key".bytes())
  k.close()
  k.close()
  io.println("closed")
  io.println("${k.expose()}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "pw: [redacted], 7 bytes, hunter2\n" +
		"true false false\n" +
		"Config(port: 8080, databaseUrl: [redacted]) postgres://u:p@h/db\n" +
		"churn 14890 hunter2\n" +
		"wiped when freed: 1\n" +
		"closed\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "secret.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		var stderr strings.Builder
		cmd := exec.Command(exe)
		// without VELES_GC_POISON, which a stress run may set: it fills a swept
		// object with 0xCD after the wipe, and the probe reads the bytes
		var env []string
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "VELES_GC_POISON=") {
				env = append(env, kv)
			}
		}
		cmd.Env = append(env, "VELES_GC_THRESHOLD=4096")
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err == nil {
			t.Fatalf("release=%v: expose() after close() should panic", release)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: stdout\n%s\nwant\n%s", release, got, want)
		}
		panicWant := "panic: Secret.expose: the secret was closed\n  at main.vs:35:17"
		if got := strings.ReplaceAll(stderr.String(), "\r\n", "\n"); !strings.HasPrefix(got, panicWant) {
			t.Errorf("release=%v: stderr\n%s\nwant it to start with\n%s", release, got, panicWant)
		}
	}
}
