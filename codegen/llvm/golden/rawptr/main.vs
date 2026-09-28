use io

extern "C" {
  fun malloc(n: u64): *raw u8
  fun free(p: *raw u8)
}

extern struct Pair {
  a: i32
  b: i64
}

// Raw pointer arithmetic (D50): element-scaled steps, a count between two
// pointers, and address order.
fun main() {
  // SAFETY: every access stays inside the four i64 allocated here, freed once
  unsafe {
    val base = malloc(32) as *raw i64
    var p = base
    loop (i in 0..<4) {
      *p = i * 10
      p += 1
    }
    val last = p - 1
    io.println("${*last} ${p - base} ${base < p} ${last >= p}")
    val pairs = base as *raw Pair
    io.println("${((pairs + 1) as *raw u8) - (base as *raw u8)}")
    free(base as *raw u8)
  }
}
