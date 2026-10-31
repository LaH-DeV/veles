test "C strings preserve empty ASCII and Unicode text with a terminator" {
  loop (text in ["", "notes.db", "\u{E9}/\u{1F600}.txt"]) {
    with owned = try CString.of(text)
    // SAFETY: owned is open and contains a NUL-terminated UTF-8 copy.
    val copied = unsafe {
      readString(owned.ptr())
    }
    expect(copied == text)
    // SAFETY: the allocation includes the text and its trailing NUL.
    val bytes = unsafe {
      readBytes(owned.ptr(), text.len() + 1)
    }
    expect(bytes.len() == text.len() + 1)
    expect(bytes.at(text.len()) == 0)
    loop (index in 0..<text.len()) expect(bytes.at(index) == text.byteAt(index))
  }
}

test "NUL errors report the first byte offset including after Unicode" {
  val texts = ["\u{0}", "a\u{0}b\u{0}", "\u{E9}\u{0}tail", "\u{1F600}\u{0}"]
  loop ((text, offset) in texts.zip([0, 1, 2, 4])) {
    expect(when (CString.of(text)) {
      is Err(error) => error.at == offset &&
        error.message() == "a C string cannot hold a NUL byte (one is at byte $offset)"
      is Ok(owned)  => {
        owned.close()
        false
      }
    })
  }
}

test "copied text and bytes outlive the C string allocation" {
  val owned = try CString.of("hello")
  // SAFETY: owned is open and contains NUL-terminated UTF-8.
  val text = unsafe {
    readString(owned.ptr())
  }
  // SAFETY: the open allocation contains these five bytes.
  val bytes = unsafe {
    readBytes(owned.ptr(), 5)
  }
  owned.close()
  expect(text == "hello")
  expect(bytes.decodeUtf8() == "hello")
}

test "C string copies share closure and repeated close is harmless" {
  val owned = try CString.of("shared")
  val alias = owned
  alias.close()
  owned.close()
}

test "C string pointer access through an alias after close panics" {
  expectPanics(() => {
    val owned = require(CString.of("shared"))
    val alias = owned
    alias.close()
    owned.ptr()
  })
}

test "allocations are zeroed and empty byte copies are supported" {
  loop (length in [0, 1, 32, 257]) {
    val memory = alloc(length)
    // SAFETY: memory owns length zeroed bytes, including the zero-length case.
    val bytes = unsafe {
      readBytes(memory, length)
    }
    // SAFETY: reading zero bytes does not access the allocation.
    val empty = unsafe {
      readBytes(memory, 0)
    }
    // SAFETY: memory came from alloc and is freed exactly once.
    unsafe {
      free(memory)
    }
    expect(bytes.len() == length)
    expect(empty.isEmpty())
    loop (byte in bytes) expect(byte == 0)
  }
}

test "handles round trip integers text and nullable payloads" {
  with number = handle<i64>(9223372036854775807)
  // SAFETY: this open handle was created with the same payload type.
  expect(unsafe {
    Handle<i64>.from(number.ptr())
  } == 9223372036854775807)
  with text = handle("\u{E9}/payload")
  // SAFETY: this open handle contains a string.
  expect(unsafe {
    Handle<string>.from(text.ptr())
  } == "\u{E9}/payload")
  with absent = handle<i64?>(null)
  // SAFETY: this open handle contains a nullable i64.
  expect(unsafe {
    Handle<i64?>.from(absent.ptr())
  } == null)
}

test "handle copies share closure and repeated close is harmless" {
  val owned = handle("shared")
  val alias = owned
  // SAFETY: alias shares the open string handle.
  expect(unsafe {
    Handle<string>.from(alias.ptr())
  } == "shared")
  alias.close()
  owned.close()
}

test "handle pointer access through an alias after close panics" {
  expectPanics(() => {
    val owned = handle(42)
    val alias = owned
    alias.close()
    owned.ptr()
  })
}

test "C callbacks cannot recover a value through a closed handle" {
  expectPanics(() => {
    val owned = handle(42)
    val pointer = owned.ptr()
    owned.close()
    // SAFETY: the runtime checks this stale handle before any dereference.
    unsafe {
      Handle<i64>.from(pointer)
    }
  })
}

test "closing one handle does not invalidate another" {
  val first = handle("first")
  with second = handle("second")
  first.close()
  // SAFETY: second is still open and contains a string.
  expect(unsafe {
    Handle<string>.from(second.ptr())
  } == "second")
}
