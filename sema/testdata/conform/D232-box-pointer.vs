// D232: `&x` boxes as a trait object over x itself; boxing a copy of a
// named place whose implementation changes `this` is refused.
use io

trait Sink {
  fun put(b: u8)
}

trait Job : Sendable {
  fun run(): i64
}

trait Shape {
  fun area(): i64
}

struct Collect {
  var bytes: List<u8> = []

  implement Sink {
    fun put(b: u8) {
      this.bytes.push(b)
    }
  }

  implement Job {
    fun run(): i64 => this.bytes.len()
  }
}

struct Square {
  side: i64

  implement Shape {
    fun area(): i64 => this.side * this.side
  }
}

struct Writer {
  var out: Collect = Collect()

  fun flush(data: List<u8>) {
    writeAll(this.out, data)                   // error: the 'Sink' object gets a copy of 'this.out', and 'put' changes it
    writeAll(&this.out, data)
  }
}

fun writeAll(to: Sink, data: List<u8>) {
  loop (b in data) to.put(b)
}

fun total(shapes: List<Shape>): i64 => shapes.map(s => s.area()).sum()

fun main() {
  var out = Collect()
  writeAll(out, [1, 2])                        // error: the 'Sink' object gets a copy of 'out', and 'put' changes it, so 'out' would never see the change; pass '&out'
  writeAll(&out, [1, 2])
  writeAll(out.copy(), [3])
  writeAll(Collect(), [4])
  val job: Job = out.copy()
  val square = Square(side: 2)
  var w = Writer()
  w.flush([5])
  io.println("${out.bytes.len()} ${total([square, Square(side: 3)])} ${job.run()} ${w.out.bytes.len()}")
}
