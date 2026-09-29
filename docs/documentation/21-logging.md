# 21. Logging

`log` writes one line per event to standard error, with a level and named
fields. It is built so that leaving a debug line in costs nothing while
debugging is off, and so that the lines are useful both to a person at a
terminal and to a log collector reading a pipe (D91).

## Four levels and fields

```veles
use io { println }
use log { field }

fun main() {
  log.info("served", field("path", "/notes"), field("ms", 3))
  log.debug("this is not shown: the default level is info")
  log.warn("disk is filling", field("free", "12%"))
  log.error("could not save", field("path", "/notes/1"))
  println("done")
}
```

Output:
```text
done
```

The log lines go to standard error, so a program's own output is untouched.
The four functions are `debug`, `info`, `warn` and `error`, from least to
most serious; there is no `fatal`, because a panic already stops the task
with its location (chapter 7).

What a line looks like depends on where standard error goes. On a terminal
it is text for a person:

```text
2026-09-29T10:15:03.123Z INFO  served path=/notes ms=3
```

and anywhere else — a file, a pipe, a container's log driver — one JSON
object per line, for whatever reads it next:

```text
{"time":"2026-09-29T10:15:03.123Z","level":"info","msg":"served","path":"/notes","ms":3}
```

`field(key, value)` takes any value that can be encoded (chapter 18). A
number stays a number in the JSON line and a string stays a string; in a
text line a string is written bare unless it would run into the next field
(a space or an `=` in it), then it is quoted. A value that cannot be encoded
is logged as `null` rather than failing the call: logging never throws.

## The level

The threshold starts at `info`. Set it for a run with the environment,

```bash
VELES_LOG=debug ./server        # debug, info, warn, error, or off
```

or in code with `log.setLevel(log.Level.Warn)`. A message is logged when its
level is at least the threshold; `Level.Off` lets nothing through. An
unknown `VELES_LOG` is reported once and ignored. `log.enabled(level)` says
whether a message of that level would be logged now.

## A message that costs nothing when it is off

The message of `debug`, `info`, `warn` and `error` is a **lazy** parameter
(D90): the text you write is turned into a function and evaluated only if
the level is on.

```veles
use io { println }
use log

val built = Mutex(value: 0)

fun dump(): string {
  built.withLock(n => *n += 1)
  "a large summary"
}

fun main() {
  log.debug("state: ${dump()}")     // not evaluated: Debug is off
  println("built ${built.get()} time(s)")
  log.setLevel(log.Level.Debug)
  log.debug("state: ${dump()}")     // evaluated now
  println("built ${built.get()} time(s)")
}
```

Output:
```text
built 0 time(s)
built 1 time(s)
```

The arguments of `field(...)` are evaluated at the call whether or not the
line is logged; on a hot path that builds expensive fields, guard the call
with `if (log.enabled(log.Level.Debug)) { ... }`.

`lazy` is a modifier of a parameter of type `fun(): T` (`fun debug(lazy msg:
fun(): string)`): an argument that is not a lambda is wrapped in one, and the
function calls `msg()` when it wants the value. Only the standard library may
declare one for now.

## Fields for a whole piece of work

`log.withFields` adds fields to every line logged inside a function, by it,
by what it calls and by the tasks it starts, without passing anything down:

```veles
use log { field }

fun handle(id: string) {
  log.withFields([field("request", id)], () => {
    log.info("start")
    process()
    log.info("done")
  })
}

fun process() {
  log.debug("deep inside")     // carries request=... too
}
```

It is a task-local (chapter 12): tasks started inside see the fields bound
where they were started, and the binding ends when the function returns,
throws or is cancelled. A nested `withFields` adds to the outer one.

The HTTP server uses it. `http.requestId()` binds the request's id as the
field `id` for the whole request, so every line a handler logs — and
`http.logging()`'s access line — carries it. `serve` itself logs each
request, failure and shutdown through `log`, so the server's own lines
follow `VELES_LOG` and the JSON-or-text rule too.

## Lines from many threads

Each call writes its line whole, so lines from different threads or tasks
never interleave: a reader can split the stream at newlines and parse every
piece.

Next: back to the [index](index.md).
