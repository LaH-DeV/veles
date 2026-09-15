use io

fun apply(f: fun(i32): i32, x: i32): i32 = f(x)

fun <T, U> twice(x: T, f: fun(T): U): U = f(x)

fun counter(): fun(): i32 {
  var n = 0
  () => {
    n += 1
    n
  }
}

fun double(x: i32): i32 = x * 2

struct Acc {
  total: i32 = 0
  mut fun addAll(xs: List<i32>) {
    xs.forEach(x => self.total += x)
  }
}

fun main() {
  val nums = [3, 1, 2]
  val doubled = nums.map(x => x * 2)
  val total = nums.fold(0, (acc, x) => acc + x)
  val typed = nums.map((x: i32) => "<$x>")
  io.println("$doubled $total $typed")
  io.println("${nums.filter(x => x > 1)} ${nums.any(x => x > 2)} ${nums.all(x => x > 2)}")
  io.println("${nums.find(x => x == 2) ?: -1} ${nums.indexOf(1)} ${nums.contains(9)} ${nums.reversed()}")
  io.println("${nums.sorted()} ${nums.sortedBy(x => -x)} ${["bb", "a", "ccc"].sortedBy(s => s.len())}")
  io.println("${nums.first() ?: 0} ${nums.last() ?: 0} ${nums.joinToString(", ")}")
  val next = counter()
  next()
  next()
  io.println("counter ${next()} apply ${apply(double, 21)} apply ${apply(x => x + 1, 1)}")
  io.println("twice ${twice(4, x => x * x)} ${twice("hi", s => s.len())}")
  var captured = "before"
  val show = () => captured
  captured = "after"
  io.println("capture ${show()}")
  var acc = Acc()
  acc.addAll([1, 2, 3])
  io.println("acc ${acc.total}")
  val pairs = [(1, "one"), (2, "two")]
  io.println("${pairs.map((n, s) => "$n=$s")}")
  val adder = (a: i32) => (b: i32) => a + b
  io.println("curried ${adder(2)(3)}")
}
