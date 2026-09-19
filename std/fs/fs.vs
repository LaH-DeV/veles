/// Files and directories. Every failing call throws `IoError`.
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44).
use os, path as paths

extern "C" {
  fun veles_fs_read_file(path: string, out: *raw string): i64
  fun veles_fs_read_bytes(path: string, out: *raw string): i64
  fun veles_fs_write_bytes(path: string, bytes: List<u8>, append: bool): i64
  fun veles_fs_write_file(path: string, text: string): i64
  fun veles_fs_append_file(path: string, text: string): i64
  fun veles_fs_stat(path: string): i64
  fun veles_fs_list_dir(path: string, out: *raw string): i64
  fun veles_fs_mkdir(path: string): i64
  fun veles_fs_remove(path: string): i64
  fun veles_fs_rename(from: string, to: string): i64
  fun veles_fs_cwd(out: *raw string): i64
}

/// The whole file as text; a file that is not valid UTF-8 is an error.
public fun readFile(path: string): string throws IoError {
  var out = ""
  val code = unsafe {
    veles_fs_read_file(path, &out)
  }
  if (code != 0) throw os.ioError(code, path)
  out
}

/// The whole file as bytes.
public fun readBytes(path: string): List<u8> throws IoError {
  var data = ""
  val code = unsafe {
    veles_fs_read_bytes(path, &data)
  }
  if (code != 0) throw os.ioError(code, path)
  data.bytes()
}

/// Writes `bytes` to `path`, replacing the file.
public fun writeBytes(path: string, bytes: List<u8>) throws IoError {
  val code = unsafe {
    veles_fs_write_bytes(path, bytes, false)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Appends `bytes` to `path`, creating the file when missing.
public fun appendBytes(path: string, bytes: List<u8>) throws IoError {
  val code = unsafe {
    veles_fs_write_bytes(path, bytes, true)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Writes `text` to `path`, replacing the file.
public fun writeFile(path: string, text: string) throws IoError {
  val code = unsafe {
    veles_fs_write_file(path, text)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Appends `text` to `path`, creating the file when missing.
public fun appendFile(path: string, text: string) throws IoError {
  val code = unsafe {
    veles_fs_append_file(path, text)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// True when a file or directory exists at `path`.
public fun exists(path: string): bool = unsafe {
  veles_fs_stat(path)
} != 0

/// True when `path` is a directory.
public fun isDir(path: string): bool = unsafe {
  veles_fs_stat(path)
} == 2

/// True when `path` is a regular file.
public fun isFile(path: string): bool = unsafe {
  veles_fs_stat(path)
} == 1

/// The names in a directory (not paths), sorted.
public fun listDir(path: string): List<string> throws IoError {
  var out = ""
  val code = unsafe {
    veles_fs_list_dir(path, &out)
  }
  if (code != 0) throw os.ioError(code, path)
  if (out.isEmpty()) return []
  out.split("\n").sorted()
}

/// Creates the directory and any missing parents.
public fun mkdir(path: string) throws IoError {
  val parent = paths.dir(path)
  if (!parent.isEmpty() && parent != path && !exists(parent)) try mkdir(parent)
  val code = unsafe {
    veles_fs_mkdir(path)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Removes a file or an empty directory.
public fun remove(path: string) throws IoError {
  val code = unsafe {
    veles_fs_remove(path)
  }
  if (code != 0) throw os.ioError(code, path)
}

/// Renames (moves) `from` to `to`.
public fun rename(from: string, to: string) throws IoError {
  val code = unsafe {
    veles_fs_rename(from, to)
  }
  if (code != 0) throw os.ioError(code, from)
}

/// The current working directory.
public fun cwd(): string throws IoError {
  var out = ""
  val code = unsafe {
    veles_fs_cwd(&out)
  }
  if (code != 0) throw os.ioError(code, "")
  out
}
