use io

struct Stack<T> {
  items: MutableList<T> = []

  fun push(x: T) {
    self.items.push(x)
  }
  fun pop(): T? = self.items.pop()
  fun len(): i64 = self.items.len()
}

trait Show {
  fun show(): string
  fun shout(): string = self.show() + "!"
}

struct Cat {
  name: string

  impl Show {
    fun show(): string = "cat ${self.name}"
  }
}
impl Show for i64 {
  fun show(): string = "int $self"
}

fun <T: Show> describe(x: T): string = x.shout()

fun <T> first(xs: List<T>, fallback: T): T {
  if (xs.len() == 0) return fallback
  xs.atOrPanic(0)
}

fun main() {
  val s = Stack<i64>()
  s.push(1)
  s.push(2)
  io.println("len ${s.len()} pop ${s.pop() ?: -1} pop ${s.pop() ?: -1} pop ${s.pop() ?: -1}")
  io.println(describe(Cat(name: "Tom")))
  io.println(describe(42))
  io.println("first ${first([7, 8], 0)} ${first([] as List<string>, "none")}")
}
