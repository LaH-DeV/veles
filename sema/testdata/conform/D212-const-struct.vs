// D212: `const struct` — a type immutable all the way down. No `var`
// field, and every field deeply immutable: numbers, strings, immutable
// collections of such, tuples, nullables, arrays, other const structs,
// sealed types whose variants are all const, errors of such. What can
// change is refused, naming the field and what to use instead. Its values
// are Sendable, map keys, module values and constants.
use io

const struct Tls {
  cert: string
}

const struct Config {
  host:  string
  ports: List<i64>
  tls:   Tls?
  span:  (i64, i64)
  wait:  Duration
  fun url(): string => "https://$host:${ports.first() ?: 443}"
}

const struct Pair<A, B> {
  first:  A
  second: B
}

sealed trait Shape {
  const struct Circle {
    r: f64
  }
  const compact struct Rect {
    flag: u8
    w:    f64
    h:    f64
  }
}

const struct Drawing {
  shapes: List<Shape>
  names:  Map<string, Pair<i64, string>>
  grid:   Array<i64, 4>
}

struct Plain {
  n: i64
}

sealed trait Mixed {
  const struct Fine {
    n: i64
  }
  struct Loose {
    n: i64
  }
}

trait Named {
  fun name(): string
}

error Broken {
  var code: i64
}

const struct Bad {
  var count: i64               // error: 'count' is a 'var' field; a const struct holds only immutable data
  items:     List<i64>         // a value (D231)
  rows:      List<Set<i64>>
  byKey:     Map<string, i64>
  plain:     Plain             // error: declare 'Plain' as 'const struct'
  mixed:     Mixed             // error: 'mixed' is a 'Mixed', which holds a 'Loose'
  ptr:       *i64              // error: a pointer reaches a value that can change
  f:         fun(i64): i64     // error: a function value can hold state
  named:     Named             // error: an object of a trait can hold state
  ch:        Channel<i64>      // error: a channel is shared state
  lock:      Mutex<i64>        // error: a synchronized type is shared state
  sb:        StringBuilder     // error: use 'string'
  broken:    Broken?           // error: 'Broken' has a 'var' field 'code'
}

val shared = Config(host: "localhost", ports: [8080], tls: null, span: (1, 2), wait: Duration.seconds(1))
const built = Pair(first: 1, second: "one")

fun pairUp<T>(x: T): Pair<i64, T> => Pair(first: 1, second: x)

fun main() {
  val ch = Channel<Config>(capacity: 1)
  ch.send(shared)
  val got = await ch.recv()
  val byConfig: Map<Config, i64> = [shared: 1]
  val d = Drawing(shapes: [Circle(r: 1.0), Rect(flag: 1, w: 2.0, h: 3.0)], names: ["a": built], grid: [1, 2, 3, 4])
  val wrong: Pair<i64, List<i64>> = Pair(first: 1, second: [])
  val viaCall = pairUp(Set<i64>())
  val locked = Pair<i64, Mutex<i64>>(first: 1, second: Mutex(2)) // error: 'Pair' is a const struct, so its field 'second' is a 'Mutex<i64>'
  io.println("${got?.url()} ${byConfig.get(shared)} ${d.shapes.len()} ${built.second} ${wrong.first} ${viaCall.first} ${locked.first}")
}
