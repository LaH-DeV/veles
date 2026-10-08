// D141: a task handle stays in the scope or gather that started it, and in
// a fail-fast block awaiting a throwing child gives its value — the error is
// the block's. In `gather` the handle keeps the Result.
use io

error Boom { n: i64 }

fun work(n: i64): i64 throws Boom {
  if (n < 0) throw Boom(n)
  n * 10
}

fun outcome(n: i64): Result<i64, Boom> {
  if (n < 0) return Err(Boom(n))
  Ok(n)
}

fun pool(): i64 throws Boom {
  var total: i64 = 0
  scope {
    val tasks: MutableList<Task<i64>> = []
    loop (i in 0..<4) tasks.push(async work(i))
    loop (t in tasks) total += await t
  }
  total
}

fun annotated(): i64 throws Boom {
  scope {
    val tasks: MutableList<Task<Result<i64, Boom>>> = []
    tasks.push(async work(1)) // error: a task of this 'scope' is 'Task<i64>', not 'Task<Result<i64, Boom>>'
    return 0
  }
}

fun stray(): i64 throws Boom {
  scope {
    val t = async work(1)
    return try await t // error: 'await' gives the task's value, a 'i64'
  }
}

fun kept(): i64 throws Boom {
  val tasks: MutableList<Task<i64>> = []
  var last: Task<i64>? = null
  scope {
    tasks.push(async work(1)) // error: this task cannot be stored in 'tasks', which outlives the block
    val t = async work(2)
    last = t // error: 't' cannot be stored in 'last', which outlives the block
  }
  if (last == null) 0 else 1
}

fun returned(): Task<i64> {
  scope {
    return async work(1) // error: this task cannot be returned: a task belongs to the 'scope'
  }
}

fun captured(): i64 throws Boom {
  var later: fun(): i64 = () => 0
  scope {
    val t = async work(1)
    later = () => await t // error: this lambda captures a task, so it cannot be stored in 'later'
  }
  later()
}

fun inGather(): i64 {
  val (a, b) = gather {
    async work(1)
    val t = async outcome(-1)
    when (val r = await t) {
      is Ok => io.println("ok $r")
      is Err => io.println("handled ${r.message()}")
    }
  }
  if (a is Ok && b.ok) a else 0
}

fun main() throws Boom {
  io.println("${try pool()} ${inGather()}")
}
