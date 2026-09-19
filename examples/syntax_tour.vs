// A tour of the Veles surface syntax, taken from veles-spec.md.
use io
use math.geometry as geo, otherModule

fun main() throws {
  val sum = add(5, 7)
  printSum(sum)
}

fun add(a: i32, b: i32) = a + b

fun printSum(sum: i32) {
  io.println("The sum is $sum and ${sum * 2}")
  otherModule.someFunc()
}

// D12 — sealed traits
sealed trait Shape
struct Circle : Shape {
  radius: f64
}
struct Rect : Shape {
  w: f64
  h: f64
}

// D13 — when with patterns, guards, destructuring
fun area(shape: Shape): f64 = when (shape) {
  is Shape.Circle(radius) if radius > 10.0 => PI * radius * radius
  is Shape.Circle(radius)                  => PI * radius * radius
  is Shape.Rect(w, h)                      => w * h
}

fun classify(n: i32): string = when {
  n < 0  => "negative"
  n == 0 => "zero"
  else   => "positive"
}

fun describe(n: i32?): string = when (n) {
  null     => "nothing"
  1, 2     => "small"
  in 3..10 => "medium"
  is i32   => "big: $n"
}

// D22 — methods, mut fun, expression bodies
struct Counter {
  n: i32 = 0

  fun get(): i32 = self.n
  mut fun bump() {
    self.n += 1
  }

  // D23 (v0.23) — a trait impl inside the body of your own type; a static
  // function has no self and is called on the type: Counter.zero()
  impl Display {
    fun toString(): string = "Counter(${self.n})"
  }
  static fun zero(): Counter = Counter()
}

// D27/D42 — traits with associated types and default bodies
trait Iterable {
  type Iter: Iterator
  fun iterator(): Iter
}

trait Iterator {
  type Item
  mut fun next(): Item?
  fun count(): i32 {
    var n = 0
    loop (x in self) {
      n += 1
    }
    n
  }
}

// D40 — declared effects on trait methods and function types; a bare
// `throws` leaves the error type to each impl (v0.24)
trait Fetcher {
  fun fetch(url: string): Bytes suspends throws
}

val handler: fun(Request): Response suspends throws HttpError = handle

// D23 — impl blocks
impl<T> Display for Stack<T> {
  type Output = string
  fun show(): string = "stack"
  override fun hint(): i32 = 1
}

// D23 addendum — extend blocks: inherent methods for a type you declare
extend<T: Display> Stack<T> {
  public fun render(): string = self.items.map(x => x.show()).join(" ")
  mut fun drain() {
    self.items.clear()
  }
}

// D31 — recursion through pointers
struct Node<T> : Tree<T> {
  value: T
  left:  *Tree<T>
  right: *Tree<T>
}

// D28 — construction, named args, generics at call sites
fun construct() {
  val c = Config(port: 8080, host: "localhost")
  val s = Stack<i32>()
  val p: (*User)? = null
  var xs: MutableList<i32> = []
  val m = ["a": 1, "b": 2]
  val empty = [:]
  val pair: (i32, string) = (1, "x")
  val (a, b) = pair
  val q = pair.0
}

// D29/D30/D32 — ranges, elvis, safe call, lambdas
fun collections(nums: List<i32>, counts: Map<string, i32>, word: string, a: A?) {
  val doubled = nums.map(x => x * 2)
  val total = nums.fold(0, (acc, x) => acc + x)
  val typed = nums.map((x: i32) => x * 2)
  val n = counts.get(word) ?: 0
  val deep = a?.b?.c
  loop (i in 0..<nums.len()) {
    io.println("$i")
  }
  loop (i in 1..100) { }
  val sorted = nums
    .sortedBy(x => -x)
    .take(3)
  loop :outer {
    loop {
      break outer
    }
  }
  val big = if (n > 5) "big" else "small"
  val block = nums.map(x => {
    val y = x * 2
    y + 1
  })
}

// D34/D36/D38 — concurrency
fun concurrent() throws {
  scope {
    async worker(1, jobs, results)
    async worker(2, jobs, results)
  }
  val results = gather {
    async fetch(a)
    async fetch(b)
  }
  race {
    val job = jobs.recv() => process(job)
    val cmd = ctrl.recv() => handle(cmd)
    timer(1.second)       => giveUp()
  }
  val msg = await ch.recv()
  ch.send(msg)
}

// D43/D44/D50 — with, unsafe, raw pointers
fun resources(path: string) throws {
  with (f = File.open(path), g = File.open(path)) {
    process(f)
  }
  unsafe {
    val n = strlen(ptr)
    val next = ptr + 1
  }
}

extern "C" {
  fun strlen(s: *raw u8): usize
  fun puts(s: *raw u8): i32
}

// D45 — error unions, try
fun loadConfig(path: string): Config throws IoError | ParseError {
  val text = try fs.readText(path)
  val port = try parsePort(text)
  Config(port: port)
}

// D51 — attributes
@test
fun userQueryReturnsRows() { }

@deprecated("use parseConfig instead")
fun oldLoad(path: string): Config throws = loadConfig(path)

// D21 — wrapping arithmetic; D44 unsafe fun
unsafe fun wrap(a: i32, b: i32): i32 = a +% b

fun typeTests(x: Shape, y: Any) {
  if (x is Shape.Circle && !(y !is string)) { }
  val z = y as i64
  val addr = &x
}

public const PI: f64 = 3.14159265358979
public var counter = 0
