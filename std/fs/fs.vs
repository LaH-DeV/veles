/// Files and directories. Every failing call throws `IoError`.
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44).
use io, os, path as paths, time

extern "C" {
  fun veles_fs_open(path: string, mode: i64, handle: *raw i64): i64
  fun veles_fs_file_read_at(handle: i64, offset: i64, max: i64, out: *raw string): i64
  fun veles_fs_file_write(handle: i64, bytes: List<u8>, append: bool): i64
  fun veles_fs_file_size(handle: i64, size: *raw i64): i64
  fun veles_fs_file_close(handle: i64): i64
  fun veles_fs_file_sync(handle: i64): i64
  fun veles_fs_file_seek(handle: i64, offset: i64): i64
  fun veles_fs_file_lock(handle: i64, wait: bool): i64
  fun veles_fs_file_unlock(handle: i64): i64
  fun veles_fs_write_atomic(path: string, bytes: List<u8>): i64
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
  private append: bool
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

  /// Moves the position to `offset` bytes from the start: the next `read`
  /// continues there, and so does the next `write` of a file opened with
  /// `Write` (one opened to `Append` always writes at its end). A negative
  /// offset panics; one past the end reads nothing, and a write there leaves
  /// a gap that reads as zeros.
  public fun seek(offset: i64) throws IoError {
    if (offset < 0) panic("File.seek: the offset is $offset; it counts from the start of the file")
    try this.check()
    // SAFETY: takes the open handle and a number; keeps nothing
    val code = unsafe {
      veles_fs_file_seek(this.handle, offset)
    }
    if (code != 0) throw os.ioError(code, this.path)
    this.next.store(offset)
  }

  /// Waits until what was written is on the disk, not only in the system's
  /// cache.
  public fun sync() throws IoError {
    try this.check()
    // SAFETY: takes the open handle; keeps nothing
    val code = unsafe {
      veles_fs_file_sync(this.handle)
    }
    if (code != 0) throw os.ioError(code, this.path)
  }

  /// An exclusive lock on the whole file, waiting until it is free. Advisory:
  /// it keeps out other programs that ask for the lock too (`flock`,
  /// `LockFileEx`), not ones that just open the file. Hold it with `with`; it
  /// is let go when the block ends, and when the file closes.
  ///
  /// ```veles
  /// with file = try fs.open("app.pid", fs.FileMode.Append)
  /// with lock = try file.lock()
  /// // only one process is here at a time
  /// ```
  public fun lock(): Lock throws IoError {
    try this.check()
    // SAFETY: takes the open handle and a flag; the wait does not touch memory
    val code = unsafe {
      veles_fs_file_lock(this.handle, true)
    }
    if (code != 0) throw os.ioError(code, this.path)
    Lock(handle: this.handle, path: this.path)
  }

  /// `lock()` without waiting: `null` when someone else holds the lock.
  public fun tryLock(): Lock? throws IoError {
    try this.check()
    // SAFETY: takes the open handle and a flag; keeps nothing
    val code = unsafe {
      veles_fs_file_lock(this.handle, false)
    }
    if (code == -1) return null
    if (code != 0) throw os.ioError(code, this.path)
    Lock(handle: this.handle, path: this.path)
  }

  implement io.Stream {
    /// Up to `max` bytes from where the last `read` ended; empty at the end of
    /// the file. Fewer than `max` is normal and does not mean the end.
    fun read(max: i64 = 65536): List<u8> throws IoError {
      val at = this.next.load()
      val chunk = try this.readAt(at, max)
      this.next.store(at + chunk.len())
      chunk
    }

    /// All of `bytes`, at the end of what was written so far.
    fun write(bytes: List<u8>) throws IoError {
      try this.check()
      // SAFETY: reads `bytes` within its length and keeps nothing after the call
      val code = unsafe {
        veles_fs_file_write(this.handle, bytes, this.append)
      }
      if (code != 0) throw os.ioError(code, this.path)
    }

    /// Exactly `n` bytes, or fewer when the file ends first.
    fun readExact(n: i64): List<u8> throws IoError {
      val out: MutableList<u8> = []
      loop (out.len() < n) {
        val chunk = try this.read((n - out.len()).min(65536))
        if (chunk.isEmpty()) break
        out.addAll(chunk)
      }
      out
    }

    /// The next line from where the last read ended, without its `\n` (and a
    /// `\r` before it), or `null` at the end of the file; a line that is not
    /// valid UTF-8 is an error, and one longer than `max` bytes throws `TooLong`.
    fun readLine(max: i64): string? throws IoError | io.TooLong {
      val start = this.next.load()
      var at = start
      val line: MutableList<u8> = []
      loop {
        val chunk = try this.readAt(at, 4096)
        if (chunk.isEmpty()) {
          // the end of the file: what is left is the last line
          if (line.isEmpty()) return null
          this.next.store(at)
          break
        }
        var cut = -1
        loop (i in 0..<chunk.len()) {
          if (chunk.at(i) == 10) {
            cut = i
            break
          }
        }
        if (cut < 0) {
          line.addAll(chunk)
          at += chunk.len()
          if (line.len() > max) throw this.tooLong(max)
          continue
        }
        line.addAll(chunk.take(cut))
        this.next.store(at + cut + 1)
        break
      }
      if (line.len() > max) throw this.tooLong(max)
      if (!line.isEmpty() && line.at(line.len() - 1) == 13) line.removeAt(line.len() - 1)
      line.toList().decodeUtf8() ?: throw IoError(path: this.path, code: 0, detail: "line is not valid UTF-8", kind: IoKind.InvalidData)
    }

    /// `text` as UTF-8, at the end of what was written so far.
    fun writeText(text: string) throws IoError {
      try this.write(text.bytes())
    }

    /// Does nothing: a file has no other side to tell.
    fun shutdownWrite() throws IoError { }
  }

  fun tooLong(max: i64): io.TooLong =
    io.TooLong(message: "a line of ${this.path} is longer than $max bytes", limit: max)

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

/// A held file lock (`File.lock`). Closing it lets go; it does nothing
/// after the file itself is closed, which has let go already.
public struct Lock {
  private handle: i64
  private path:   string
  private held:   Atomic<bool> = Atomic(value: true)

  implement Closeable {
    fun close() {
      if (!this.held.swap(false)) return
      // SAFETY: the swap lets one caller through; an already closed handle just
      // answers with an error that is ignored
      val _ = unsafe {
        veles_fs_file_unlock(this.handle)
      }
    }
  }
}

/// Replaces the contents of `path` with `bytes` so that no reader and no crash
/// ever sees half of it: the bytes go to a new file beside it, reach the
/// disk, and the file is renamed over `path` (on POSIX the directory is
/// synced as well). An existing file keeps its permissions. When it fails the
/// old contents are still there and the new file is removed. On Windows a
/// file that another handle has open cannot be replaced: close it first.
public fun writeAtomic(path: string, bytes: List<u8>) throws IoError {
  try checkPath(path)
  // SAFETY: reads `bytes` within its length and keeps nothing after the call
  val code = unsafe {
    veles_fs_write_atomic(path, bytes)
  }
  if (code != 0) throw os.ioError(code, path)
}

//// The lines of a file, one at a time, as `fs.lines` hands them out. Each
/// item is the line (without its `\n`, and a `\r` before it) or the error that
/// stopped the reading: a read that failed, a line that is not UTF-8, or a
/// line longer than `max` bytes (an `IoError` of kind `InvalidData`). After an error, or at the end
/// of the file, there are no more items and the file is closed. To stop
/// early, `with` closes it.
public struct Lines {
  private file:     File
  private path:     string
  private max:      i64
  private var over: bool = false

  implement Iterator {
    type Item = Result<string, IoError>
    fun next(): Result<string, IoError>? {
      if (this.over) return null
      val read = this.file.readLine(this.max)
      when (read) {
        is Ok(line) => {
          if (line == null) {
            this.close()
            return null
          }
          val item: Result<string, IoError> = Ok(line)
          item
        }
        is Err(e)   => {
          this.close()
          val failure: IoError = when (e) {
            is io.TooLong => IoError(path: this.path, code: 0, detail: e.message(), kind: IoKind.InvalidData)
            is IoError    => e
          }
          val item: Result<string, IoError> = Err(failure)
          item
        }
      }
    }
  }

  implement Closeable {
    fun close() {
      this.over = true
      this.file.close()
    }
  }
}

/// The lines of `path`, read as they are asked for — memory holds one line,
/// however big the file is. A line longer than `max` bytes is an error item,
/// not an allocation. Each item is a `Result`; `try line` passes its error on:
///
/// ```veles
/// loop (line in try fs.lines("access.log", max: 8192)) {
///   val text = try line
///   handle(text)
/// }
/// ```
public fun lines(path: string, max: i64): Lines throws IoError {
  if (max <= 0) panic("fs.lines: max is $max; it is the longest line, in bytes, that is accepted")
  Lines(file: try open(path), path, max)
}

/// Copies the file `from` to `to`, replacing it, a piece at a time — memory
/// stays small however big the file is. Copying a file onto itself does
/// nothing.
public fun copy(from: string, to: string) throws IoError {
  if (from == to) return
  with source = try open(from)
  with target = try open(to, FileMode.Write)
  loop {
    val chunk = try source.read(65536)
    if (chunk.isEmpty()) break
    try target.write(chunk)
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
  File(handle, path, append: mode == FileMode.Append)
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
