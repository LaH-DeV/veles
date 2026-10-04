// Tests of std/config (D125): each type a variable can become, the names
// the fields get, the order the sources are read in, files of both kinds, and
// that every problem is reported at once without a secret in any message.

use fs, os, path, time

enum Level {
  Debug
  Info
  Warn
}

struct Pool {
  size:    i64 = 5
  timeout: Duration = Duration.seconds(30)
  implement Decodable
}

struct Database {
  url:  Secret<string>
  pool: Pool = Pool()
  implement Decodable
}

struct Server {
  port:  i64 = 8080
  host:  string = "0.0.0.0"
  debug: bool = false
  ratio: f64 = 0.5
  @key("LOG_LEVEL")
  level: Level = Level.Info
  timeout:  Duration = Duration.seconds(30)
  hosts:    List<string> = ["a", "b"]
  limits:   List<i64> = []
  note:     string?
  since:    time.Timestamp?
  name:     string
  database: Database
  implement Decodable
}

struct Small {
  workers: u8 = 4
  retries: i32 = 3
  implement Decodable
}

struct Tagged {
  @key(env: "TOKEN_FILE")
  tokenPath: string = "none"
  @key("DATABASE")
  db: Pool = Pool()
  maybe: Pool?
  implement Decodable
}

struct WithMap {
  labels: Map<string, string> = [:]
  implement Decodable
}

test fun env(pairs: Map<string, string>): fun(string): string? = name => pairs.get(name)

// the two variables nothing can default, and `extra` on top
test fun minimal(extra: Map<string, string> = [:]): Map<string, string> {
  val vars: MutableMap<string, string> = ["NAME": "svc", "DATABASE_URL": "postgres://u:p@h/db"]
  loop ((name, value) in extra.entries()) {
    vars.set(name, value)
  }
  vars.toMap()
}

test fun read<T: Decodable>(vars: Map<string, string>, files: List<string> = [], prefix: string = ""): T throws Error =
  try loadWith<T>(env(vars), files, prefix)

test fun failures<T: Decodable>(vars: Map<string, string>, files: List<string> = [], prefix: string = ""): List<string> {
  when (loadWith<T>(env(vars), files, prefix)) {
    is Ok(_)  => ["loaded without a problem"]
    is Err(e) => e.problems.map(p => p.toString())
  }
}

test "defaults fill what is not set" {
  val s = try read<Server>(minimal())
  expect(s.port == 8080)
  expect(s.host == "0.0.0.0")
  expect(s.debug == false)
  expect(s.ratio == 0.5)
  expect(s.level == Level.Info)
  expect(s.timeout == Duration.seconds(30))
  expect(s.hosts == ["a", "b"])
  expect(s.limits.isEmpty())
  expect(s.note == null)
  expect(s.since == null)
  expect(s.name == "svc")
  expect(s.database.pool.size == 5)
  expect(s.database.pool.timeout == Duration.seconds(30))
}

test "every type is read from its text" {
  val vars = minimal([
    "PORT": "9000", "HOST": "localhost", "DEBUG": "true", "RATIO": "0.25",
    "LOG_LEVEL": "Warn", "TIMEOUT": "1m30s", "HOSTS": "x, y ,z", "LIMITS": "1,2, 3",
    "NOTE": "hello world", "SINCE": "2026-10-04T12:00:00Z",
    "DATABASE_POOL_SIZE": "20", "DATABASE_POOL_TIMEOUT": "250ms",
  ])
  val s = try read<Server>(vars)
  expect(s.port == 9000)
  expect(s.host == "localhost")
  expect(s.debug)
  expect(s.ratio == 0.25)
  expect(s.level == Level.Warn)
  expect(s.timeout == Duration.seconds(90))
  expect(s.hosts == ["x", "y", "z"])
  expect(s.limits == [1, 2, 3])
  expect(s.note == "hello world")
  expect(s.since != null)
  expect(s.database.pool.size == 20)
  expect(s.database.pool.timeout == Duration.millis(250))
}

test "a field is its name in capitals with underscores, nested names are joined" {
  val names = describe<Server>().map(v => v.name)
  expect(names == ["PORT", "HOST", "DEBUG", "RATIO", "LOG_LEVEL", "TIMEOUT", "HOSTS", "LIMITS", "NOTE", "SINCE", "NAME", "DATABASE_URL", "DATABASE_POOL_SIZE", "DATABASE_POOL_TIMEOUT"])
}

