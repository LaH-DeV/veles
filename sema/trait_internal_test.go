package sema

import (
	"strings"
	"testing"
)

// A trait's default method written `internal` is its module's own hook
// (std/cli's `Args.readArgs`): the module calls it on any implementing type,
// another module cannot — as a static call or a method call — though every
// implementing type carries it.
func TestTraitInternalMethod(t *testing.T) {
	lib := `public trait Args {
  internal static fun make(): This => This.build(f => f.Type.default())
  internal fun describe(): string => "args"
  fun shown(): string => this.describe()
}

public fun parse<T: Args>(): T => T.make()
`
	ok := checkFiles(t, map[string]string{
		"lib/lib.vs": lib,
		"main.vs": `use io
use lib

struct Options {
  workers: i64 = 4
  implement lib.Args
}

fun main() {
  val o = lib.parse<Options>()
  io.println("${o.workers} ${o.shown()}")
}
`,
	})
	if errs := errorsOf(ok); errs != "" {
		t.Fatalf("the module's own calls and a public default are allowed:\n%s", errs)
	}
	bad := checkFiles(t, map[string]string{
		"lib/lib.vs": lib,
		"main.vs": `use io
use lib

struct Options {
  workers: i64 = 4
  implement lib.Args
}

fun main() {
  val o = Options.make()
  io.println(o.describe())
}
`,
	})
	errs := errorsOf(bad)
	for _, want := range []string{
		"'make' is internal to module 'lib', where trait 'Args' is declared",
		"'describe' is internal to module 'lib', where trait 'Args' is declared",
	} {
		if !strings.Contains(errs, want) {
			t.Errorf("want %q in:\n%s", want, errs)
		}
	}
}
