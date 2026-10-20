package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D174: an annotation and the trait that reads it are declared in one
// package and used from another. The reader walks the fields of a type it
// has never seen — a `private` field included, because the type opted in
// with `implement Validate` — and a table built by a `const fun` from the
// annotations is part of the program. Debug and release.
func TestAnnotationsAcrossPackages(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("lib/veles.toml", "[package]\nname = \"vlib\"\nversion = \"0.1.0\"\n")
	write("lib/vlib.vs", `public annotation Min(value: i64) on field for Validate
public annotation Label(text: string) on field, struct for Validate
public annotation Skip on field for Validate

public trait Validate {
  fun problems(): List<string> {
    val out: MutableList<string> = []
    inline loop (f in This.fields) {
      if (!f.has<Skip>()) {
        if (val min = f.attr<Min>()) {
          if (f.get(this) < min.value) out.push("${f.name} must be at least ${min.value}")
        }
      }
    }
    out
  }

  static const fun labels(): List<string> {
    val out: MutableList<string> = []
    loop (f in This.fields) {
      out.push(f.name + "=" + (f.attr<Label>()?.text ?: "-"))
    }
    out
  }

  static fun blank(): This => This.build(f => f.Type.default())
}
`)
	write("app/veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n\n[dependencies]\nvlib = \"../lib\"\n")
	write("app/main.vs", `use io
use vlib { Label, Min, Skip, Validate }

struct Account {
  @Min(5)
  private balance: i64
  @Skip
  note: string
  @vlib.Label("who")
  owner: string
  implement Validate
}

const LABELS: List<string> = Account.labels()

fun main() {
  val a = Account(balance: 1, note: "x", owner: "ann")
  io.println("${a.problems()}")
  io.println("${LABELS}")
  io.println("${Account.blank()}")
}
`)
	want := "[balance must be at least 5]\n[balance=-, note=-, owner=who]\nAccount(balance: 0, note: , owner: )\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(root, "app.exe")
		if code := Run(Options{Path: filepath.Join(root, "app"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("release=%v: build failed (%d)", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}

// D174: what a program may not do with annotations is refused where it is
// written, each with the rule.
func TestAnnotationMisuse(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.vs"), []byte(`annotation Short(letter: string) on field

struct O {
  @Short("w")
  workers: i64 = 4
}

const fun letters<T>(): List<string> {
  val out: MutableList<string> = []
  loop (f in T.fields) {
    out.push(f.attr<Short>()?.letter ?: "")
  }
  out
}

fun main() {
  val runtime = letters<O>()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Run(Options{Path: root, Mode: "check"}); code == 0 {
		t.Fatal("calling a function that reads annotations from code that runs was accepted")
	}
}
