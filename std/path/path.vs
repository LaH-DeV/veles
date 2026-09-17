/// Path manipulation as text; nothing here touches the file system.
///
/// Both `/` and `\` separate components on input; output uses `/`, which
/// every platform's file API accepts.

/// Joins the parts with separators: `join("a", "b", "c")` is `a/b/c`. An
/// empty part adds nothing; an absolute part starts over. A list joins with
/// `join(parts...)`.
pub fun join(parts: string...): string {
  var out = ""
  loop (p in parts) {
    out = join2(out, p)
  }
  out
}

fun join2(a: string, b: string): string {
  if (a.isEmpty() || isAbsolute(b)) return b
  if (b.isEmpty()) return a
  if (isSep(a.byteAt(a.len() - 1))) a + b else a + "/" + b
}

/// Everything before the last separator, `""` when there is none.
pub fun dir(p: string): string {
  val i = lastSep(p)
  if (i < 0) "" else if (i == 0) "/" else p.substring(0, i) ?: p
}

/// Everything after the last separator.
pub fun base(p: string): string {
  val i = lastSep(p)
  if (i < 0) p else p.substring(i + 1, p.len()) ?: p
}

/// The extension of the last component including its dot (`".vs"`), or `""`.
pub fun ext(p: string): string {
  val name = base(p)
  val i = name.lastIndexOf(".")
  if (i <= 0) "" else name.substring(i, name.len()) ?: ""
}

/// The last component without its extension.
pub fun stem(p: string): string {
  val name = base(p)
  val i = name.lastIndexOf(".")
  if (i <= 0) name else name.substring(0, i) ?: name
}

/// True for `/x`, `C:\x`, `C:/x` and `\server\x`.
pub fun isAbsolute(p: string): bool {
  if (p.isEmpty()) return false
  if (isSep(p.byteAt(0))) return true
  p.len() >= 3 && p.byteAt(1) == 58 && isSep(p.byteAt(2))  // drive letter, ':'
}

fun isSep(b: u8): bool = b == 47 || b == 92

fun lastSep(p: string): i64 {
  var i = p.len() - 1
  loop (i >= 0) {
    if (isSep(p.byteAt(i))) return i
    i -= 1
  }
  -1
}
