package driver

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// D123: a variadic C function is called with each platform's convention
// (LLVM's, given the triple), the extra arguments promoted as C promotes
// them: an i8 and a u8 widen to int, an f32 to double, a bool to int, an
// unsuffixed literal is an int. POSIX `open` and `fcntl` — the reason for
// the decision — are variadic too (`_open` on Windows).
func TestCVariadicCalls(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	files := `extern "C" {
  fun open(path: *raw u8, flags: i32, ...): i32
  fun fcntl(fd: i32, cmd: i32, ...): i32
  fun close(fd: i32): i32
}

test "open with a mode, then fcntl reads its flags" {
  with path = try ffi.CString.of("made.txt")
  // SAFETY: path is a NUL-terminated string open for the call
  val fd = unsafe { open(path.ptr(), 0x41, 0o600) } // O_WRONLY | O_CREAT
  expect(fd >= 0)
  // SAFETY: fd was opened above; F_GETFL takes no third argument
  val flags = unsafe { fcntl(fd, 3) }
  expect(flags & 3 == 1) // O_WRONLY
  // SAFETY: as above
  expect(unsafe { close(fd) } == 0)
}
`
	if runtime.GOOS == "windows" {
		files = `extern "C" {
  fun _open(path: *raw u8, flags: i32, ...): i32
  fun _close(fd: i32): i32
}

test "_open with a mode" {
  with path = try ffi.CString.of("made.txt")
  // SAFETY: path is a NUL-terminated string open for the call
  val fd = unsafe { _open(path.ptr(), 0x0101, 0x0180) } // _O_WRONLY | _O_CREAT, _S_IREAD | _S_IWRITE
  expect(fd >= 0)
  // SAFETY: fd was opened above
  expect(unsafe { _close(fd) } == 0)
}
`
	}
	src := `use ffi

extern "C" {
  fun snprintf(buf: *raw u8, size: u64, format: *raw u8, ...): i32
}

fun format(f: string, call: fun(*raw u8, *raw u8): i32): string throws ffi.NulByte {
  with fmt = try ffi.CString.of(f)
  val buf: MutableList<u8> = MutableList.repeat(0, 128)
  buf.withRaw(p => call(p, fmt.ptr()))
  // SAFETY: snprintf wrote a NUL-terminated string into the 128 bytes
  return buf.withRaw(p => unsafe { ffi.readString(p) })
}

test "snprintf promotes as C does" {
  val small: u8 = 200
  val neg: i8 = -5
  val wide: i16 = -300
  val big: i64 = 9000000000
  val half: f32 = 0.5
  // SAFETY: each format names exactly the arguments passed; the buffer is 128 bytes
  val text = try format("%d|%u|%d|%lld|%.3f|%c|%d|%d|%.1f", (p, f) => unsafe { snprintf(p, 128, f, neg, small, wide, big, half, 65, true, -7, 2.3) })
  expect(text == "-5|200|-300|9000000000|0.500|A|1|-7|2.3")
}

test "a pointer and a string" {
  with word = try ffi.CString.of("veles")
  // SAFETY: the format reads one string
  val text = try format("[%s]", (p, f) => unsafe { snprintf(p, 128, f, word.ptr()) })
  expect(text == "[veles]")
}

` + strings.ReplaceAll(files, "made.txt", filepath.ToSlash(filepath.Join(t.TempDir(), "made.txt")))
	for _, release := range []bool{false, true} {
		out, code := runTestFiles(t, map[string]string{"main.vs": src}, Options{Release: release})
		if code != 0 || !strings.Contains(out, "3 passed") {
			t.Fatalf("release=%v: exit %d, output:\n%s", release, code, out)
		}
	}
}
