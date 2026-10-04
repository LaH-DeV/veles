// A service's settings in one struct: read from the environment, a dotenv
// file and a JSON file beneath it, with a usage table made from the same
// struct and every problem reported at once.
//
// Run it with DEMO_PORT=9999 set and the port below changes: the environment
// wins over both files. The expected output assumes no DEMO_ variables.

use config, fs, io { println }, os, path

enum Level {
  Debug
  Info
  Warn
  Error
}

struct Pool {
  size:    i64 = 10
  timeout: Duration = Duration.seconds(5)
  implement Decodable
}

struct Settings {
  port: i64 = 8080
  host: string = "127.0.0.1"
  @key("LOG_LEVEL")
  level: Level = Level.Info
  origins: List<string> = []
  apiKey:  Secret<string>
  pool:    Pool = Pool()
  implement Decodable
}

fun usage() {
  println("variables (all start with DEMO_):")
  loop (v in config.describe<Settings>("DEMO_")) {
    val given = v.default ?: ""
    val kind = if (v.required) "required" else "default ${if (given.isEmpty()) "(empty)" else given}"
    println("  ${v.name.padEnd(18)} ${kind.padEnd(20)} ${v.what}")
  }
}

fun show(s: Settings) {
  println("  listening on ${s.host}:${s.port}, logging at ${s.level}")
  println("  origins: ${s.origins}")
  println("  pool of ${s.pool.size}, timeout ${s.pool.timeout}")
  println("  api key: ${s.apiKey} (${s.apiKey.expose().len()} bytes)")
}

fun main() throws IoError {
  usage()
  val dir = path.join(os.tempDir(), "veles-config-example")
  if (!fs.isDir(dir)) try fs.mkdir(dir)

  // a dotenv file for the machine, a JSON file for the deployment on top of it
  val base = path.join(dir, "base.env")
  val deploy = path.join(dir, "deploy.json")
  try fs.writeFile(base, "# local settings\nDEMO_HOST=0.0.0.0\nDEMO_PORT=9090\nDEMO_LOG_LEVEL=Debug\nDEMO_API_KEY='s3cr3t-with-$-sign'\nDEMO_ORIGINS=https://a.example, https://b.example  # two\n")
  try fs.writeFile(deploy, "{\"port\": 8443, \"pool\": {\"size\": 32, \"timeout\": \"750ms\"}}")

  println("loaded from both files:")
  when (config.load<Settings>(files: [base, deploy], prefix: "DEMO_")) {
    is Ok(s)  => show(s)
    is Err(e) => println(e.message())
  }

  // wrong in four places and missing in one: all of it in one answer
  val broken = path.join(dir, "broken.env")
  try fs.writeFile(broken, "DEMO_PORT=eighty\nDEMO_LOG_LEVEL=Loud\nDEMO_POOL_SIZE=many\nDEMO_POOL_TIMEOUT=soon\n")
  println("loaded from a broken file:")
  when (config.load<Settings>(files: [broken], prefix: "DEMO_")) {
    is Ok(s)  => show(s)
    is Err(e) => {
      loop (p in e.problems) {
        println("  ${p.toString().replace(broken, "broken.env")}")
      }
    }
  }
  try fs.remove(base)
  try fs.remove(deploy)
  try fs.remove(broken)
}
