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
  val dir = path.join(os.env("TEMP") ?: os.env("TMPDIR") ?: "/tmp", "veles-tutorial-15")
  try fs.mkdir(dir)                       // parents too; fine if it exists
  val file = path.join(dir, "todo.txt")

  try fs.writeFile(file, "milk\n")        // replaces
  try fs.appendFile(file, "bread\n")      // creates when missing
  val items = (try fs.readFile(file)).lines()
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

An `IoError` carries what went wrong and where: `detail` is the system's
description, `path` the file, `code` the platform error number, and
`message()` combines the first two.

```veles
use fs, io

fun main() {
  when (val r = fs.readFile("no/such/file.txt")) {
    is Ok  => io.println("read ${r.len()} bytes")
    is Err => io.println("${r.message()} (code ${r.code > 0})")
  }
}
```

Output:
```text
No such file or directory: no/such/file.txt (code true)
```

`readFile` insists on UTF-8, because a string is always well-formed text
(D18); anything else is an `IoError` ("Illegal byte sequence"). For raw
data use the byte forms: `fs.readBytes(path)` gives a `List<u8>`,
`fs.writeBytes(path, bytes)` and `fs.appendBytes` take one, and
`text.bytes()` / `bytes.decodeUtf8()` convert. `io.readAll()` reads the
rest of standard input as text.

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

## Building text

`+` on strings copies both sides every time, so building a large file
with it is quadratic. `stringBuilder()` appends in place:

```veles
use io

fun main() {
  val out = stringBuilder()
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
os.exit(2)                           // flushes output first

val r = try os.run("clang", ["--version"])   // Output { code, stdout, ok() }
if (!r.ok()) io.eprintln("clang failed with ${r.code}")
```

`os.run` waits for the program and captures its standard output; its
standard error is passed through, or captured into the same text with
`os.run("clang", ["--version"], mergeStderr: true)`. A non-zero exit is
not an error — it is reported in `Output.code` — only a program that
cannot be started throws.

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
var rng = random.Rng.seeded(7)           // a generator of your own, with the same methods
```

Times are plain `i64` milliseconds, the unit `sleep` and timers already
use. The generator is xoshiro256** — fast and good for games, tests and
sampling, not for secrets.

Next: [Networking](16-networking.md) for TCP, or [Attributes and the
test runner](14-attributes-and-testing.md), or back to the
[index](index.md).
