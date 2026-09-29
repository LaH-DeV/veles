use io { println }

fun apply(f: fun(i64): i64, x: i64): i64 = f(x)

fun twice<T, U>(x: T, f: fun(T): U): U = f(x)

fun counter(): fun(): i64 {
  var n = 0
  () => {
    n += 1
    n
  }
}

fun double(x: i64): i64 = x * 2

struct Acc {
  var total: i64 = 0
  fun addAll(xs: List<i64>) {
    xs.forEach(x => this.total += x)
  }
}

fun main() {
  val nums = [3, 1, 2]
  val doubled = nums.map(x => x * 2)
  val total = nums.fold(0, (acc, x) => acc + x)
  val typed = nums.map((x: i64) => "<$x>")
  println("$doubled $total $typed")
  println("${nums.filter(x => x > 1)} ${nums.any(x => x > 2)} ${nums.all(x => x > 2)}")
  println("${nums.find(x => x == 2) ?: -1} ${nums.indexOf(1)} ${nums.contains(9)} ${nums.reversed()}")
  println("${nums.sorted()} ${nums.sortedBy(x => -x)} ${["bb", "a", "ccc"].sortedBy(s => s.len())}")
  println("${nums.first() ?: 0} ${nums.last() ?: 0} ${nums.join(", ")}")
  val next = counter()
  next()
  next()
  println("counter ${next()} apply ${apply(double, 21)} apply ${apply(x => x + 1, 1)}")
  println("twice ${twice(4, x => x * x)} ${twice("hi", s => s.len())}")
  var captured = "before"
  val show = () => captured
  captured = "after"
  println("capture ${show()}")
  var acc = Acc()
  acc.addAll([1, 2, 3])
  println("acc ${acc.total}")
  val pairs = [(1, "one"), (2, "two")]
  println("${pairs.map((n, s) => "$n=$s")}")
  val adder = (a: i64) => (b: i64) => a + b
  println("curried ${adder(2)(3)}")
}
