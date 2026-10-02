// D137: a static function of a generic struct, or one the prelude adds to a
// built-in collection, called without type arguments infers them from its
// arguments and the expected type, as a constructor does; when nothing pins
// one, the error asks for them.
use io

struct Box<T> {
  value: T
  public static fun of(v: T): Box<T> = Box(value: v)
  public static fun none(): Box<T>? = null
  public static fun build(f: fun(): T): Box<T> = Box(value: f())
}

fun main() {
  val text = Box.of("x")
  val absent: Box<i64>? = Box.none()
  val built = Box.build(() => 2.5)
  val flags = MutableList.repeat(false, 3)
  val squares = MutableList.make(4, i => i * i)
  val small: MutableList<i8> = MutableList.repeat(1, 2)
  io.println("${text.value} ${absent == null} ${built.value} $flags $squares $small")
  val unknown = Box.none() // error: cannot infer type parameter 'T' of 'Box' from this call; write the type arguments, e.g. 'Box<T>.none(...)', or annotate the binding
  io.println("${unknown == null}")
}
