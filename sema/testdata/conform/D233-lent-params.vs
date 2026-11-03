// D233: a pointer handed to a function value is lent for the call. The body
// may read and change through it and pass it on to a call that keeps no
// pointer, but not return, store or capture it.
use io

struct Saved {
  var items: *List<string>
}

fun keep(xs: *List<string>): *List<string> => xs

fun stash(xs: *List<string>) {
  val s = Saved(items: xs)
  io.println("${s.items.len()}")
}

fun count(xs: *List<string>): i64 => xs.len()

val wrapped = Mutex<List<Saved>>(value: [])

fun wrapOut(xs: *List<string>) {
  wrapped.withLock(all => all.push(Saved(items: xs)))
}

fun each(f: fun(*List<string>)) {
  var xs: List<string> = ["a"]
  f(&xs)
}

fun forward(xs: *List<string>, f: fun(*List<string>)) {
  f(xs)
}

fun twice(f: fun()) {
  f()
  f()
}

struct Handlers {
  var all: List<fun()> = []
}

fun register(into: *Handlers, f: fun()) {
  into.all.push(f)
}

fun main() {
  val notes = Mutex(value: ["a"])
  val leaked = notes.withLock(xs => xs)                   // error: 'xs' is lent to this function only for the call (D233): the body returns it
  val empty: List<string> = []
  var saved = Saved(items: &empty)
  notes.withLock(xs => {                                  // error: 'xs' is lent to this function only for the call (D233): the body stores it
    saved.items = xs
  })
  val later = notes.withLock(xs => () => xs.len())        // error: 'xs' is lent to this function only for the call (D233): the body captures it
  val passed = notes.withLock(xs => keep(xs))             // error: 'xs' is lent to this function only for the call (D233): the body passes it where it is kept
  each(stash)
  each(wrapOut)                                           // error: 'wrapOut' keeps its pointer parameter 'xs', so it cannot be used as a function value
  notes.withLock(xs => {                                  // error: 'xs' is lent to this function only for the call (D233)
    val s = Saved(items: xs)
    saved = s
  })
  val copy = notes.withLock(xs => *xs)
  val size = notes.withLock(xs => count(xs))
  each(xs => xs.push("b"))
  each(xs => forward(xs, ys => ys.push("c")))
  each(count2)
  notes.withLock(xs => twice(() => xs.push("d")))
  notes.withLock(xs => twice(() => {                      // error: 'xs' is lent to this function only for the call (D233): the body captures it
    saved.items = xs
  }))
  var handlers = Handlers()
  notes.withLock(xs => register(&handlers, () => xs.push("e")))   // error: 'xs' is lent to this function only for the call (D233): the body captures it
  io.println("${leaked.len()} ${later()} ${passed.len()} ${copy} ${size}")
}

fun count2(xs: *List<string>) {
  io.println("${xs.len()}")
}
