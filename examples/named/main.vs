use io
// use io { println :: write, eprintln :: writeErr, readLine }
// use io { println : write, readLine }
// use io { println as write, readLine }
// from io import { println as write, readLine }
// from io import { println::write, readLine }

fun main() {
  io.println("Named imports could be good")
  io.println("I think from io import { println::write, readLine } looks good")
  io.println("And in that case we have two syntaxes for imports, but one is import-ing, second one is use-ing")
  io.println("Only import would be possible to get named and renamed, use would work as currently (no named uses)")
}
