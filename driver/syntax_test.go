package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D189–D192, end to end: the membership operator, a loop head that binds, the
// bit and wrapping compound assignments, and a string slice that never fails.
// Run in a debug and a release build.
func TestSyntaxAdditions(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

struct Tokens {
  var i: i64 = 0
  items: List<string>

  fun next(): string? {
    val t = this.items.at(this.i)
    this.i += 1
    t
  }
}

struct Bits {
  var word: u32 = 0xF0
}

sealed trait Shape {
  struct Circle { r: i64 }
  struct Rect {
    w: i64
    h: i64
  }
}

fun area(s: Shape): i64 => when (s) {
  is Circle => s.r * s.r * 3
  is Rect => s.w * s.h
}

enum Level { Debug, Info, Warn, Error }

extend Level {
  fun label(): string => when (this) {
    Level.Debug => "dbg"
    Level.Info => "inf"
    Level.Warn => "wrn"
    Level.Error => "err"
  }
  fun isSevere(): bool => this >= Level.Warn
}

extend Level with fun next(): Level? => Level.fromValue(this.value + 1)

fun kind(c: u8): string {
  if (c in '0'..'9') return "digit"
  if (c in ['a', 'e', 'i', 'o', 'u']) return "vowel"
  if (c in 'a'..'z') return "consonant"
  if (c in " \t\r\n") return "space"
  if (c !in "+-*/") return "other"
  return "operator"
}

fun main() {
  // D189: in / !in
  val stops: Set<string> = ["the", "a"]
  val headers: Map<string, i64> = ["host": 1]
  val xs = [10, 20, 30]
  val url = "https://example.com"
  var out = ""
  loop (b in "7e k+z!".bytes()) out += kind(b) + " "
  io.println(out)
  io.println("${"the" in stops} ${"cat" !in stops} ${"host" in headers} ${20 in xs} ${40 in xs} ${5 in 1..<5} ${5 in 1..5}")
  val m: u8 = 'm'
  io.println("${"://" in url} ${"zzz" in url} ${m in url} ${m in "xyz"} ${(m in 'a'..'z') && !(m in 0..9)}")

  // D190: a loop head that binds
  val t = Tokens(items: ["a", "bb", "", "ccc"])
  var seen = 0
  loop (val tok = t.next()) {
    if (tok.isEmpty()) continue
    seen += tok.len()
  }
  val u = Tokens(items: ["x", "yy", "zzz"])
  var total = 0
  loop (val tok = u.next() && tok.len() < 3) {
    total += tok.len()
  }
  io.println("seen $seen total $total")

  // D191: compound assignments
  var a: u32 = 0b1010
  a |= 0b0101
  a &= 0b1100
  a ^= 0b0110
  a <<= 4
  a >>= 1
  var w: u8 = 250
  w +%= 10
  var low: u8 = 3
  low -%= 5
  var p: u8 = 200
  p *%= 2
  var n: i64 = 3
  var shifted: u32 = 1
  shifted <<= n
  val bits = Bits()
  bits.word |= 0x0F
  io.println("$a $w $low $p $shifted ${bits.word}")

  // D194: extension functions on an enum
  io.println("${Level.Warn.label()} ${Level.Warn.isSevere()} ${Level.Debug.isSevere()} ${Level.Info.next()?.label()} ${Level.Error.next()}")

  // D193: variants declared inside a sealed trait
  io.println("${area(Circle(r: 2))} ${area(Rect(w: 2, h: 3))}")

  // D192: slice
  val text = "héllo wörld"
  io.println("${text.slice(0, 3)}|${text.slice(2, 3)}|${text.slice(-4, 2)}|${text.slice(8, 99)}|${text.slice(5, 2).isEmpty()}")
}
`
	want := "digit vowel space consonant operator consonant other \n" +
		"true true true true false false true\n" +
		"true false true false true\n" +
		"seen 6 total 3\n" +
		"80 4 254 144 8 255\n" +
		"wrn true false wrn null\n" +
		"12 6\n" +
		"hé|é|hé|örld|true\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "syntax.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: stdout\n%s\nwant\n%s", release, got, want)
		}
	}
}
