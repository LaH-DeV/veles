# 15. Files, paths and processes

Three small modules turn Veles from a calculator into a tool: `fs` reads
and writes files, `path` manipulates file names as text, and `os` gives
the program its arguments, its environment and other programs. None of
them is in scope by default — `use` the ones you need.

## Reading and writing files

Every `fs` call that can fail throws `IoError`, so the rules of
[chapter 7](07-errors.md) apply: `try` to propagate, `when` to handle.

```veles
use fs, io, os, path

fun main() throws IoError {
  val dir = path.join(os.tempDir(), "veles-tutorial-15")
  try fs.mkdir(dir)                       // parents too; fine if it exists
  val file = path.join(dir, "todo.txt")

  try fs.writeFile(file, "milk\n")        // replaces
  try fs.appendFile(file, "bread\n")      // creates when missing
  val items = try fs.readFile(file).lines()
  io.println("${items.len()} items: ${items.join(", ")}")

  io.println("${fs.exists(file)} ${fs.isFile(file)} ${fs.isDir(dir)}")
  io.println("${try fs.listDir(dir)}")

  try fs.remove(file)
  try fs.remove(dir)                      // must be empty
  io.println("${fs.exists(dir)}")
}
```

Output:
```text
2 items: milk, bread
true true true
[todo.txt]
false
```

`fs.stat(path)` says more than the three questions above: the size, the time
of the last write and whether it is a directory. A link is followed, and a
missing entry is an `IoError` (`NotFound`):

```veles
// fragment
val st = try fs.stat("report.pdf")
io.println("${st.size} bytes, written ${st.modified}, directory: ${st.isDir}")
```

`st.modified` is a `Timestamp` ([chapter 20](20-time.md)), to the microsecond
where the file system keeps that much (NTFS keeps 100 ns, ext4 a nanosecond,
some systems only whole seconds); it is what a `Last-Modified` header and a
build tool's "is this out of date" both read.

`fs.open(path, mode)` opens a file for what does not fit in memory or should
not be held whole: reading a piece at a time, or writing as the data arrives.
`FileMode.Read` (the default) needs the file to exist, `Write` creates it or
empties it, `Append` creates it or keeps what it has. A `File` is `Closeable`,
so `with` closes it:

```veles
use fs, io, os, path

fun main() throws IoError {
  val p = path.join(os.tempDir(), "veles-tutorial-15-open.bin")
  with (f = try fs.open(p, fs.FileMode.Write)) {
    try f.write("0123456789".bytes())
    try f.write("abcdef".bytes())
  }
  with (f = try fs.open(p)) {
    io.println("${try f.size()} bytes")
    io.println("${(try f.readAt(8, 4)).decodeUtf8() ?: ""}")
    io.println("${(try f.read(4)).decodeUtf8() ?: ""} ${(try f.read(100)).decodeUtf8() ?: ""}")
  }
  try fs.remove(p)
}
```

Output:
```text
16 bytes
89ab
0123 456789abcdef
```

`read(max)` continues from where the last `read` ended and returns an empty
list at the end of the file; fewer than `max` bytes is normal, not the end.
`readAt(offset, max)` reads anywhere, in any order, and does not move the
place `read` continues from. A file opened to read cannot be written and the
other way round, and a closed file refuses; every failure is an `IoError`.

`listDir` gives one directory's names. `fs.walk(root)` gives every file
under a directory, as paths, depth-first with each directory's entries in
name order — the same tree always walks the same way. It does not follow
a symbolic link to a directory (or a Windows junction), so a link that
points back up the tree cannot make it endless, and it keeps its own
stack, so a deep tree cannot overflow yours:

```veles
// fragment
loop (file in try fs.walk("src")) {
  if (path.ext(file) == ".vs") io.println(file)   // src/a.vs, src/lexer/b.vs, ...
}
```

An `IoError` carries what went wrong and where: `kind` says which failure
it is, the same on every platform (D76); `detail` is the system's
description, `path` the file, `code` the platform's own error number, and
`message()` combines description and path. Decide by `kind` —
`IoKind.NotFound`, `PermissionDenied`, `AlreadyExists`, `IsADirectory`,
`ConnectionRefused`, `TimedOut`, `AddressInUse`, `BrokenPipe`, … and
`Other` for what the enum does not name — never by `code`, which differs
between systems. A path holding a NUL byte is `InvalidInput` before
anything is opened: the system would read it only up to the NUL, a
different file from the one your code checked.

```veles
use fs, io

fun main() {
  when (val r = fs.readFile("no/such/file.txt")) {
    is Ok  => io.println("read ${r.len()} bytes")
    is Err => io.println("${r.message()} (${r.kind})")
  }
  // a missing file is a default; any other failure is still an error
  val config = fs.readFile("app.toml") catch (e) {
    if (e.kind != IoKind.NotFound) panic("cannot read app.toml: ${e.message()}")
    ""
  }
  io.println("config: ${config.len()} bytes")
}
```

Output:
```text
No such file or directory: no/such/file.txt (NotFound)
config: 0 bytes
```

