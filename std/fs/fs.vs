/// Files and directories. Every failing call throws `IoError`.
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44).
use os, path as paths, time

extern "C" {
  fun veles_fs_open(path: string, mode: i64, handle: *raw i64): i64
  fun veles_fs_file_read_at(handle: i64, offset: i64, max: i64, out: *raw string): i64
  fun veles_fs_file_write(handle: i64, bytes: List<u8>): i64
  fun veles_fs_file_size(handle: i64, size: *raw i64): i64
  fun veles_fs_file_close(handle: i64): i64
  fun veles_fs_info(path: string, kind: *raw i64, size: *raw i64, mtime: *raw i64): i64
  fun veles_fs_read_file(path: string, out: *raw string): i64
  fun veles_fs_read_bytes(path: string, out: *raw string): i64
  fun veles_fs_write_bytes(path: string, bytes: List<u8>, append: bool): i64
  fun veles_fs_write_file(path: string, text: string): i64
  fun veles_fs_append_file(path: string, text: string): i64
  fun veles_fs_stat(path: string): i64
  fun veles_fs_lstat(path: string): i64
  fun veles_fs_list_dir(path: string, out: *raw string): i64
  fun veles_fs_mkdir(path: string): i64
  fun veles_fs_remove(path: string): i64
  fun veles_fs_rename(from: string, to: string): i64
  fun veles_fs_cwd(out: *raw string): i64
}

/// The whole file as text; a file that is not valid UTF-8 is an error.
public fun readFile(path: string): string throws IoError {
  try checkPath(path)
  var out = ""
  // SAFETY: takes `path` by value and stores the file's text in `out`, a local
  // that outlives the call; it keeps no pointer
  val code = unsafe {
    veles_fs_read_file(path, &out)
  }
  if (code != 0) throw os.ioError(code, path)
  out
}

/// The whole file as bytes.
public fun readBytes(path: string): List<u8> throws IoError {
  try checkPath(path)
  var data = ""
  // SAFETY: takes `path` by value and stores the bytes in `data`, a local that
  // outlives the call; it keeps no pointer
  val code = unsafe {
    veles_fs_read_bytes(path, &data)
  }
  if (code != 0) throw os.ioError(code, path)
  data.bytes()
}

