/// `text` in double quotes for a message: the backslash, the quote and control
/// characters escaped.
fun quoted(text: string): string {
  val out = StringBuilder()
  out.append("\"")
  loop (byte in text.bytes()) {
    when (byte) {
      '"'  => out.append("\\\"")
      '\\' => out.append("\\\\")
      '\n' => out.append("\\n")
      '\t' => out.append("\\t")
      '\r' => out.append("\\r")
      else => {
        if (byte < 0x20 || byte == 0x7F) {
          out.append("\\x${byte.toI64().toString(radix: 16).padStart(2, "0")}")
        } else {
          out.appendByte(byte)
        }
      }
    }
  }
  out.append("\"")
  out.toString()
}
