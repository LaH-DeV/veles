# 15. Files, paths and processes

Three small modules turn Veles from a calculator into a tool: `fs` reads
and writes files, `path` manipulates file names as text, and `os` gives
the program its arguments, its environment and other programs. None of
them is in scope by default — `use` the ones you need.

## Reading and writing files

Every `fs` call that can fail throws `IoError`, so the rules of
[chapter 7](07-errors.md) apply: `try` to propagate, `when` to handle.

```veles
use io
use os
use fs
use path

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
use io
use fs

fun main() {
  when (fs.readFile("no/such/file.txt")) {
    is Ok(text) => io.println("read ${text.len()} bytes")
    is Err(e) => io.println("${e.message()} (code ${e.code > 0})")
  }
}
```

Output:
```text
No such file or directory: no/such/file.txt (code true)
```

## Paths are text

`path` never touches the disk. It accepts both `/` and `\` and produces
`/`, which every platform's file API accepts.

```veles
use io
use path

fun main() {
  val p = path.joinAll(["src", "compiler", "lexer.vs"])
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
standard error is passed through. A non-zero exit is not an error — it is
reported in `Output.code` — only a program that cannot be started throws.

Next: [Attributes and the test runner](14-attributes-and-testing.md), or
back to the [index](index.md).