/// Writes `bytes` to `path`, replacing the file.
public fun writeBytes(path: string, bytes: List<u8>) throws IoError {
  try checkPath(path)
  // SAFETY: reads `bytes` within its length and keeps nothing after the call
  val code = unsafe {
    veles_fs_write_bytes(path, bytes, false)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Appends `bytes` to `path`, creating the file when missing.
public fun appendBytes(path: string, bytes: List<u8>) throws IoError {
  try checkPath(path)
  // SAFETY: reads `bytes` within its length and keeps nothing after the call
  val code = unsafe {
    veles_fs_write_bytes(path, bytes, true)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Writes `text` to `path`, replacing the file.
public fun writeFile(path: string, text: string) throws IoError {
  try checkPath(path)
  // SAFETY: takes `path` and `text` by value; it keeps nothing after the call
  val code = unsafe {
    veles_fs_write_file(path, text)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Appends `text` to `path`, creating the file when missing.
public fun appendFile(path: string, text: string) throws IoError {
  try checkPath(path)
  // SAFETY: takes `path` and `text` by value; it keeps nothing after the call
  val code = unsafe {
    veles_fs_append_file(path, text)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// True when a file or directory exists at `path`.
public fun exists(path: string): bool = statKind(path) != 0

/// True when `path` is a directory.
public fun isDir(path: string): bool = statKind(path) == 2

/// True when `path` is a regular file.
public fun isFile(path: string): bool = statKind(path) == 1

/// What the file system says about a path, following symbolic links.
public struct Stat {
  /// The length in bytes (of a directory, whatever the system reports).
  public size: i64
  /// The last write, to the microsecond where the file system keeps that
  /// much (whole seconds on some).
  public modified: time.Timestamp
  public isDir:    bool

  /// Anything that is not a directory.
  public fun isFile(): bool = !this.isDir
}

/// Size, last-write time and kind of the entry at `path`; a missing one is
/// an `IoError` (`kind` NotFound). `isFile` and `isDir` answer only the kind
/// and never throw.
public fun stat(path: string): Stat throws IoError {
  try checkPath(path)
  var kind: i64 = 0
  var size: i64 = 0
  var mtime: i64 = 0
  // SAFETY: takes `path` by value and stores three numbers in locals that
  // outlive the call; it keeps no pointer
  val code = unsafe {
    veles_fs_info(path, &kind, &size, &mtime)
  }
  if (code != 0) throw os.ioError(code, path)
  Stat(size, modified: time.Timestamp.ofMicros(mtime), isDir: kind == 2)
}

/// How `open` opens a file.
public enum FileMode {
  /// An existing file, for `read` and `readAt`.
  Read
  /// A new file, or an existing one emptied, for `write`.
  Write
  /// A file created when missing, for `write` at its end.
  Append
}

/// An open file, for what does not fit in memory or should not be held
/// whole: reading a piece at a time, or writing as the data arrives. Close it
/// with `with` or `close()`. A file opened to read cannot be written, and the
/// other way round; every failure throws `IoError`.
///
/// ```veles
/// with f = try fs.open("big.bin")              // closed when the block ends
/// val header = try f.readAt(0, 16)             // any place, in any order
/// loop {
///   val chunk = try f.read(65536)              // then from where the last read ended
///   if (chunk.isEmpty()) break
/// }
/// ```
public struct File {
  private handle: i64
  private path:   string
  private next:   Atomic<i64> = Atomic(value: 0)
  private isOpen: Atomic<bool> = Atomic(value: true)

  /// The length in bytes, now.
  public fun size(): i64 throws IoError {
    try this.check()
    var size: i64 = 0
    // SAFETY: stores the length in `size`, a local that outlives the call
    val code = unsafe {
      veles_fs_file_size(this.handle, &size)
    }
    if (code != 0) throw os.ioError(code, this.path)
    size
  }

  /// Up to `max` bytes from where the last `read` ended; empty at the end of
  /// the file. Fewer than `max` is normal and does not mean the end.
  public fun read(max: i64 = 65536): List<u8> throws IoError {
    val at = this.next.load()
    val chunk = try this.readAt(at, max)
    this.next.store(at + chunk.len())
    chunk
  }

  /// Up to `max` bytes at `offset`, whatever `read` has done; empty at or past
  /// the end.
  public fun readAt(offset: i64, max: i64): List<u8> throws IoError {
    try this.check()
    var data = ""
    // SAFETY: stores the bytes in `data`, a local that outlives the call; the
    // handle is open (checked above) and nothing else closes it during the call
    val code = unsafe {
      veles_fs_file_read_at(this.handle, offset, max, &data)
    }
    if (code != 0) throw os.ioError(code, this.path)
    data.bytes()
  }

  /// All of `bytes`, at the end of what was written so far.
  public fun write(bytes: List<u8>) throws IoError {
    try this.check()
    // SAFETY: reads `bytes` within its length and keeps nothing after the call
    val code = unsafe {
      veles_fs_file_write(this.handle, bytes)
    }
    if (code != 0) throw os.ioError(code, this.path)
  }

  fun check() throws IoError {
    if (!this.isOpen.load()) throw IoError(path: this.path, code: 9, detail: "the file is closed", kind: IoKind.InvalidInput)
  }

  implement Closeable {
    fun close() {
      if (!this.isOpen.swap(false)) return
      // SAFETY: the swap lets exactly one caller through, so the handle is
      // closed once
      val _ = unsafe {
        veles_fs_file_close(this.handle)
      }
    }
  }
}

/// Opens `path`. `Read` needs the file to exist; `Write` creates it or
/// empties it; `Append` creates it or keeps what it holds.
public fun open(path: string, mode: FileMode = FileMode.Read): File throws IoError {
  try checkPath(path)
  var handle: i64 = 0
  val m = when (mode) {
    FileMode.Read   => 0
    FileMode.Write  => 1
    FileMode.Append => 2
  }
  // SAFETY: takes `path` by value and stores the handle in `handle`, a local
  // that outlives the call
  val code = unsafe {
    veles_fs_open(path, m, &handle)
  }
  if (code != 0) throw os.ioError(code, path)
  File(handle, path)
}

// 0 when nothing is there (or the path cannot name anything), 1 a file,
// 2 a directory
fun statKind(path: string): i64 {
  if (path.contains("\u{0}")) return 0
  // SAFETY: takes `path` by value and returns a number; it keeps nothing
  unsafe {
    veles_fs_stat(path)
  }
}

// A path holding a NUL byte would reach the system cut short at the NUL —
// naming a different file from the one the caller checked (`root/..\0/x`
// passes a `..` check as a segment and opens `root/..`) — so it is refused,
// as Go and Rust refuse it.
fun checkPath(path: string) throws IoError {
  if (path.contains("\u{0}")) {
    throw IoError(path: path.replace("\u{0}", "\\0"), code: 22, detail: "a path cannot hold a NUL byte", kind: IoKind.InvalidInput)
  }
}

/// The names in a directory (not paths), sorted.
public fun listDir(path: string): List<string> throws IoError {
  try checkPath(path)
  var out = ""
  // SAFETY: takes `path` by value and stores the names in `out`, a local that
  // outlives the call; it keeps no pointer
  val code = unsafe {
    veles_fs_list_dir(path, &out)
  }
  if (code != 0) throw os.ioError(code, path)
  if (out.isEmpty()) return []
  out.split("\n").sorted()
}

/// Every file under `root`, recursively: full paths (joined onto `root`),
/// depth-first, each directory's entries in name order — so the same tree
/// always walks the same way. Directories themselves are not in the list.
///
/// Symbolic links are not followed below `root`: a link to a file is listed,
/// a link to a directory (or, on Windows, a junction) is neither listed nor
/// entered, so a link that loops back cannot make the walk endless. `root`
/// itself may be a link, and a file (then the list is just `root`). A
/// directory that cannot be read ends the walk with its `IoError`.
public fun walk(root: string): List<string> throws IoError {
  try checkPath(root)
  if (isFile(root)) return [root]
  var out: MutableList<string> = []
  // paths still to visit, the next one last: an explicit stack rather than
  // recursion, so a deep tree cannot overflow the call stack
  var pending: MutableList<string> = []
  try pushEntries(root, pending)
  loop {
    val p = pending.pop() ?: break
    // SAFETY: takes `p` by value and returns a number; it keeps nothing
    val kind = unsafe {
      veles_fs_lstat(p)
    }
    when (kind) {
      2    => try pushEntries(p, pending)
      3    => if (!isDir(p)) out.push(p)
      else => out.push(p)
    }
  }
  out.toList()
}

// pushEntries puts a directory's entries on the walk's stack in reverse, so
// they come off in name order.
fun pushEntries(dir: string, pending: MutableList<string>) throws IoError {
  val names = try listDir(dir)
  loop (name in names.reversed()) pending.push(paths.join(dir, name))
}

/// Creates the directory and any missing parents.
public fun mkdir(path: string) throws IoError {
  try checkPath(path)
  val parent = paths.dir(path)
  if (!parent.isEmpty() && parent != path && !exists(parent)) try mkdir(parent)
  // SAFETY: takes `path` by value and returns an error code; it keeps nothing
  val code = unsafe {
    veles_fs_mkdir(path)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Removes a file or an empty directory.
public fun remove(path: string) throws IoError {
  try checkPath(path)
  // SAFETY: takes `path` by value and returns an error code; it keeps nothing
  val code = unsafe {
    veles_fs_remove(path)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Renames (moves) `from` to `to`.
public fun rename(from: string, to: string) throws IoError {
  try checkPath(from)
  try checkPath(to)
  // SAFETY: takes both paths by value and returns an error code; it keeps nothing
  val code = unsafe {
    veles_fs_rename(from, to)
  }
  if (code != 0) throw os.ioError(code, from)
}

/// The current working directory.
public fun cwd(): string throws IoError {
  var out = ""
  // SAFETY: stores the directory in `out`, a local that outlives the call
  val code = unsafe {
    veles_fs_cwd(&out)
  }
  if (code != 0) throw os.ioError(code, "")
  out
}
