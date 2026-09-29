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

fun main() {
  val s = Stack<i64>()
  s.push(1)
  s.push(2)
  println("len ${s.len()} pop ${s.pop() ?: -1} pop ${s.pop() ?: -1} pop ${s.pop() ?: -1}")
  println(describe(Cat(name: "Tom")))
  println(describe(42))
  println("first ${first([7, 8], 0)} ${first([] as List<string>, "none")}")
}
