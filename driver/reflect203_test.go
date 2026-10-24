package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D203: what the reflection gives a library, as values when the program
// runs — a struct built field by field from declared defaults, optional and
// list fields told apart by constant branches, an enum's members and a
// sealed family's variants (their docs and annotations included), and enums
// read back by `Parsable`. Debug and release.
func TestReflectionForLibraries(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	const program = `use io

annotation Alias(name: string) on variant, enumMember

enum Level {
  /// Few messages.
  Low
  @Alias("hi")
  High
}

sealed trait Command {
  /// Builds the program.
  struct Build {
    release: bool = false
  }
  struct Test {
    filter: string = ""
  }
}

struct Options {
  workers: i64 = 4
  limit:   i64? = null
  files:   List<string> = []
  level:   Level = Level.Low
  name:    string
}

fun decode<V>(text: string): V? {
  if (V implements Parsable) return V.parse(text)
  null
}

fun given(name: string): string? => when (name) {
  "limit" => "7"
  "files" => "a"
  "level" => "High"
  "name"  => "tool"
  else    => null
}

fun filled(): Options => Options.build(f => when {
  f.isOptional => decode<f.Inner>(given(f.name) ?: "")
  f.isList     => if (val text = given(f.name)) [decode<f.Inner>(text) ?: ""] else []
  f.hasDefault => if (val text = given(f.name)) decode<f.Type>(text) ?: f.default() else f.default()
  else         => decode<f.Type>(given(f.name) ?: "") ?: f.Type.default()
})

const fun memberNames<T>(): List<string> {
  val out: MutableList<string> = []
  loop (m in T.members) out.push("${m.index}:${m.name}:${m.doc}")
  out
}

const LEVELS: List<string> = memberNames<Level>()
const COMMANDS: List<string> = memberNames<Command>()

// annotations on members, read from a MemberInfo in a const fun
const fun aliased<T>(): List<string> {
  val out: MutableList<string> = []
  loop (m in T.members) {
    if (val a = m.attr<Alias>()) out.push("${m.name}=${a.name}") else if (!m.has<Alias>()) out.push(m.name)
  }
  out
}

const ALIASED: List<string> = aliased<Level>()

fun aliases(): string {
  var out = ""
  inline loop (m in Level.members) {
    if (val a = m.attr<Alias>()) out += m.name + "=" + a.name
  }
  out
}

fun main() {
  val o = filled()
  io.println("${o.workers} ${o.limit} ${o.files} ${o.level} ${o.name}")
  io.println("$LEVELS $COMMANDS ${aliases()} ${decode<Level>("Low")} ${decode<Level>("Mid")} $ALIASED")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "4 7 [a] High tool\n" +
		"[0:Low:Few messages., 1:High:] [0:Build:Builds the program., 1:Test:] High=hi Low null [Low, High=hi]\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "main.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("release=%v: build failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); err != nil || got != want {
			t.Errorf("release=%v: got\n%s(%v)\nwant\n%s", release, got, err, want)
		}
	}
}
