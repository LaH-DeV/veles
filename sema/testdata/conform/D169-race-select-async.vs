// D169: `race { }` is a block like gather whose tasks race — the first to
// finish wins, the others are cancelled and joined; `select { }` is the
// construct with arms, each `src => body`, a value named `val x = src =>`;
// `async { }` starts a task from a block. The D168 spellings (a race with
// arms, `src as x`, bare arms, `awaitFirst`) are errors with a fix.
use io

error Down {
  host: string
}

fun mirror(name: string, delay: Duration): string {
  await sleep(delay)
  name
}

fun flaky(delay: Duration): string throws Down {
  await sleep(delay)
  throw Down(host: "eu")
}

val atStart = race { // error: 'race' cannot appear in a global initializer
  async mirror("eu", Duration.millis(1))
  async mirror("us", Duration.millis(1))
}

fun fastest(): string => race {
  async mirror("eu", Duration.millis(30))
  async mirror("us", Duration.millis(10))
}

fun hedged(): string => race {
  async mirror("primary", Duration.millis(50))
  async {
    await sleep(Duration.millis(10))
    mirror("backup", Duration.millis(5))
  }
}

fun firstOf(names: List<string>): string => race {
  loop (n in names) async mirror(n, Duration.millis(5))
}

fun failsFirst(): string throws Down => race {
  async flaky(Duration.millis(1))
  async mirror("late", Duration.millis(50))
}

fun noTask(): string => race { // error: this 'race' starts no task
  io.println("nothing raced")
}

fun oneTask(): string => race { // warning: a 'race' of one task is the call itself
  async mirror("only", Duration.millis(1))
}

fun twoTypes(): string => race {
  async mirror("a", Duration.millis(1))
  async { 42 } // error: the tasks of a 'race' give one type: this one gives 'i64', the first gives 'string' (D169)
}

fun selected(inbox: Channel<string>): string => select {
  val msg = inbox.recv()                                => msg ?: "closed"
  val line = async mirror("computed", Duration.millis(1)) => line
  sleep(Duration.seconds(1))                            => "timeout"
}

fun blockValue(): i64 => scope {
  val total = async {
    var sum = 0
    loop (i in 1..10) sum += i
    if (sum > 1000) return -1
    sum
  }
  io.println("meanwhile")
  await total
}

fun capturesVar(): i64 => scope {
  var n = 0
  val t = async { n += 1 } // error: cannot cross a task boundary: it captures 'n', a 'var'
  io.println("meanwhile")
  await t
  n
}

fun oldArms(inbox: Channel<string>) {
  race { // error: a 'race' with arms is a 'select' now; 'race { }' runs tasks and the first to finish wins (D169)
    val msg = inbox.recv() => io.println(msg ?: "")
    sleep(Duration.millis(1)) => { }
  }
}

fun oldAs(inbox: Channel<string>) {
  select {
    inbox.recv() as msg => io.println(msg ?: "") // error: a select arm names its value 'val msg = inbox.recv() => …' (D169)
    sleep(Duration.millis(1)) => { }
  }
}

fun oldBare(inbox: Channel<string>): string? => select {
  inbox.recv() // error: a select arm says what happens when it wins: 'val v = inbox.recv() => v'
  sleep(Duration.millis(1)) => null
}

fun awaited(): string => scope {
  val t = async mirror("t", Duration.millis(1))
  select {
    val v = await t => v // error: a select arm does not take 'await': the select is where the task waits; write 't' (D168)
    sleep(Duration.millis(5)) => "late"
  }
}

fun noValue() {
  select {
    val late = sleep(Duration.millis(2)) => { } // error: a 'sleep' arm has no value to name; remove 'val late =' (D169)
  }
}

fun unusedName(inbox: Channel<string>) {
  select {
    val _ = inbox.recv() => { } // error: write 'inbox.recv() => …': an arm need not name a value it does not use (D168)
    sleep(Duration.millis(1)) => { }
  }
}

fun awaitedAtOnce(): i64 => scope {
  val t = async { 41 + 1 }
  await t // warning: this 'async { }' runs with nothing alongside it; write its statements here directly
}

fun oldFirst(): string => awaitFirst(async mirror("a", Duration.millis(1)), async mirror("b", Duration.millis(1))) // error: 'awaitFirst' is now a 'race'

fun main() throws Down {
  io.println(fastest())
  io.println(hedged())
  io.println(firstOf(["a", "b"]))
  io.println(failsFirst() ?? "down")
  io.println(selected(Channel<string>(capacity: 1)))
  io.println("${blockValue()}")
}
