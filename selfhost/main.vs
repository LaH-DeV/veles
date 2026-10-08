// The self-hosted front end's driver (veles-selfhost-frontend-plan.md). The
// harness, selfhost/selfhost_test.go, runs it on every file of the corpus
// and compares what it prints with what the bootstrap compiler computes;
// byte-identical output is the gate of each phase.
//
//   selfhost --positions FILE   line:col of every byte offset, and one past the end (P0)
//   selfhost --lines FILE       the line count, then every line, and the ones on either side (P0)
//   selfhost --render FILE      a diagnostic every 37 bytes, rendered as `veles check` does (P0)
//   selfhost --tokens FILE      every token, comment and diagnostic of the scan (P1, gate G1)
//   selfhost --quote FILE       every line quoted as the tree printer quotes a string (P3)
//   selfhost --parse FILE       the tree as `veles parse` prints it, then every diagnostic (P5, gates G2, G3)
use fs, io, os
use ast { dump, quote }, lexer { kindText, tokenize }, parser { parseFile }, source { Diagnostics, File, Span }

fun usage(): Never {
  io.eprintln("usage: selfhost --positions|--lines|--render|--tokens|--quote|--parse FILE")
  os.exit(2)
}

fun positions(file: File): string {
  val out = StringBuilder()
  out.reserve(file.content.len() * 6)
  loop (offset in 0..file.content.len()) {
    val (line, col) = file.position(offset)
    out.appendLine("$line:$col")
  }
  out.toString()
}

fun lines(file: File): string {
  val out = StringBuilder()
  out.appendLine("${file.lineCount()}")
  loop (n in 0..(file.lineCount() + 1)) out.appendLine("$n|${file.line(n)}")
  out.toString()
}

// diagnostics at known places: every 37th byte, 0 to 4 bytes wide, errors
// and warnings in turn — the caret arithmetic at line ends, on `\r`, on
// empty lines and past the last line
fun render(file: File): string {
  val diags = Diagnostics()
  var offset: i64 = 0
  var k: i64 = 0
  loop (offset <= file.content.len()) {
    val span = Span(file, start: offset, end: offset + k % 5)
    if (k % 2 == 0) diags.errorAt(span, "probe $k") else diags.warnAt(span, "probe $k")
    offset += 37
    k += 1
  }
  diags.render()
}

// text on one line: the backslash, line breaks, tabs and the other control
// bytes escaped (the harness's own convention, the same on both sides)
fun escaped(text: string): string {
  val out = StringBuilder()
  loop (i in 0..<text.len()) {
    val b = text.byteAt(i)
    when (b) {
      '\\' => out.append("\\\\")
      '\n' => out.append("\\n")
      '\r' => out.append("\\r")
      '\t' => out.append("\\t")
      else => {
        if (b < 0x20 || b == 0x7F) out.append("\\x${b.toI64().toString(radix: 16).padStart(2, "0")}") else out.appendByte(b)
      }
    }
  }
  out.toString()
}

fun tokens(file: File): string {
  val diags = Diagnostics()
  val scan = tokenize(file, &diags)
  val out = StringBuilder()
  loop (t in scan.tokens) {
    out.appendLine("tok ${t.kind.value} ${kindText(t.kind)} ${t.span.start} ${t.span.end} ${if (t.autoSemi) 1 else 0} ${escaped(t.text)}")
    if (!t.doc.isEmpty()) out.appendLine("  doc ${escaped(t.doc)}")
    loop (part in t.parts) {
      if (part.isExpr) {
        out.appendLine("  expr ${part.span.start} ${part.span.end} ${escaped(part.expr)}")
      } else {
        out.appendLine("  text ${escaped(part.text)}")
      }
    }
  }
  loop (c in scan.comments) out.appendLine("comment ${c.span.start} ${c.span.end} ${escaped(c.text)}")
  out.appendLine("moduledoc ${escaped(scan.moduleDoc)}")
  loop (d in diags.items) out.appendLine("diag ${escaped("$d")}")
  out.toString()
}

// the tree, as `veles parse` prints it, then every diagnostic (P4–P5, gates G2 and G3)
fun parse(file: File): string {
  val diags = Diagnostics()
  val tree = parseFile(file, &diags)
  val out = StringBuilder()
  out.append(dump(tree))
  loop (d in diags.items) out.appendLine("diag ${escaped("$d")}")
  out.toString()
}

// every line quoted as the tree printer quotes a string (P3)
fun quoteLines(file: File): string {
  val out = StringBuilder()
  loop (n in 1..file.lineCount()) out.appendLine(quote(file.line(n)))
  out.toString()
}

fun main() throws IoError {
  val [mode, path] = os.args() else usage()
  val file = File.of(path, try fs.readFile(path))
  val text = when (mode) {
    "--positions" => positions(file)
    "--lines"     => lines(file)
    "--render"    => render(file)
    "--tokens"    => tokens(file)
    "--quote"     => quoteLines(file)
    "--parse"     => parse(file)
    else          => usage()
  }
  io.print(text)
}
