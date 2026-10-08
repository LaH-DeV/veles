# 22. Configuration

A program's settings — the port, the database, the log level — come from the
outside: the environment, a `.env` file next to the program, a JSON file the
deployment provides. `config` reads them into a struct you declare, once,
with types, defaults and every problem reported together (D125).

## One struct, one call

```veles
use config
use fs
use io { println }
use os
use path

struct Settings {
  port:        i64 = 8080
  host:        string = "127.0.0.1"
  debug:       bool = false
  timeout:     Duration = Duration.seconds(30)
  databaseUrl: Secret<string>
  implement Decodable
}

fun main() throws IoError {
  val file = path.join(os.tempDir(), "veles-doc-config.env")
  try fs.writeFile(file, "DOCS_PORT=9090\nDOCS_DATABASE_URL=postgres://app:pw@db/app\nDOCS_TIMEOUT=2m\n")
  when (config.load<Settings>(files: [file], prefix: "DOCS_")) {
    is Ok(s)  => {
      println("${s.host}:${s.port} debug=${s.debug} timeout=${s.timeout}")
      println("database: ${s.databaseUrl}")
    }
    is Err(e) => println(e.message())
  }
  try fs.remove(file)
}
```

Output:
```text
127.0.0.1:9090 debug=false timeout=2m
database: [redacted]
```

A real program writes `config.load<Settings>(files: [".env"])`; the `DOCS_`
prefix here keeps this page from reading your own environment. `implement
Decodable` is the same line that makes a type readable from JSON (chapter
18): the compiler writes the reading code from the fields, and `config` asks
it which variables to look for.

## Names

A field is its name in capitals with underscores: `port` is `PORT`,
`databaseUrl` is `DATABASE_URL`. A field whose type is a struct reads the
variables that begin with its own name, so `db: DbSettings` with a `poolSize`
field is `DB_POOL_SIZE`. `@key("NAME")` says the name outright, and
`@key(env: "NAME")` says it for the environment only — the same field can
have another name in JSON. A `prefix: "APP_"` goes in front of every name.

```veles
// fragment
struct Pool {
  size:    i64 = 10               // POOL_SIZE
  timeout: Duration = Duration.seconds(5)    // POOL_TIMEOUT
  implement Decodable
}

struct Settings {
  @key("LOG_LEVEL")
  level:   Level = Level.Info      // LOG_LEVEL, as written
  pool:    Pool = Pool()           // POOL_SIZE, POOL_TIMEOUT
  @key(env: "TOKEN_FILE")
  tokenPath: string?               // TOKEN_FILE; "tokenPath" in a JSON file
  implement Decodable
}
```

## What a variable can hold

Variables are text; the type of the field says how to read it.

| Field type | Written as |
|---|---|
| `i64`, `u8`, … | `8080` — a number that does not fit the type is a problem |
| `f64` | `0.25` |
| `bool` | `true` or `false`, nothing else (`yes`, `1` and `TRUE` are problems) |
| `string` | the text as it is |
| `Secret<T>` | like `T`, and never shown in a message |
| `Duration` | `30s`, `5m`, `1h30m`, `250ms` |
| `Timestamp` | RFC 3339: `2026-10-04T12:00:00Z` |
| an enum | a member's name, exactly: `Warn` |
| `List<T>` | comma separated, spaces around each item ignored: `a, b ,c` |
| `T?` | `null` when the variable is not set |

A field with a default uses it when nothing sets the variable. A field with
neither a default nor a `?` is *required*: not setting it is a problem. A
`Map` has no variable form, so `config.load` refuses a struct that has one with
a panic that names the field, the first time it runs; mark the field `@skip`
(with a default) or read it yourself.

An **empty** variable is a value, not an absent one: `PORT=` is the text
`""` — fine for a `string`, a problem for a number. Unsetting a variable is
how to fall back to the file or the default.

## Where values come from

From strongest to weakest: the environment, then each file in `files` from
the last to the first, then the field's default. So a deployment can put a
JSON file after a machine's `.env`, and a person debugging can still override
one value with `PORT=9000 veles run`.

- **dotenv** (any file that is not `*.json`): `NAME=value` lines; blank lines
  and `#` lines are skipped; `export NAME=value` works; a value may be
  `'single quoted'` (literal) or `"double quoted"` (with `\n`, `\t`, `\"` and
  `\\`, and it may span lines); an unquoted value ends at the line or at ` #`.
  There is **no interpolation**: `${HOME}` is those characters, and a `$` in
  a password is a `$`.
- **JSON** (`*.json`): an object whose keys are the field names (`poolSize`,
  not `POOL_SIZE`), with a nested object for a struct field, and real
  numbers, booleans and lists.
- A file that is not there is skipped — a `.env` is optional by convention. A
  file that is there and cannot be read or parsed is a problem.

## Every problem at once

A first deployment usually has several things wrong. `config.load` finds
all of them, names each by the variable a person sets, says where the value came
from, and puts the lot in one `config.Error`.

```veles
use config
use fs
use io { println }
use os
use path

struct Settings {
  port:        i64 = 8080
  debug:       bool = false
  databaseUrl: Secret<string>
  implement Decodable
}

fun main() throws IoError {
  val file = path.join(os.tempDir(), "veles-doc-problems.env")
  try fs.writeFile(file, "DOCS_PORT=eighty\nDOCS_DEBUG=yes\n")
  when (config.load<Settings>(files: [file], prefix: "DOCS_")) {
    is Ok(s)  => println("port ${s.port}")
    is Err(e) => {
      loop (p in e.problems) {
        println(p.toString().replace(file, "settings.env"))
      }
    }
  }
  try fs.remove(file)
}
```

Output:
```text
DOCS_PORT: expected an integer, found "eighty" (from settings.env)
DOCS_DEBUG: expected true or false, found "yes" (from settings.env)
DOCS_DATABASE_URL: not set (expected text)
```

`config.Error` has `message()`, so `fun main() throws config.Error` prints
those lines after `error: main failed with` and exits with status 1 when the
settings are wrong.
The value of a `Secret` field is never in a message: a problem with a secret
says what was expected, not what was found.

## Telling people what a program reads

`config.describe<Settings>()` lists the variables with what each expects, its
default, and whether it is required or secret — for a `--help`, a README, or
a startup log line (`prefix:` as in `load`).

```veles
use config
use io { println }

enum Level {
  Debug
  Info
  Warn
}

struct Settings {
  port:        i64 = 8080
  @key("LOG_LEVEL")
  level:       Level = Level.Info
  origins:     List<string> = []
  databaseUrl: Secret<string>
  implement Decodable
}

fun main() {
  loop (v in config.describe<Settings>()) {
    val mark = if (v.required) " (required)" else ""
    println("${v.name}: ${v.what}$mark")
  }
}
```

Output:
```text
PORT: an integer
LOG_LEVEL: one of "Debug", "Info", "Warn"
ORIGINS: a list of text, separated by commas
DATABASE_URL: text (required)
```

`examples/config` loads a struct from two files, prints this table, and shows
the problems of a broken one; `veles new --template server` reads its
`HOST` and `PORT` this way.

Next: back to the [index](index.md).
