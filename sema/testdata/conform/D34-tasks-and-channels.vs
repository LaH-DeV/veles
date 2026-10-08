// D16/D34/D35/D60: channels, sleeping, async and the structured blocks.
use io

struct Handle {
  items: MutableList<i64>
}

fun work(): i64 => 1

val sleeper = await sleep(Duration.millis(1)) // error: 'sleep' suspends; a global initializer cannot suspend // error: a global cannot have type '()'

fun untypedChannel() {
  val _ = Channel() // error: cannot infer the element type of the channel; write 'Channel<T>(...)'
}
fun unsendableChannel() {
  val _ = Channel<Handle>() // error: 'Handle' is not Sendable and cannot cross a task boundary through a channel
}
fun twoArguments() {
  val _ = Channel<i64>(1, 2) // error: Channel takes at most one argument, 'capacity'
}
fun wrongName() {
  val _ = Channel<i64>(size: 1) // error: Channel has no parameter 'size'; use 'capacity'
}
fun noSuchChannelMethod(ch: Channel<i64>) {
  ch.peek() // error: no method 'peek' on type 'Channel<i64>'
}
fun sleepTwo() {
  await sleep(Duration.millis(1), Duration.millis(2)) // error: 'sleep' takes one argument: a 'Duration'
}
fun sleepMillis() {
  await sleep(100) // error: 'sleep' takes a 'Duration', not a number of milliseconds
}
fun sleepText() {
  await sleep("1s") // error: 'sleep' takes a 'Duration', found 'string'
}
fun sleepNotAwaited() {
  sleep(Duration.millis(1)) // error: 'sleep()' always suspends and must be awaited
}
fun asyncValue(f: fun(): i64) {
  scope {
    async f() // error: 'f' is not a sendable function
  }
}
fun raceOnNumber() {
  val _ = race {
    work() => 1 // error: a race arm waits on 'ch.recv()', 'ch.send(v)', 'sleep(d)' or 'await task', not 'i64'
  }
}

fun main() {
  io.println("tasks")
}
