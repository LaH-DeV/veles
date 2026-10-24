// D205: `codec.KeyStyle` became the prelude's `Case`, for keys and enum text
// alike. The old spellings are errors whose fixes write the new ones.
use io
use codec

fun main() {
  val a = codec.KeyStyle.SnakeCase // error: 'KeyStyle' was replaced by 'Case' (D205): write 'Case.Snake'
  val b = KeyStyle.UpperSnake // error: 'KeyStyle' was replaced by 'Case' (D205): write 'Case.ScreamingSnake'
  val c: KeyStyle = Case.Camel // error: 'KeyStyle' was replaced by 'Case' (D205): write 'Case'
  val d: codec.KeyStyle = Case.Kebab // error: 'KeyStyle' was replaced by 'Case' (D205): write 'Case'
  val e = KeyStyle.Shouting // error: 'KeyStyle' was replaced by 'Case' (D205): its members are Case.AsWritten, Case.Snake
  io.println(codec.styleKey("userId", a) + codec.styleKey("userId", b) + codec.styleKey("user_id", c) + codec.styleKey("userId", d))
}