test "a prefix goes in front of every name" {
  val vars: Map<string, string> = ["APP_NAME": "svc", "APP_DATABASE_URL": "u", "APP_PORT": "81", "PORT": "99", "NAME": "wrong"]
  val s = try read<Server>(vars, prefix: "APP_")
  expect(s.port == 81)
  expect(s.name == "svc")
  expect(describe<Small>("X_").map(v => v.name) == ["X_WORKERS", "X_RETRIES"])
}

test "@key names a variable outright, and @key(env:) only for the environment" {
  val vars: Map<string, string> = ["TOKEN_FILE": "/run/token", "DATABASE_SIZE": "7", "DATABASE_TIMEOUT": "5s"]
  val t = try read<Tagged>(vars)
  expect(t.tokenPath == "/run/token")
  expect(t.db.size == 7)
  expect(t.db.timeout == Duration.seconds(5))
  expect(t.maybe == null)
  expect(describe<Tagged>().map(v => v.name) == ["TOKEN_FILE", "DATABASE_SIZE", "DATABASE_TIMEOUT", "MAYBE_SIZE", "MAYBE_TIMEOUT"])
}

test "a nested struct that can be null is there when any of its variables is" {
  expect((try read<Tagged>([:])).maybe == null)
  val t = try read<Tagged>(["MAYBE_SIZE": "9"])
  expect(t.maybe?.size == 9)
  expect(t.maybe?.timeout == Duration.seconds(30))
}

test "a secret reads like a string and prints as redacted" {
  val s = try read<Server>(minimal())
  expect(s.database.url.expose() == "postgres://u:p@h/db")
  expect("${s.database.url}" == "[redacted]")
}

test "an empty variable is a value" {
  val s = try read<Server>(minimal(["HOST": "", "NOTE": "", "HOSTS": ""]))
  expect(s.host == "")
  expect(s.note == "")
  expect(s.hosts.isEmpty())
  expect(failures<Server>(minimal(["PORT": ""])) == ["PORT: expected an integer, found \"\" (from the environment)"])
}

test "the environment beats later files, which beat earlier ones, which beat defaults" {
  val dir = scratch("order")
  val early = path.join(dir, "early.env")
  val late = path.join(dir, "late.env")
  try fs.writeFile(early, "PORT=1\nHOST=early\nDEBUG=true\n")
  try fs.writeFile(late, "PORT=2\nHOST=late\n")
  val s = try read<Server>(minimal(["PORT": "3"]), [early, late])
  expect(s.port == 3)
  expect(s.host == "late")
  expect(s.debug)
  expect(s.ratio == 0.5)
}

test "a JSON file gives typed values by field name, nested for nested structs" {
  val dir = scratch("json")
  val file = path.join(dir, "settings.json")
  try fs.writeFile(file, "{\"port\": 7000, \"debug\": true, \"hosts\": [\"p\", \"q\"], \"name\": \"from json\", \"database\": {\"url\": \"u\", \"pool\": {\"size\": 11, \"timeout\": \"2m\"}}, \"note\": null}")
  val s = try read<Server>([:], [file])
  expect(s.port == 7000)
  expect(s.debug)
  expect(s.hosts == ["p", "q"])
  expect(s.name == "from json")
  expect(s.note == null)
  expect(s.database.pool.size == 11)
  expect(s.database.pool.timeout == Duration.seconds(120))
  // the environment still wins, one variable at a time
  val mixed = try read<Server>(["PORT": "1", "DATABASE_POOL_SIZE": "2"], [file])
  expect(mixed.port == 1)
  expect(mixed.database.pool.size == 2)
  expect(mixed.database.pool.timeout == Duration.seconds(120))
}

test "files are skipped when missing, and a dotenv file may sit beneath a JSON one" {
  val dir = scratch("mixed")
  val dotenv = path.join(dir, "base.env")
  val json = path.join(dir, "over.json")
  try fs.writeFile(dotenv, "PORT=5\nHOST=base\n")
  try fs.writeFile(json, "{\"host\": \"over\"}")
  val s = try read<Server>(minimal(), [dotenv, path.join(dir, "missing.env"), json])
  expect(s.port == 5)
  expect(s.host == "over")
}

