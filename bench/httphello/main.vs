// I/O: an HTTP "hello" server on the loopback interface and 64 clients
// each making 250 keep-alive requests against it, all in one process —
// what the executor's socket wait (poll today, plan E3) costs per request.
// The checksum is the number of body bytes read.
use http { Client, Response, handler, testServer }
use io { println }
use time

fun client(url: string): i64 suspends throws http.FetchError {
  with c = Client(maxIdlePerHost: 1)
  var bytes: i64 = 0
  loop (_ in 0..<250) {
    bytes += (try c.get(url).text()).len()
  }
  bytes
}

fun main() throws IoError | http.FetchError {
  with srv = try testServer(handler(_ => Response.text("hello, world")))
  val url = srv.url + "/"
  val sw = time.Stopwatch.start()
  val tasks: MutableList<Task<Result<i64, http.FetchError>>> = []
  scope {
    loop (_ in 0..<64) tasks.push(async client(url))
  }
  var total: i64 = 0
  loop (t in tasks) total += try await t
  println("BENCH httphello 16000 ${sw.elapsed().toNanos()} $total")
}
