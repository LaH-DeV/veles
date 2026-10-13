package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// D150: an executor's state is named by its code alone; a pool holds
// none; a declared executor is not closed by code; its functions are
// called, not taken as values, from outside; what crosses to it is
// Sendable; its arguments are named, and its name is its declaration's.
func TestExecutorDeclarationRules(t *testing.T) {
	const gl = `executor gl {
  var textures: MutableMap<i64, string> = [:]
  public fun texture(id: i64): string => textures.get(id) ?: "none"
  public fun count(): i64 => textures.len()
  public fun fill(ids: MutableList<i64>) { }
}
`
	expectClean(t, prelude+gl+`fun main() {
  io.println(gl.texture(1))
  val n = gl.run(() => gl.count())
  io.println("$n")
}`)
	expectError(t, prelude+gl+`fun main() {
  io.println("${gl.textures.len()}")
}`, "'textures' is the state of executor 'gl': only gl's own code names it")
	expectError(t, prelude+gl+`fun main() {
  gl.close()
}`, "executor 'gl' is declared: it runs for the whole program and is not closed by code")
	expectError(t, prelude+gl+`fun main() {
  val f = gl.texture
  io.println(f(1))
}`, "'gl.texture' runs on executor 'gl': outside its code it is called, not taken as a value")
	expectError(t, prelude+gl+`fun main() {
  val ids: MutableList<i64> = [1]
  gl.fill(ids)
}`, "argument of type 'MutableList<i64>' is not Sendable, and this call runs 'fill' on executor 'gl'")
	// a plain function cannot wait for the executor
	expectError(t, prelude+gl+`fun main() throws ThreadError {
  with t = try Thread.start(name: "t", f: () => io.println(gl.texture(1)))
}`, "this lambda suspends")
	// a lambda that may run on any thread is not gl's code
	expectError(t, prelude+`executor gl {
  var made: i64 = 0
  public fun count(): i64 => blocking(() => made)
}
fun main() { io.println("${gl.count()}") }`, "'gl.made' is the state of executor 'gl', and only gl's own code names it")
	expectError(t, prelude+`executor gl {
  fun inner(): i64 => 1
  public fun outer(): i64 => blocking(() => inner())
}
fun main() { io.println("${gl.outer()}") }`, "'inner' runs on executor 'gl', and this function may run on another thread: call it as 'gl.inner(…)'")
	// a pool holds no state
	expectError(t, prelude+`executor cpu(threads: 2) {
  var hits: i64 = 0
}
fun main() { }`, "a pool runs its functions on several threads at once, so it holds no state of its own")
	expectError(t, prelude+`executor gl(name: "x")
fun main() { }`, "an executor is named by its declaration: its threads are named 'gl'")
	expectError(t, prelude+`executor gl(2)
fun main() { }`, "an executor's arguments are named")
	expectError(t, prelude+`executor gl(stack: 1)
fun main() { }`, "an executor takes 'threads:', 'priority:' and 'cpus:', not 'stack:'")
	// a module-level var is still refused; an executor's is not
	expectClean(t, prelude+`executor gl {
  var hits: i64 = 0
  public fun hit(): i64 {
    hits += 1
    hits
  }
}
fun main() { io.println("${gl.hit()}") }`)
	// `async gl.f(x)` evaluates its arguments here, a call among them too,
	// and a held launch (D111) keeps its arguments before it
	expectClean(t, prelude+gl+`fun id(): i64 => 1
fun main() {
  var n: i64 = 2
  scope {
    async gl.texture(id())
    val t = async gl.texture(n + id())
    n = 7
    io.println("${await t}")
  }
}`)
	// what crosses is still checked: Sendable
	expectError(t, prelude+gl+`fun ids(): MutableList<i64> => MutableList()
executor cpu(threads: 2) {
  public fun sum(xs: MutableList<i64>): i64 => xs.len()
}
fun main() {
  scope {
    async cpu.sum(ids())
  }
}`, "argument of type 'MutableList<i64>' is not Sendable, and this call runs 'sum' on executor 'cpu'")
}

// D150 across modules: `mod.gl.f(x)` reaches a public function of a public
// executor, and runs it there; a function not marked public stays the
// module's (M5).
func TestExecutorAcrossModules(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("render/lib.vs", "public executor gl {\n  var frames: i64 = 0\n  public fun frame(): i64 {\n    frames += 1\n    frames\n  }\n  fun secret(): i64 => frames\n}\n")
	write("main.vs", "use io\nuse render\nfun main() {\n  io.println(\"${render.gl.frame()}\")\n  io.println(\"${render.gl.secret()}\")\n}\n")
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(root, diags)
	if err != nil {
		t.Fatal(err)
	}
	Check(pkg, diags, false)
	got := diags.Render()
	if want := "'gl.secret' is private to module 'render'"; !strings.Contains(got, want) {
		t.Errorf("missing %q in:\n%s", want, got)
	}
	if strings.Contains(got, "frame") {
		t.Errorf("the public function should be reachable:\n%s", got)
	}
}
