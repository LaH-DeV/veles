// std/fs, std/path, std/os and StringBuilder: a program that touches the
// file system. Everything that can fail throws IoError (D4).
use fs, io { println }, os, path

fun main() throws IoError {
  val root = path.join(os.tempDir(), "veles-files-example")
  try fs.mkdir(path.join(root, "notes"))
  println("created: ${fs.isDir(root)} ${fs.isDir(path.join(root, "notes"))}")

  // build a report in linear time, then write it
  val report = StringBuilder()
  report.appendLine("# report")
  loop (i in 1..3) {
    report.appendLine("line $i")
  }
  val file = path.join(root, "notes", "report.md")
  try fs.writeFile(file, report.toString())
  try fs.appendFile(file, "done\n")

  val text = try fs.readFile(file)
  println("${text.lines().len()} lines, first: ${text.lines().first()}, last: ${text.lines().last()}")
  println("size ${text.len()} isFile ${fs.isFile(file)} exists ${fs.exists(path.join(root, "nope"))}")

  // paths are text
  println("${path.base(file)} ${path.ext(file)} ${path.stem(file)} ${path.base(path.dir(file))}")
  println("${path.join("a", "b/c")} ${path.join("a/", "b")} ${path.join("a", "/abs")} ${path.isAbsolute("C:\\x")}")

  try fs.writeFile(path.join(root, "notes", "a.txt"), "")
  println("listed: ${try fs.listDir(path.join(root, "notes"))}")
  try fs.rename(path.join(root, "notes", "a.txt"), path.join(root, "notes", "z.txt"))
  println("renamed: ${try fs.listDir(path.join(root, "notes"))}")

  // errors carry the path, the system's description and a portable kind
  when (fs.readFile(path.join(root, "missing.txt"))) {
    is Ok(t)  => println("unexpected: $t")
    is Err(e) => println("failed: ${e.detail} (${path.base(e.path)}) code ${e.code > 0} kind ${e.kind}")
  }
  // a missing file is a default; any other failure is still an error
  val settings = fs.readFile(path.join(root, "settings.toml")) ?? { e =>
    if (e.kind != IoKind.NotFound) throw e
    "# defaults"
  }
  println("settings: $settings")

  // clean up: files first, then the directories
  loop (name in try fs.listDir(path.join(root, "notes"))) {
    try fs.remove(path.join(root, "notes", name))
  }
  try fs.remove(path.join(root, "notes"))
  try fs.remove(root)
  println("cleaned: ${!fs.exists(root)}")

  // who and where this process is
  println("process: pid ${os.pid() > 0}, host named ${(try os.hostname()).len() > 0}, temp is a dir ${fs.isDir(os.tempDir())}")
}