test "every problem comes at once, by variable name, with where it came from" {
  val dir = scratch("problems")
  val file = path.join(dir, "bad.env")
  try fs.writeFile(file, "RATIO=lots\nLOG_LEVEL=Loud\n")
  val got = failures<Server>(["PORT": "eighty", "DEBUG": "yes", "TIMEOUT": "soon", "LIMITS": "1,x,3", "DATABASE_POOL_SIZE": "-"], [file])
  expect(got.contains("PORT: expected an integer, found \"eighty\" (from the environment)"))
  expect(got.contains("DEBUG: expected true or false, found \"yes\" (from the environment)"))
  expect(got.contains("RATIO: expected a number, found \"lots\" (from ${file})"))
  expect(got.contains("LIMITS: expected an integer, found \"x\" (from the environment)"))
  expect(got.contains("DATABASE_POOL_SIZE: expected an integer, found \"-\" (from the environment)"))
  expect(got.contains("NAME: not set (expected text)"))
  expect(got.contains("DATABASE_URL: not set (expected text)"))
  expect(got.any(p => p.startsWith("LOG_LEVEL:") && p.contains("Loud") && p.endsWith("(from ${file})")))
  expect(got.any(p => p.startsWith("TIMEOUT:") && p.contains("soon")))
  expect(got.len() == 9)
}

test "a number out of range is a problem naming the variable" {
  val got = failures<Small>(["WORKERS": "300", "RETRIES": "9999999999"])
  expect(got.len() == 2)
  expect(got.any(p => p.startsWith("WORKERS:") && p.contains("300")))
  expect(got.any(p => p.startsWith("RETRIES:") && p.contains("9999999999")))
}

test "a secret's value is in no message" {
  val alone = failures<Database>(["URL_X": "unused"])
  expect(alone == ["URL: not set (expected text)"])
  val enough = failures<Server>(minimal(["PORT": "p0rt"]))
  expect(!enough.any(p => p.contains("postgres")))
}

test "a file that cannot be parsed is a problem and the rest still loads" {
  val dir = scratch("broken")
  val dotenv = path.join(dir, "broken.env")
  val json = path.join(dir, "broken.json")
  try fs.writeFile(dotenv, "OK=1\nnot a variable\n")
  try fs.writeFile(json, "{\"port\": ")
  val got = failures<Server>(minimal(), [dotenv, json])
  expect(got.any(p => p.startsWith(dotenv) && p.contains("line 2")))
  expect(got.any(p => p.startsWith(json)))
}

test "a JSON file that is not an object is a problem" {
  val dir = scratch("array")
  val file = path.join(dir, "list.json")
  try fs.writeFile(file, "[1, 2]")
  expect(failures<Server>(minimal(), [file]) == ["${file}: expected an object of settings"])
}

test "describe lists the variables with their types, defaults and secrecy" {
  val vars = describe<Server>()
  val port = vars.at(0) ?: panic("PORT")
  expect(port.name == "PORT")
  expect(port.what == "an integer")
  expect(!port.required)
  expect(port.default == "8080")
  val level = vars.at(4) ?: panic("LOG_LEVEL")
  expect(level.what == "one of \"Debug\", \"Info\", \"Warn\"")
  expect(level.default == "Info")
  val name = vars.at(10) ?: panic("NAME")
  expect(name.required)
  expect(name.default == null)
  val url = vars.at(11) ?: panic("DATABASE_URL")
  expect(url.name == "DATABASE_URL")
  expect(url.secret)
  expect(url.required)
  val hosts = vars.at(6) ?: panic("HOSTS")
  expect(hosts.default == "a,b")
}

test "a type that no variable can hold panics at the caller, naming the field" {
  expectPanics(() => describe<WithMap>())
  expectPanics(() => failures<WithMap>([:]))
}

test "the real environment is read by load" {
  // PATH is set on every machine this runs on
  val real = os.env("PATH") ?: panic("PATH is set on every machine this runs on")
  val s = try load<OnlyPath>()
  expect(s.path == real)
}

struct OnlyPath {
  @key("PATH")
  path: string
  implement Decodable
}

test fun scratch(name: string): string {
  val dir = path.join(os.tempDir(), "veles-config-test-$name")
  when (makeDir(dir)) {
    is Err(e) => panic("cannot make ${dir}: ${e.message()}")
    is Ok(_)  => dir
  }
}

test fun makeDir(dir: string) throws IoError {
  if (!fs.isDir(dir)) try fs.mkdir(dir)
}
