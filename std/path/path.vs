/// Path manipulation as text; nothing here touches the file system.
///
/// Both `/` and `\` separate components on input; output uses `/`, which
/// every platform's file API accepts.

/// Joins the parts with separators: `join("a", "b", "c")` is `a/b/c`. An
/// empty part adds nothing; an absolute part starts over. A list joins with
/// `join(parts...)`.
public fun join(parts: string...): string {
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
public fun dir(p: string): string {
  val i = lastSep(p)
  if (i < 0) "" else if (i == 0) "/" else p.substring(0, i) ?: p
}

/// Everything after the last separator.
public fun base(p: string): string {
  val i = lastSep(p)
  if (i < 0) p else p.substring(i + 1, p.len()) ?: p
}

/// The extension of the last component including its dot (`".vs"`), or `""`.
public fun ext(p: string): string {
  val name = base(p)
  val i = name.lastIndexOf(".")
  if (i <= 0) "" else name.substring(i, name.len()) ?: ""
}

/// The last component without its extension.
public fun stem(p: string): string {
  val name = base(p)
  val i = name.lastIndexOf(".")
  if (i <= 0) name else name.substring(0, i) ?: name
}

/// True for `/x`, `C:\x`, `C:/x` and `\server\x`.
public fun isAbsolute(p: string): bool {
  if (p.isEmpty()) return false
  if (isSep(p.byteAt(0))) return true
  p.len() >= 3 && p.byteAt(1) == 58 && isSep(p.byteAt(2))  // drive letter, ':'
}

fun isSep(b: u8): bool => b == 47 || b == 92

fun lastSep(p: string): i64 {
  var i = p.len() - 1
  loop (i >= 0) {
    if (isSep(p.byteAt(i))) return i
    i -= 1
  }
  -1
}

/// The shortest path that names the same file, worked out from the text
/// alone: separators collapse to one `/`, `.` components go, and `a/..`
/// cancels. A `..` that climbs above the start of a relative path is kept
/// (`../x`); one that climbs above a root is dropped (`/..` is `/`). The
/// empty path is `.`. Symbolic links are not consulted, so `a/link/..` is
/// `a` even when `link` points somewhere else.
public fun clean(p: string): string {
  if (p.isEmpty()) return "."
  val root = rootOf(p)
  var parts: MutableList<string> = []
  loop (seg in p.substring(root.len(), p.len())?.replace("\\", "/")?.split("/") ?: []) {
    when {
      seg.isEmpty() || seg == "." => continue
      seg != ".." => parts.push(seg)
      !parts.isEmpty() && parts.at(-1) != ".." => parts.pop()
      root.isEmpty() => parts.push("..")
    }
  }
  val body = parts.join("/")
  if (root.isEmpty()) {
    if (body.isEmpty()) "." else body
  } else {
    root + body
  }
}

/// True when `p` is `root` itself or lies inside it, comparing the
/// cleaned paths component by component: `/srv/www2` is not within
/// `/srv/www`, and `/srv/www/../etc` is not within anything under `/srv/www`.
/// This is the check that keeps a path built from a request inside the
/// directory it was meant for:
///
///     val file = path.join(root, requested)
///     if (!path.within(root, file)) throw forbidden()
///
/// Lexical, like `clean`: a symbolic link inside `root` that points out of
/// it is not detected. Both separators count, on every platform, so
/// `..\secret` is caught on Linux as well as on Windows.
public fun within(root: string, p: string): bool {
  val r = clean(root)
  val q = clean(p)
  if (r == ".") return !isAbsolute(q) && q != ".." && !q.startsWith("../")
  if (q == r) return true
  q.startsWith(if (r.endsWith("/")) r else r + "/")
}

// rootOf is the part of p that `..` cannot climb above, written with `/`
// but as long as it is in p (clean skips that many bytes): "/" for `/x` and
// `\x`, "C:/" for `C:\x`, and for a network path `\\server\share\x` the
// server and share too, which Windows treats as the root the way it treats
// a drive letter; "" for a relative path.
fun rootOf(p: string): string {
  if (p.len() >= 3 && p.byteAt(1) == ':' && isSep(p.byteAt(2))) return (p.substring(0, 2) ?: "") + "/"
  if (p.len() >= 2 && isSep(p.byteAt(0)) && isSep(p.byteAt(1))) {
    var end = 2
    var seps = 0
    loop (end < p.len() && seps < 2) {
      if (isSep(p.byteAt(end))) seps += 1
      end += 1
    }
    return (p.substring(0, end) ?: p).replace("\\", "/")
  }
  if (isSep(p.byteAt(0))) return "/"
  ""
}
