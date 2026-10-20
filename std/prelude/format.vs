// What the compiler calls for "${x:spec}" (D171). It reads the spec at compile
// time, checks it against the type of `x`, and calls one of these with the
// parts, so nothing here parses a format string or can be given a wrong one.
// They are not public: the way to format is the spec itself.

// the alignments the compiler passes: 1 left, 2 right, 3 centred
const ALIGN_LEFT: i64 = 1
const ALIGN_RIGHT: i64 = 2
const ALIGN_CENTRE: i64 = 3

// `text` padded with `fill` (one character) to `width` characters — code
// points, so a table of names with accents lines up.
fun fmtPad(text: string, fill: string, align: i64, width: i64): string {
  val n = text.chars().len()
  if (n >= width || fill.isEmpty()) return text
  val gap = width - n
  var left = 0
  if (align == ALIGN_RIGHT) left = gap else if (align == ALIGN_CENTRE) left = gap / 2
  fill.repeat(left) + text + fill.repeat(gap - left)
}

// "${s:<10}": the text, aligned in `width` characters
fun fmtText(text: string, fill: string, align: i64, width: i64): string => fmtPad(text, fill, align, width)

// the digits of `body` in groups of three from the right of the integer part
fun fmtGroup(body: string): string {
  val dot = body.indexOf(".")
  val whole = if (dot < 0) body else body.substring(0, dot) ?: body
  val rest = if (dot < 0) "" else body.substring(dot, body.len()) ?: ""
  val out = StringBuilder()
  val n = whole.len()
  loop (i in 0..<n) {
    if (i > 0 && (n - i) % 3 == 0) out.append(",")
    out.append(whole.substring(i, i + 1) ?: "")
  }
  out.toString() + rest
}

// "${n:+08,d}": a number already written as digits (a leading `-` for a
// negative one). `sign` is 0 none, 1 plus, 2 space for a non-negative number;
// `prefix` is "0x"-like text after the sign; `suffix` is "%".
fun fmtNum(text: string, fill: string, align: i64, sign: i64, prefix: string, zero: bool, group: bool, upper: bool, suffix: string, width: i64): string {
  val neg = text.startsWith("-")
  var body = if (neg) text.substring(1, text.len()) ?: text else text
  if (upper) body = body.toUpper()
  if (group) body = fmtGroup(body)
  val signText = when {
    neg       => "-"
    sign == 1 => "+"
    sign == 2 => " "
    else      => ""
  }
  val head = signText + prefix
  if (zero) {
    val used = (head + body + suffix).chars().len()
    if (used < width) body = "0".repeat(width - used) + body
    return head + body + suffix
  }
  fmtPad(head + body + suffix, fill, if (align == 0) ALIGN_RIGHT else align, width)
}
