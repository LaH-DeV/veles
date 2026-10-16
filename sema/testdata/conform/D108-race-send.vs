// D108: a race arm may be `ch.send(v)`; when it wins the value is in the
// channel, when another arm wins it was not sent. The arm binds nothing.
use io

fun offer(queue: Channel<string>, line: string): bool {
  select {
    queue.send(line)           => true
    sleep(Duration.millis(50)) => false
  }
}

fun bound(queue: Channel<string>) {
  select {
    val r = queue.send("x")   => io.println("$r") // error: a send arm (when it wins, the value is in the channel, D108) has no value to name; remove 'val r ='
    sleep(Duration.millis(1)) => { }
  }
}

fun mixed(out: Channel<i64>, inbox: Channel<i64>): i64 {
  select {
    out.send(1)          => 0
    val v = inbox.recv() => v ?: -1
  }
}

fun main() {
  val q = Channel<string>(capacity: 1)
  io.println("${offer(q, "a")}")
  bound(q)
  io.println("${mixed(Channel<i64>(capacity: 1), Channel<i64>())}")
}