`readFile` insists on UTF-8, because a string is always well-formed text
(D18); anything else is an `IoError` ("Illegal byte sequence"). For raw
data use the byte forms: `fs.readBytes(path)` gives a `List<u8>`,
`fs.writeBytes(path, bytes)` and `fs.appendBytes` take one, and
`text.bytes()` / `bytes.decodeUtf8()` convert. `io.readAll()` reads the
rest of standard input as text.

## Writing without losing data

`fs.writeFile` replaces a file in place: a crash or a full disk in the middle
leaves half of it. For a file that must always be whole — state, a cache
index, anything another program reads — use `writeAtomic`: the bytes go to a
new file beside it, reach the disk, and the new file is renamed over the old
one, so a reader and a crash see the old contents or the new, never a mixture.
An existing file keeps its permissions. (On Windows a file that is open
elsewhere cannot be replaced: close it first.)

```veles
use fs, io { println }, os, path

fun main() throws IoError {
  val dir = path.join(os.tempDir(), "veles-doc-atomic")
  if (!fs.isDir(dir)) try fs.mkdir(dir)
  val file = path.join(dir, "state.txt")
  try fs.writeAtomic(file, "version 1".bytes())
  try fs.writeAtomic(file, "version 2, longer".bytes())
  println(try fs.readFile(file))
  try fs.copy(file, file + ".bak")           // a piece at a time: any size
  println("${try fs.listDir(dir)}")
  try fs.remove(file + ".bak")
  try fs.remove(file)
  try fs.remove(dir)
}
```

Output:
```text
version 2, longer
[state.txt, state.txt.bak]
```

An open `File` has more: `file.seek(offset)` moves where the next `read` or
`write` goes (a file opened to `Append` always writes at its end),
`file.sync()` waits until what was written is on the disk, and
`file.lock()` takes an exclusive lock that other programs asking for it wait
for — *advisory*, like `flock`: it keeps out the polite, not a program that
just opens the file. `tryLock()` answers `null` instead of waiting. A lock is
a `Closeable`, held with `with` and let go at the end of the block (and when
the file closes).

```veles
// fragment
with file = try fs.open("app.lock", fs.FileMode.Append)   // created when missing, never emptied
with lock = (try file.tryLock()) ?: panic("already running")
// only one process is here
```

`fs.lines(path, max: n)` reads a big file one line at a time. It is an
iterator of `Result`s, because a read can fail halfway: `try line` passes an
error on, a line longer than `max` bytes is one (so a file with no newlines
cannot become one huge allocation), and after an error or at the end the file
is closed. `with lines = try fs.lines(...)` closes it early.

```veles
use fs, io { println }, os, path

fun longest(file: string): i64 throws IoError {
  var longest = 0
  loop (line in try fs.lines(file, max: 4096)) {
    longest = longest.max((try line).len())
  }
  longest
}

fun main() throws IoError {
  val file = path.join(os.tempDir(), "veles-doc-lines.txt")
  try fs.writeFile(file, "short\nthe longest line here\nmid line\n")
  println("longest line: ${try longest(file)} bytes")
  try fs.remove(file)
}
```

Output:
```text
longest line: 21 bytes
```

## Binary data

A file format or a network header is numbers laid out in bytes, and the
order matters: *big endian* (`Be`, network order) puts the most significant byte
first, *little endian* (`Le`) the least. Every integer type converts to its
bytes and back, in both orders, as an `Array<u8, N>` of its width:

```veles
use io { println }

fun main() {
  val port = 8080.wrapU16()
  println("${port.toBeBytes()} ${port.toLeBytes()} ${u16.fromBeBytes([31, 144])}")

  // a header: a magic number, a version and a length
  val packet: MutableList<u8> = []
  packet.pushU32Be(0xCAFEBABE.wrapU32())
  packet.pushU16Le(2.wrapU16())
  packet.pushI64Be(-1)
  println("$packet")

  // read it back at offsets; null when the bytes are not all there
  println("${packet.readU32Be(0)} ${packet.readU16Le(4)} ${packet.readI64Be(6)} ${packet.readU32Be(12)}")
}
```

Output:
```text
[31, 144] [144, 31] 8080
[202, 254, 186, 190, 2, 0, 255, 255, 255, 255, 255, 255, 255, 255]
3405691582 2 -1 null
```

The readers (`readU16Be`, `readI32Le`, `readU64Be`, … for 16, 32 and 64
bits, signed and unsigned) are on `List<u8>` and return `null` for an
offset that is negative or too close to the end: no panic, because the bytes
come from outside. The writers (`pushU16Be`, `pushI64Le`, …) are on
`MutableList<u8>`. `isize` and `usize` are eight bytes, as the targets are
64-bit. `examples/binfile` writes and reads a small file format with these
and `writeAtomic`.

## Paths are text

`path` never touches the disk. It accepts both `/` and `\` and produces
`/`, which every platform's file API accepts.

```veles
use io, path

