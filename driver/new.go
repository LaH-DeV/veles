package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// New is `veles new <dir>`: a package that runs and tests on the first
// try — a manifest, a main module with a function, `main` and a test that
// passes, and a .gitignore for what `veles build` writes. The package is
// named after the directory. An existing directory is used only when it
// is empty, so nothing is ever overwritten.
// Templates are what `veles new --template` offers: each a `main.vs` that
// builds, runs and passes its tests as created.
var Templates = []string{"app", "server"}

func New(dir, template string) int {
	name := filepath.Base(filepath.Clean(dir))
	if !packageName(name) {
		fmt.Fprintf(os.Stderr, "veles new: %q is not a package name: start with a letter, then letters, digits or '_' (other packages write it in `use`)\n", name)
		return 2
	}
	var main, next string
	switch template {
	case "", "app":
		main = appTemplate
		next = "  veles run       # Hello, world!\n  veles test      # runs the test in main.vs\n"
	case "server":
		main = strings.ReplaceAll(serverTemplate, "NAME", name)
		next = "  veles run       # serves on http://127.0.0.1:8080/ (HOST, PORT)\n  veles test      # the handlers, in memory\n"
	default:
		fmt.Fprintf(os.Stderr, "veles new: there is no template %q; there are: %s\n", template, strings.Join(Templates, ", "))
		return 2
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		fmt.Fprintf(os.Stderr, "veles new: %s is not empty; choose a new directory\n", dir)
		return 1
	}
	files := map[string]string{
		"veles.toml": "[package]\nname = \"" + name + "\"\nversion = \"0.1.0\"\n",
		"main.vs":    main,
		".gitignore": "/" + name + "\n/" + name + ".exe\n*.ll\n",
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "veles new:", err)
		return 1
	}
	for _, f := range []string{"veles.toml", "main.vs", ".gitignore"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(files[f]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "veles new:", err)
			return 1
		}
	}
	fmt.Printf("created package %s in %s\n\n  cd %s\n%s", name, dir, dir, next)
	return 0
}

const appTemplate = `use io

/// What the program says to ` + "`name`" + `.
fun greeting(name: string): string = "Hello, $name!"

fun main() {
  io.println(greeting("world"))
}

test "greets by name" {
  expect(greeting("Veles") == "Hello, Veles!")
}
`

// serverTemplate is an HTTP service with what one needs from its first
// day: a health check, request logging (serve's own), a request id and a
// time limit per request, the address from the environment, and a
// graceful stop on Ctrl+C or SIGTERM — and tests of its handlers that run
// in memory, without a port. NAME is the package's name.
const serverTemplate = `// NAME: an HTTP service.
//
//   veles run                       serves on http://127.0.0.1:8080/
//   HOST=0.0.0.0 PORT=9000 veles run
//   veles test                      the handlers, in memory
//
// Every request is logged to standard error. Ctrl+C or SIGTERM stops it
// gracefully: it stops accepting, lets the requests in flight finish, and
// returns.
use http, io, json, net, os

/// What ` + "`GET /api/hello`" + ` answers.
struct Greeting {
  message: string
  implement Codable
}

/// The routes, as one handler: what ` + "`serve`" + ` runs and what the tests call.
fun app(): http.Handler {
  val router = http.Router()
  // every answer carries a request id, and no handler runs past 5 seconds
  router.wrap(http.requestId())
  router.wrap(http.timeout(Duration.seconds(5)))

  // for load balancers and orchestrators: up while the process serves
  router.get("/healthz", req => http.Response.text("ok"))

  router.get("/api/hello", req => {
    val name = req.query.get("name") ?: "world"
    http.Response.json(try json.encode(Greeting(message: "Hello, $name!")))
  })

  router.handler()
}

fun main() throws {
  val host = os.env("HOST") ?: "127.0.0.1"
  val port = os.env("PORT")?.toInt() ?: 8080
  with (listener = try net.listen(host, port)) {
    io.println("listening on http://$host:${listener.port()}/ — Ctrl+C stops it")
    http.serve(listener, app(), stop: () => os.shutdownSignal())
    io.println("stopped")
  }
}

test "healthz says the service is up" {
  val resp = http.call(app(), http.Method.get, "/healthz")
  expect(resp.status == http.Status.ok)
  expect(resp.body.decodeUtf8() == "ok")
}

test "hello greets by name" {
  val resp = http.call(app(), http.Method.get, "/api/hello?name=Veles")
  expect(resp.status == http.Status.ok)
  val greeting = require(json.decode<Greeting>(resp.body.decodeUtf8() ?: ""))
  expect(greeting.message == "Hello, Veles!")
}

test "an unknown path is a 404" {
  val resp = http.call(app(), http.Method.get, "/nowhere")
  expect(resp.status == http.Status.notFound)
}
`

// packageName reports whether s can name a package: an identifier, since
// a package that depends on it writes the name in `use name.module`.
func packageName(s string) bool {
	if s == "" || !(s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z') {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_')
	}) < 0
}
