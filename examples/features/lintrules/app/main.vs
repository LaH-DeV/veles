use io { println }

fun Shout(text: string): string => text.toUpper()

fun main() {
  println(Shout("hello"))
  await sleep(Duration.millis(5))
}