fun main() {
  val p = path.join("src", "compiler", "lexer.vs")
  io.println(p)
  io.println("${path.dir(p)} | ${path.base(p)} | ${path.stem(p)} | ${path.ext(p)}")
  io.println("${path.join("a", "/absolute")} ${path.isAbsolute("C:\\tools")} ${path.isAbsolute("tools")}")
}
```

Output:
```text
src/compiler/lexer.vs
src/compiler | lexer.vs | lexer | .vs
/absolute true false
```

`path.clean(p)` is the shortest path naming the same file: separators
collapse to one `/`, `.` goes, and `a/..` cancels. `path.within(root, p)`
answers whether `p`, cleaned, is `root` or something inside it — the
check to make before opening a file whose name came from outside:

```veles
use io, path

fun main() {
  io.println(path.clean("site/./css/../img//logo.png"))
  val root = "site"
  loop (asked in ["img/logo.png", "../secrets.txt", "img\\..\\..\\secrets.txt"]) {
    val file = path.join(root, asked)
    io.println("$asked -> ${if (path.within(root, file)) "serve $file" else "refuse"}")
  }
}
```

Output:
```text
site/img/logo.png
img/logo.png -> serve site/img/logo.png
../secrets.txt -> refuse
img\..\..\secrets.txt -> refuse
```

Both are lexical: they read the text, not the disk, so a symbolic link
inside `root` that points elsewhere is not detected. Both separators count
on every platform — the third request is a real escape on Windows, where
`\` separates, and it is refused on Linux too.

## Building text

`+` on strings copies both sides every time, so building a large file
with it is quadratic. `StringBuilder()` appends in place:

```veles
use io

fun main() {
  val out = StringBuilder()
  out.appendLine("name,square")
  loop (i in 1..3) { out.appendLine("$i,${i * i}") }
  io.print(out.toString())
  io.println("${out.len()} bytes")
}
```

Output:
```text
name,square
1,1
2,4
3,9
24 bytes
```

## The process

```veles
// fragment
use os
val args = os.args()                 // without the program name
val home = os.env("HOME") ?: "?"     // null when unset
val scratch = os.tempDir()           // TMPDIR or /tmp; TMP/TEMP on Windows
val who = "${os.pid()}@${try os.hostname()}"   // process id and host name, e.g. for a log line
os.exit(2)                           // flushes output first

val r = try os.run("clang", ["--version"])   // Output { code, stdout, stderr, ok() }
if (!r.ok()) io.eprintln("clang failed with ${r.code}: ${r.stderr}")
val applied = try os.run("git", ["apply", "-"], input: patch)   // patch on its standard input
```

`os.run` waits for the program and captures its standard output and,
apart, its standard error (`Output.stderr`). `stderr: os.Stderr.Merge`
puts both into `stdout`, in the order the program wrote them;
`os.Stderr.Inherit` lets its errors through to yours as it writes them.
`input:` is written to the program's standard input, which is then
closed; without it the program reads an empty input rather than waiting
on yours. The input goes in while the output comes out, so a program that
writes a lot before it has read everything cannot stall. A non-zero exit
is not an error — it is reported in `Output.code` — only a program that
cannot be started throws.

No shell runs in between. Each argument reaches the program as exactly
one argument, whatever it holds — `os.run("git", ["log", name])` is safe
with any `name`, including `x; rm -rf ~` or `$(curl ...)`, which arrive as
text. The program is looked up on `PATH`. When you want a shell's
features, run the shell and own the script: `os.run("sh", ["-c", script])`.
On Windows a `.bat`/`.cmd` file is refused, because `cmd.exe` re-reads its
arguments by rules no quoting survives; an argument holding a NUL byte is
refused everywhere (`IoKind.InvalidInput`).

## Time and randomness

```veles
// fragment
use time
val t = time.now()                       // Timestamp: microseconds since 1970-01-01T00:00:00Z
val d = t.utc()                          // DateTime: year, month, day, hour, minute, second, micros, offset
io.println("$d ${d.date()} ${t.local().hour}")   // 2026-09-17T12:34:56.789Z 2026-09-17 14
val sw = time.Stopwatch.start()
work()
io.println("took ${sw.elapsed()}")       // a Duration on the monotonic clock: "took 1.2ms"
```

The wall clock and the monotonic one are different types on purpose, and a
length of time is a third. [Time](20-time.md) is the chapter.

```veles
// fragment
use random
random.seed(42)                          // reproducible from here; unseeded, it starts from the clock
val n = random.range(1, 7)               // 1..<7
val x = random.float()                   // 0.0..<1.0
val pick = random.pick(names)            // T?, null when empty
random.shuffle(deck)                     // MutableList, in place
var rng = random.Rng.seeded(7)           // a generator of your own: the same methods, no lock
```

The module's generator is shared by every task and takes a lock on each
call, so a task drawing numbers in a hot loop is faster with an `Rng` of
its own. The generator is xoshiro256** — fast and good for games, tests
and sampling, not for secrets (`crypto.randomBytes` is for those).

Next: [Networking](16-networking.md) for TCP, or [Attributes and the
test runner](14-attributes-and-testing.md), or back to the
[index](index.md).
