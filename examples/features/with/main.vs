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

// D100: the rest of the block is the body; its value is computed before the
// closes, the last opened closing first
fun lengths(): i64 {
  with a = open("first")
  with b = open("second")
  a.name.len() + b.name.len()
}

fun ticker(ticks: Channel<i64>) {
  var n = 0
  loop {
    await sleep(Duration.millis(5))
    n += 1
    ticks.send(n)
  }
}

// a task bound by `with` runs in the background until the block ends, and
// is then cancelled and joined
fun background() {
  val ticks = Channel<i64>(capacity: 1000)
  with r = open("before the ticker")
  with t = async ticker(ticks)
  await sleep(Duration.millis(40))
  println("ticked: ${ticks.len() > 0}")
}

fun main() {
  println("early ${early(true)}")
  println("early ${early(false)}")
  loop (i in 0..3) {
    with l = open("loop$i")
    if (i == 1) continue
    if (i == 2) break
    println("used ${l.name}")
  }
  when (failing()) {
    is Ok(v)  => println("ok $v")
    is Err(e) => println("failed $e")
  }
  println("lengths ${lengths()}")
  background()
  println("done")
}
