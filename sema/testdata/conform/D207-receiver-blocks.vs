// D207: `fun on T()` is a receiver function, whose `this` is a T; a block
// `{ … }` alone is one where one is expected (a receiver block), or a plain
// `fun()` where that is. A block on the same line after a call or a method
// name is its last argument. `apply` runs a block on a value in place,
// `edit` on a copy; `apply` on a `val` is refused.
use io

struct Server {
  var host: string = "localhost"
  var port: i64 = 80
}

struct Job {
  task: Task<()>
}

struct Log {
  implement Closeable { fun close() { } }
}

error Bad { }

fun risky(): i64 throws Bad => 1

fun later(f: fun()) {
  f()
}

fun configure(server: *Server, body: fun on Server()) {
  body(*server)
}

val defaults = Server()

fun main() {
  var server = Server()
  configure(&server) {
    port = 8080
  }
  configure(&server, { port = 8081 })
  val name = "  Big Box ".edit {
    trim()
    toLower()
    replace(" ", "-")
  }
  later { io.println("${server.port} $name") }
  val fixed = Server()
  fixed.apply { port = 1 } // error: 'apply' changes 'fixed' in place, and 'fixed' is a 'val'; 'edit' gives a changed copy: 'val changed = fixed.edit { … }' (D207)
  val loose = { // error: a block '{ … }' alone is a function where one is expected — a 'fun on T()' (a receiver block) or a 'fun()'; here nothing says which: give it a type, 'val f: fun() = { … }' (D207)
    io.println("y")
  }
  val each: fun(i64) = { io.println("z") } // error: this block takes no parameters, but a 'fun(i64)' is expected, which passes 1; write '(a) => { … }' (D207)
  val twice: fun on Server(i64) = { port = 2 } // error: this block takes no parameters, but a 'fun on Server(i64)' is expected, which passes 1; write '(a) => { … }' (D207)
  val two = server.edit({ }, { }) // error: 'edit' takes one block: 'x.edit { … }' (D207)
  val wrong = server.edit((s: Server) => s.port) // error: lambda takes 1 parameter but a function of 0 is expected here
  val looped = Server().edit {
    loop (i in 0..<3) {
      if (i == 1) continue
      port += i
    }
    if (port > 2) return // error: 'return' cannot leave a receiver block given to 'apply' or 'edit': the block runs to its end, and the call gives back the value; return after it (D207)
  }
  loop (_ in 0..<2) {
    server.apply {
      break // error: 'break' cannot leave a receiver block: the block runs to its end; a loop inside it may use 'break' (D207)
    }
  }
  configure(&server, (n: i64) => { }) // error: lambda takes 1 parameter but a function of 0 is expected here
  val plain: fun on Server() = { port = 4 }
  plain(defaults) // error: 'defaults' is a global 'val', a constant every task shares; a function on it may change it — copy it first: 'var copy = defaults' (D207)
  with log = Log()
  val copied = log.edit { } // error: 'log' is bound by 'with', which closes it when the block ends; a copy would be a second handle to the same resource (D100, D207)
  val c: (extern fun on Server())? = null // error: a C function pointer has no receiver; write 'extern fun(*T, ...)' (D69, D207)
  io.println("$two $wrong $looped $each $twice $copied $c $loose")
}

fun holding(job: Job) {
  val again = job.edit { } // error: 'Job' holds a running task, which a copy would duplicate; 'edit' is refused for task-holding types (D111, D207)
}

// a literal block is checked in place: `try` in it reaches the caller
fun tuned(): Server throws Bad => Server().edit { port = try risky() }

fun viaThrowing(f: fun on Server() throws Bad): Server {
  val s = Server()
  s.edit(f) // error: a receiver function given to 'edit' cannot throw; handle the error inside it, or write the block in place, where 'try' reaches the caller (D207)
}

fun viaValue(f: fun on Server()): string {
  val other = 5
  val text = other.edit(f) // error: 'edit' takes a block that runs on a 'i64' — '{ … }', a 'fun on i64()' — but this is 'fun on Server()' (D207)
  "$text"
}
