/// Crossing into C (D69).
///
/// The collector's memory never becomes a raw pointer that C may keep:
/// what C keeps is copied into memory the collector does not manage and
/// freed explicitly — a `CString` by `with` — and what C only looks at for
/// the length of a call can be lent without a copy (`List.withRaw`, in the
/// prelude). A Veles value C must hand back later, as the `void *userdata`
/// of a callback, travels as a `Handle`.
///
/// ```veles
/// with (path = try ffi.CString.of("notes.db")) {
///   unsafe { sqlite3_open(path.ptr(), &db) }
/// }
/// ```
///
/// Reading through a raw pointer is `unsafe`, so `readString` and
/// `readBytes` are `unsafe fun`s: they trust the pointer they are given.

extern "C" {
  fun veles_ffi_cstring(s: string): *raw u8
  fun veles_ffi_nul_at(s: string): i64
  fun veles_ffi_from_cstring(p: *raw u8, out: *raw string)
  fun veles_ffi_bytes(p: *raw u8, n: i64, out: *raw string)
  fun veles_ffi_alloc(n: i64): *raw u8
  fun veles_ffi_free(p: *raw u8)
  fun veles_ffi_handle_new(box: *raw u8): *raw u8
  fun veles_ffi_handle_get(h: *raw u8): *raw u8
  fun veles_ffi_handle_release(h: *raw u8)
}

/// A Veles string holding a NUL byte cannot become a C string: C would
/// read it as ending there, which is how a checked name turns into a
/// different, unchecked one.
public error NulByte {
  /// The byte offset of the first NUL.
  public at: i64
  fun message(): string = "a C string cannot hold a NUL byte (one is at byte ${this.at})"
}

struct CState {
  var ptr: (*raw u8)?
}

/// A NUL-terminated copy of a string in C's own memory, for a `const char *`
/// parameter. It is freed by `close`, so it belongs in a `with`; copies of
/// a `CString` share one buffer, and closing twice is harmless.
public struct CString {
  private state: *CState

  /// Copies `s` for C. A string holding a NUL byte is refused (`NulByte`).
  public static fun of(s: string): CString throws NulByte {
    val at = unsafe {
      veles_ffi_nul_at(s)
    }
    if (at >= 0) throw NulByte(at)
    val state = CState(ptr: unsafe {
      veles_ffi_cstring(s)
    })
    CString(state: &state)
  }

  /// The `char *` to pass. Reading it after `close` is a bug, and panics.
  public fun ptr(): *raw u8 = this.state.ptr ?: panic("a CString was used after close")

  implement Closeable {
    fun close() {
      val p = this.state.ptr ?: return
      this.state.ptr = null
      unsafe {
        veles_ffi_free(p)
      }
    }
  }
}

/// The C text at `p`, up to its NUL, copied into a Veles string. The bytes
/// are taken as they are; a C library that does not promise UTF-8 may
/// hand back something `utf8` would refuse.
public unsafe fun readString(p: *raw u8): string {
  var s = ""
  unsafe {
    veles_ffi_from_cstring(p, &s)
  }
  s
}

/// `n` bytes from `p`, copied.
public unsafe fun readBytes(p: *raw u8, n: i64): List<u8> {
  var s = ""
  unsafe {
    veles_ffi_bytes(p, n, &s)
  }
  s.bytes()
}

/// `n` zeroed bytes of memory the collector does not manage — for a C API
/// that keeps a buffer past the call. Give it back with `free`.
public fun alloc(n: i64): *raw u8 = unsafe {
  veles_ffi_alloc(n)
}

/// Returns memory from `alloc` (or from C's `malloc`) to C.
public unsafe fun free(p: *raw u8) {
  unsafe {
    veles_ffi_free(p)
  }
}

/// A Veles value lent to C as an opaque pointer — the `void *userdata` a C
/// API passes back to a callback. The value stays alive while the handle
/// is open, and the pointer C holds is an index, not an address the
/// collector owns. Close it when C is done with it.
///
/// ```veles
/// with (h = ffi.handle(counter)) {
///   unsafe { visit(tree, &onNode, h.ptr()) }
/// }
/// extern "C" fun onNode(node: *raw u8, data: *raw u8): i32 {
///   val counter = unsafe { ffi.Handle<Counter>.from(data) }
///   ...
/// }
/// ```
public struct Handle<T> {
  private state: *CState

  /// The `void *` to give C.
  public fun ptr(): *raw u8 = this.state.ptr ?: panic("a Handle was used after close")

  /// The value behind a pointer a `Handle<T>` gave C. It trusts that the
  /// pointer came from a `Handle` of this `T`; one that was closed panics.
  public static unsafe fun from(p: *raw u8): T {
    val box = unsafe {
      veles_ffi_handle_get(p)
    }
    unsafe {
      *(box as *raw T)
    }
  }

  implement Closeable {
    fun close() {
      val h = this.state.ptr ?: return
      this.state.ptr = null
      unsafe {
        veles_ffi_handle_release(h)
      }
    }
  }
}

/// Lends `value` to C as a `Handle`.
public fun handle<T>(value: T): Handle<T> {
  var held = value
  val box: *raw T = unsafe {
    &held
  }
  val h = unsafe {
    veles_ffi_handle_new(box as *raw u8)
  }
  val state = CState(ptr: h)
  Handle(state: &state)
}
