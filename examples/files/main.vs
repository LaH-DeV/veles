// std/fs, std/path, std/os and StringBuilder: a program that touches the
// file system. Everything that can fail throws IoError (D4).
use io
use os
use fs
use path

fun main() throws IoError {
  val root = path.join(os.env("TEMP") ?: os.env("TMPDIR") ?: "/tmp", "veles-files-example")
  try fs.mkdir(path.join(root, "notes"))
  io.println("created: ${fs.isDir(root)} ${fs.isDir(path.join(root, "notes"))}")

  // build a report in linear time, then write it
  val report = stringBuilder()
  report.appendLine("# report")
  loop (i in 1..3) {
    report.appendLine("line $i")
  }
  val file = path.join(root, "notes", "report.md")
  try fs.writeFile(file, report.toString())
  try fs.appendFile(file, "done\n")

  val text = try fs.readFile(file)
  io.println("${text.lines().len()} lines, first: ${text.lines().first()}, last: ${text.lines().last()}")
  io.println("size ${text.len()} isFile ${fs.isFile(file)} exists ${fs.exists(path.join(root, "nope"))}")

  // paths are text
  io.println("${path.base(file)} ${path.ext(file)} ${path.stem(file)} ${path.base(path.dir(file))}")
  io.println("${path.join("a", "b/c")} ${path.join("a/", "b")} ${path.join("a", "/abs")} ${path.isAbsolute("C:\\x")}")

  try fs.writeFile(path.join(root, "notes", "a.txt"), "")
  io.println("listed: ${try fs.listDir(path.join(root, "notes"))}")
  try fs.rename(path.join(root, "notes", "a.txt"), path.join(root, "notes", "z.txt"))
  io.println("renamed: ${try fs.listDir(path.join(root, "notes"))}")

  // errors carry the path and the system's description
  when (fs.readFile(path.join(root, "missing.txt"))) {
    is Ok(t)  => io.println("unexpected: $t")
    is Err(e) => io.println("failed: ${e.detail} (${path.base(e.path)}) code ${e.code > 0}")
  }

  // clean up: files first, then the directories
  loop (name in try fs.listDir(path.join(root, "notes"))) {
    try fs.remove(path.join(root, "notes", name))
  }
  try fs.remove(path.join(root, "notes"))
  try fs.remove(root)
  io.println("cleaned: ${!fs.exists(root)}")
}
