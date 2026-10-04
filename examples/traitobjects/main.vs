use io { println }

trait Shape {
  fun area(): f64
  fun name(): string
  fun describe(): string = "${this.name()} with area ${this.area()}"
}

struct Circle {
  r: f64

  implement Shape {
    fun area(): f64 = 3.0 * this.r * this.r
    fun name(): string = "circle"
  }
}
struct Square {
  side: f64

  implement Shape {
    fun area(): f64 = this.side * this.side
    fun name(): string = "square"
    override fun describe(): string = "a square of side ${this.side}"
  }
}

trait Counter {
  fun bump(): i64
}

struct Clicks {
  var n: i64 = 0

  implement Counter {
    fun bump(): i64 {
      this.n += 1
      this.n
    }
  }
}

// A sink takes lines; some sinks also buffer them and can be flushed.
// `is Flusher` asks the value's own type at run time (D117), so `finish`
// works for every sink and flushes the ones that buffer.
trait Sink {
  fun write(line: string)
}

trait Flusher {
  fun flush(): i64
}

struct Console {
  implement Sink {
    fun write(line: string) {
      println("console: $line")
    }
  }
}

struct Buffered {
  pending: MutableList<string> = []

  implement Sink {
    fun write(line: string) {
      this.pending.push(line)
    }
  }
  implement Flusher {
    fun flush(): i64 {
      loop (line in this.pending) {
        println("buffered: $line")
      }
      val n = this.pending.len()
      this.pending.clear()
      n
    }
  }
}

fun finish(sink: Sink) {
  if (sink is Flusher) println("flushed ${sink.flush()} lines")
}

// `is Circle` asks whether a shape is that type, and reads it back (D135).
fun widest(shapes: List<Shape>): string {
  var best = 0.0
  loop (s in shapes) {
    when (s) {
      is Circle(r) => best = best.max(2.0 * r)
      is Square    => best = best.max(s.side)
      else         => { }
    }
  }
  "widest $best"
}

fun total(shapes: List<Shape>): f64 {
  var sum = 0.0
  loop (s in shapes) {
    sum += s.area()
  }
  sum
}

error Refused { }

// a trait method that may fail: an implementor that never does still fills
// the slot (D40), and a suspending method is callable through the object
trait Source {
  fun next(): i64 suspends throws Refused
}

struct Steady {
  implement Source {
    fun next(): i64 = 7
  }
}

struct Picky {
  limit: i64

  implement Source {
    fun next(): i64 {
      await sleep(Duration.millis(1))
      if (this.limit < 3) throw Refused()
      this.limit
    }
  }
}

// a function held in a field, called and tried like any other
struct Handler {
  run: fun(i64): i64 throws Refused
}

fun pull(source: Source): string {
  when (source.next()) {
    is Ok(n)  => "got $n"
    is Err(_) => "refused"
  }
}

fun main() {
  println(pull(Steady()))
  println(pull(Picky(limit: 2)))
  println(pull(Picky(limit: 5)))
  val handler = Handler(run: n => if (n > 1) n else throw Refused())
  when (handler.run(4)) {
    is Ok(n)  => println("handled $n")
    is Err(_) => println("refused")
  }
  val shapes: List<Shape> = [Circle(r: 1.0), Square(side: 2.0)]
  loop (s in shapes) {
    println(s.describe())
  }
  println("total ${total(shapes)}")
  val one: Shape = Circle(r: 2.0)
  println("${one.name()} ${one.area()} $one")
  var c: Counter = Clicks()
  c.bump()
  println("bumped ${c.bump()}")
  val holder = (c, "pair")
  println("bumped again ${holder.0.bump()}")
  // narrowed, `c` is the Clicks inside the object: a write changes it
  if (c is Clicks) c.n = 10
  println("reset, then bumped ${c.bump()}")
  println(widest(shapes))
  val sinks: List<Sink> = [Console(), Buffered()]
  loop (sink in sinks) {
    sink.write("hello")
    sink.write("world")
    finish(sink)
  }
}
