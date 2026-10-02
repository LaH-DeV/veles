use io { println }

struct Stack<T> {
  items: MutableList<T> = []

  fun push(x: T) {
    this.items.push(x)
  }
  fun pop(): T? = this.items.pop()
  fun len(): i64 = this.items.len()
}

trait Show {
  fun show(): string
  fun shout(): string = this.show() + "!"
}

struct Cat {
  name: string

  implement Show {
    fun show(): string = "cat ${this.name}"
  }
}
implement Show for i64 {
  fun show(): string = "int $this"
}

fun describe<T: Show>(x: T): string = x.shout()

fun first<T>(xs: List<T>, fallback: T): T {
  if (xs.len() == 0) return fallback
  xs.at(0)  // a T: the check above proves the list is not empty
}

// `Default` (D119): a value generic code can start from. A tally per key
// starts every key at its type's default, whatever that type is.
struct Score {
  points: i64
  bonus:  bool
  tags:   List<string>

  implement Default
}

fun tally<V: Default>(keys: List<string>, add: fun(V): V): Map<string, V> {
  val out: MutableMap<string, V> = [:]
  loop (k in keys) {
    out.set(k, add(out.get(k) ?: V.default()))
  }
  out.toMap()
}

// `T implements Trait` (D117) is answered per instance at compile time: an
// instance compiles only the branch it takes, so the `Show` branch may call
// `shout()` although `T` is not bounded by `Show`.
fun label<T>(x: T): string {
  if (T implements Show && T implements Comparable) return "${x.shout()} (ordered)"
  if (T implements Show) return x.shout()
  "something unshowable"
}

fun main() {
  val s = Stack<i64>()
  s.push(1)
  s.push(2)
  println("len ${s.len()} pop ${s.pop() ?: -1} pop ${s.pop() ?: -1} pop ${s.pop() ?: -1}")
  println(describe(Cat(name: "Tom")))
  println(describe(42))
  println("first ${first([7, 8], 0)} ${first([], "none")}")
  val words = ["a", "b", "a"]
  println("counts ${tally(words, (n: i64) => n + 1)}")
  println("scores ${tally(words, (sc: Score) => Score(points: sc.points + 10, bonus: sc.points > 0, tags: sc.tags))}")
  println("empty ${Score.default()}")
  println("${label(5)}; ${label(Cat(name: "Tom"))}; ${label(Score.default())}")
}
