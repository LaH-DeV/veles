use io

struct Res {
  name: string
}
impl Closeable for Res {
  mut fun close() {
    io.println("close ${self.name}")
  }
}

fun open(name: string): Res {
  io.println("open $name")
  Res(name: name)
}

error Oops { }

fun early(flag: bool): i64 {
  with (a = open("a"), b = open("b")) {
    if (flag) return 1
    io.println("body ${a.name} ${b.name}")
  }
  2
}

fun failing(): i64 throws Oops {
  with (r = open("r")) {
    if (r.name == "r") throw Oops()
  }
  0
}

fun main() {
  io.println("early ${early(true)}")
  io.println("early ${early(false)}")
  loop (i in 0..3) {
    with (l = open("loop$i")) {
      if (i == 1) continue
      if (i == 2) break
      io.println("used ${l.name}")
    }
  }
  when (failing()) {
    is Ok(v)  => io.println("ok $v")
    is Err(e) => io.println("failed $e")
  }
}
