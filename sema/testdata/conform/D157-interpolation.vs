use io

error ParseError { }

fun parse(): i64 throws ParseError => throw ParseError()

fun work(): i64 => 42

@template
fun text(parts: List<string>, values: List<Result<i64, ParseError>>): string => "${parts.len()} ${values.len()}"

fun main() {
  val result = parse()
  io.println("$result")                    // error: cannot interpolate
  io.println("${parse()}")                 // error: cannot interpolate
  io.println("${parse():>8}")              // error: cannot interpolate
  io.println(text"${parse()}")             // error: cannot interpolate
  io.println(result.toString(1))           // error: takes no arguments or type arguments
  io.println(result.toString<i64>())       // error: takes no arguments or type arguments
  io.println("${try parse().toString()}")  // error: 'try' needs a Result
  with task = async work()
  io.println("$task")  // error: cannot interpolate task handle
}
