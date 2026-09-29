use io { println }

struct Res {
  name: string

  implement Closeable {
    fun close() {
      println("close ${this.name}")
    }
  }
}

fun open(name: string): Res {
  println("open $name")
  Res(name)
}

error Oops { }

fun early(flag: bool): i64 {
  with (a = open("a"), b = open("b")) {
    if (flag) return 1
    println("body ${a.name} ${b.name}")
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
  println("early ${early(true)}")
  println("early ${early(false)}")
  loop (i in 0..3) {
    with (l = open("loop$i")) {
      if (i == 1) continue
      if (i == 2) break
      println("used ${l.name}")
    }
  }
  when (failing()) {
    is Ok(v)  => println("ok $v")
    is Err(e) => println("failed $e")
  }
}
