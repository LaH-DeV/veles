/// Files and directories. Every failing call throws `IoError`.
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44).
use os
use path.{ dir }

extern "C" {
  fun veles_fs_read_file(path: string, out: *raw string): i64
  fun veles_fs_write_file(path: string, text: string): i64
  fun veles_fs_append_file(path: string, text: string): i64
  fun veles_fs_stat(path: string): i64
  fun veles_fs_list_dir(path: string, out: *raw string): i64
  fun veles_fs_mkdir(path: string): i64
  fun veles_fs_remove(path: string): i64
  fun veles_fs_rename(from: string, to: string): i64
  fun veles_fs_cwd(out: *raw string): i64
}

/// The whole file as text.
pub fun readFile(path: string): string throws IoError {
  var out = ""
  val code = unsafe {
    veles_fs_read_file(path, &out)
  }
  if (code != 0) throw os.ioError(code, path)
  out
}

/// Writes `text` to `path`, replacing the file.
pub fun writeFile(path: string, text: string) throws IoError {
  val code = unsafe {
    veles_fs_write_file(path, text)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Appends `text` to `path`, creating the file when missing.
pub fun appendFile(path: string, text: string) throws IoError {
  val code = unsafe {
    veles_fs_append_file(path, text)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// True when a file or directory exists at `path`.
pub fun exists(path: string): bool = unsafe {
  veles_fs_stat(path)
} != 0

/// True when `path` is a directory.
pub fun isDir(path: string): bool = unsafe {
  veles_fs_stat(path)
} == 2

/// True when `path` is a regular file.
pub fun isFile(path: string): bool = unsafe {
  veles_fs_stat(path)
} == 1

/// The names in a directory (not paths), sorted.
pub fun listDir(path: string): List<string> throws IoError {
  var out = ""
  val code = unsafe {
    veles_fs_list_dir(path, &out)
  }
  if (code != 0) throw os.ioError(code, path)
  if (out.isEmpty()) return []
  out.split("\n").sorted()
}

/// Creates the directory and any missing parents.
pub fun mkdir(path: string) throws IoError {
  val parent = dir(path)
  if (!parent.isEmpty() && parent != path && !exists(parent)) try mkdir(parent)
  val code = unsafe {
    veles_fs_mkdir(path)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Removes a file or an empty directory.
pub fun remove(path: string) throws IoError {
  val code = unsafe {
    veles_fs_remove(path)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Renames (moves) `from` to `to`.
pub fun rename(from: string, to: string) throws IoError {
  val code = unsafe {
    veles_fs_rename(from, to)
  }
  if (code != 0) throw os.ioError(code, from)
}

/// The current working directory.
pub fun cwd(): string throws IoError {
  var out = ""
  val code = unsafe {
    veles_fs_cwd(&out)
  }
  if (code != 0) throw os.ioError(code, "")
  out
}
