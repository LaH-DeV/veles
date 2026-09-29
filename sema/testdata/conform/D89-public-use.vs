// D89: `public use` re-exports the package's own modules; a standard module
// is not one. (The rest — facades, aliases, collisions, dependencies — needs
// several modules and is in sema/reexport_test.go.)
public use io // error: 'public use io' re-exports a module of another package

fun main() {
  io.println("re-exports")
}
