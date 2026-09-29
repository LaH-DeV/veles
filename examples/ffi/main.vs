// Calling C, and being called by it (D67, D69), with nothing but the C
// library: strings copied into C's memory, a list lent without a copy, a
// Veles comparator handed to `qsort`, and a Veles value travelling through
// C as `void *userdata`.
use ffi, io { println }

extern "C" {
  fun strlen(s: *raw u8): u64
  fun strtoll(s: *raw u8, end: (*raw u8)?, base: i32): i64  // long long: 64 bits everywhere, unlike long
  fun memset(p: *raw u8, c: i32, n: u64): *raw u8
  fun qsort(base: *raw u8, count: u64, size: u64, compare: extern fun(*raw u8, *raw u8): i32)
  fun bsearch(key: *raw u8, base: *raw u8, count: u64, size: u64, compare: extern fun(*raw u8, *raw u8): i32): (*raw u8)?
}

// C's comparator shape: negative, zero or positive, like `compareTo`.
extern "C" fun ascending(a: *raw u8, b: *raw u8): i32 {
  // SAFETY: qsort and bsearch call this only with pointers into the list of
  // i64 they were given
  val x = unsafe {
    *(a as *raw i64)
  }
  // SAFETY: as for `a`
  val y = unsafe {
    *(b as *raw i64)
  }
  if (x < y) -1 else if (x > y) 1 else 0
}

// A C library that walks something and calls back with `userdata` —
// written here in Veles so the example needs no C file.
fun forEachDigit(text: string, visit: extern fun(i64, *raw u8), userdata: *raw u8) {
  loop (b in text.bytes()) {
    // SAFETY: `visit` is a C function pointer taken from a Veles `extern fun`;
    // it gets back the `userdata` it was handed
    if (b >= '0' && b <= '9') unsafe {
      visit((b - '0') as i64, userdata)
    }
  }
}

struct Digits {
  seen: MutableList<i64>
}

extern "C" fun onDigit(d: i64, userdata: *raw u8) {
  // SAFETY: forEachDigit passes on the pointer `main` gave it, from a Handle<Digits>
  val digits = unsafe {
    ffi.Handle<Digits>.from(userdata)
  }
  digits.seen.push(d)
}

fun main() throws ffi.NulByte {
  // strings: a NUL-terminated copy C may read, freed by `with`
  with (hex = try ffi.CString.of("ff")) {
    // SAFETY: `hex` is open inside this `with`, so ptr() is a live, NUL-terminated
    // copy; strlen and strtoll only read it
    println("strlen: ${unsafe { strlen(hex.ptr()) }}, as hex: ${unsafe { strtoll(hex.ptr(), null, 16) }}")
    // SAFETY: as above: readString copies up to the NUL
    println("read back: ${unsafe { ffi.readString(hex.ptr()) }}")
  }
  when (ffi.CString.of("evil\u{0}.txt")) {
    is Ok     => println("a NUL byte was accepted")
    is Err(e) => println("refused: ${e.message()}")
  }

  // a list lent for the length of a call: no copy
  val buf: MutableList<u8> = [1, 2, 3, 4, 5, 6]
  // SAFETY: withRaw lends the list's storage for this call; memset writes 3 of
  // its 6 bytes
  buf.withRaw(p => unsafe {
    memset(p, 0, 3)
  })
  println("memset: $buf")

  // a Veles function as a C callback
  val xs: MutableList<i64> = [42, 7, 19, 3, 88, 21]
  // SAFETY: qsort sorts xs.len() elements of 8 bytes inside the lent storage
  xs.withRaw(p => unsafe {
    qsort(p as *raw u8, xs.len() as u64, 8, &ascending)
  })
  println("qsort: $xs")
  val key: i64 = 19
  // SAFETY: `key` outlives the call and bsearch reads xs.len() elements of 8
  // bytes; what it returns points into the same lent storage as `p`, so the
  // difference counts the elements between them (D50)
  val at = xs.withRaw(p => unsafe {
    val hit = bsearch(&key as *raw u8, p as *raw u8, xs.len() as u64, 8, &ascending)
    if (hit != null) (hit as *raw i64) - p else -1
  })
  println("bsearch 19: at index $at")

  // a Veles value through C's `void *userdata`
  val digits = Digits(seen: [])
  with (h = ffi.handle(digits)) {
    forEachDigit("a1b22c333", &onDigit, h.ptr())
  }
  println("digits: ${digits.seen}")

  // memory C keeps past a call: allocated and freed explicitly
  val block = ffi.alloc(4)
  // SAFETY: `block` is 4 bytes from ffi.alloc
  unsafe {
    memset(block, 7, 4)
  }
  // SAFETY: memset filled all 4 bytes of `block` above
  println("alloc: ${unsafe { ffi.readBytes(block, 4) }}")
  // SAFETY: `block` came from ffi.alloc and is freed once, after its last use
  unsafe {
    ffi.free(block)
  }
}
