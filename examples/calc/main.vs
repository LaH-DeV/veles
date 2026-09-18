// An expression calculator: a tokenizer, a Pratt parser into a sealed AST,
// an evaluator with variables and a few functions, and a tiny REPL-style
// driver over a fixed script. Errors carry the column they were found at.
use io

// ---------------------------------------------------------------------------
// tokens

sealed trait Token
struct Num : Token {
  value: f64
}
struct Name : Token {
  text: string
}
struct Op : Token {
  text: string
}
struct End : Token { }

error SyntaxError {
  message: string
  col:     i64
  fun message(): string = "${self.message} (column ${self.col + 1})"
}

fun isDigit(b: u8): bool = b >= '0' && b <= '9'
fun isAlpha(b: u8): bool = (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'

struct Lexer {
  src:  string
  pos:  i64 = 0
  toks: MutableList<(Token, i64)> = []

  mut fun run(): List<(Token, i64)> throws SyntaxError {
    loop (self.pos < self.src.len()) {
      val b = self.src.byteAt(self.pos)
      val start = self.pos
      when {
        b == ' ' || b == '\t'  => self.pos += 1
        isDigit(b) || b == '.' => {
          loop (self.pos < self.src.len() && (isDigit(self.src.byteAt(self.pos)) || self.src.byteAt(self.pos) == '.')) self.pos += 1
          val text = self.src.substring(start, self.pos) ?: ""
          val v = text.toF64() ?: throw SyntaxError(message: "bad number '$text'", col: start)
          self.toks.push((Num(value: v), start))
        }
        isAlpha(b) => {
          loop (self.pos < self.src.len() && (isAlpha(self.src.byteAt(self.pos)) || isDigit(self.src.byteAt(self.pos)))) self.pos += 1
          self.toks.push((Name(text: self.src.substring(start, self.pos) ?: ""), start))
        }
        "+-*/^%(),=".contains(self.src.substring(start, start + 1) ?: "?") => {
          self.pos += 1
          self.toks.push((Op(text: self.src.substring(start, self.pos) ?: ""), start))
        }
        else => throw SyntaxError(message: "unexpected character '${self.src.substring(start, start + 1)}'", col: start)
      }
    }
    self.toks.push((End(), self.pos))
    self.toks.toList()
  }
}

// ---------------------------------------------------------------------------
// syntax tree

sealed trait Expr
struct Literal : Expr {
  value: f64
}
struct Variable : Expr {
  name: string
}
struct Unary : Expr {
  op:      string
  operand: *Expr
}
struct Binary : Expr {
  op:    string
  left:  *Expr
  right: *Expr
}
struct Call : Expr {
  name: string
  args: List<Expr>
}
struct Assign : Expr {
  name:  string
  value: *Expr
}

/// Binding power of an infix operator; 0 when `t` is not one.
fun infixPower(t: Token): (i64, i64) = when (t) {
  is Op(text)     => when (text) {
    "+", "-"      => (10, 11)
    "*", "/", "%" => (20, 21)
    "^"           => (31, 30)  // right-associative
    else          => (0, 0)
  }
  else => (0, 0)
}

struct Parser {
  toks: List<(Token, i64)>
  pos:  i64 = 0

  fun peek(): Token = self.toks.atOrPanic(self.pos).0
  fun col(): i64 = self.toks.atOrPanic(self.pos).1

  mut fun next(): Token {
    val t = self.peek()
    if (self.pos < self.toks.len() - 1) self.pos += 1
    t
  }

  mut fun expectOp(text: string) throws SyntaxError {
    val t = self.next()
    if (!(t is Op) || t.text != text) throw SyntaxError(message: "expected '$text'", col: self.col())
  }

  mut fun parseAll(): Expr throws SyntaxError {
    val e = try self.parseExpr(0)
    if (!(self.peek() is End)) throw SyntaxError(message: "unexpected token", col: self.col())
    e
  }

  mut fun parseExpr(minPower: i64): Expr throws SyntaxError {
    var left = try self.parsePrefix()
    loop {
      val t = self.peek()
      if (t is Op && t.text == "=") {
        val target = left
        if (!(target is Variable)) throw SyntaxError(message: "can only assign to a name", col: self.col())
        self.next()
        val value = try self.parseExpr(0)
        return Assign(name: target.name, value: &value)
      }
      val (lp, rp) = infixPower(t)
      if (lp == 0 || lp < minPower) break
      val op = self.next()
      if (!(op is Op)) break
      val right = try self.parseExpr(rp)
      // box the current `left`, not the variable: `&left` would point at
      // the variable being assigned, making the tree a cycle
      val boxed = left
      left = Binary(op: op.text, left: &boxed, right: &right)
    }
    left
  }

  mut fun parsePrefix(): Expr throws SyntaxError {
    val col = self.col()
    val t = self.next()
    when (t) {
      is Num(value) => Literal(value: value)
      is Name(text) => {
        val nt = self.peek()
        if (nt is Op && nt.text == "(") {
          self.next()
          var args: MutableList<Expr> = []
          val first = self.peek()
          if (!(first is Op && first.text == ")")) {
            loop {
              args.push(try self.parseExpr(0))
              val sep = self.peek()
              if (sep is Op && sep.text == ",") {
                self.next()
                continue
              }
              break
            }
          }
          try self.expectOp(")")
          Call(name: text, args: args.toList())
        } else {
          Variable(name: text)
        }
      }
      is Op(text) if text == "-" => Unary(op: "-", operand: &(try self.parseExpr(25)))
      is Op(text) if text == "(" => {
        val inner = try self.parseExpr(0)
        try self.expectOp(")")
        inner
      }
      is End => throw SyntaxError(message: "unexpected end of expression", col: col)
      else   => throw SyntaxError(message: "unexpected token", col: col)
    }
  }
}

fun parse(src: string): Expr throws SyntaxError {
  var lx = Lexer(src: src)
  val toks = try lx.run()
  var p = Parser(toks: toks)
  try p.parseAll()
}

// ---------------------------------------------------------------------------
// evaluation

error EvalError {
  message: string
}

struct Env {
  vars: MutableMap<string, f64> = [:]

  fun eval(e: Expr): f64 throws EvalError = when (e) {
    is Literal(value)          => value
    is Variable(name)          => self.vars.get(name) ?: throw EvalError(message: "unknown variable '$name'")
    is Unary(operand)          => -(try self.eval(*operand))
    is Binary(op, left, right) => {
      val l = try self.eval(*left)
      val r = try self.eval(*right)
      when (op) {
        "+"  => l + r
        "-"  => l - r
        "*"  => l * r
        "/"  => if (r == 0.0) throw EvalError(message: "division by zero") else l / r
        "%"  => l.mod(r)
        "^"  => l.pow(r)
        else => throw EvalError(message: "unknown operator '$op'")
      }
    }
    is Call(name, args) => {
      val vals = try args.map(a => try self.eval(a))
      try self.call(name, vals)
    }
    is Assign(name, value) => {
      val v = try self.eval(*value)
      self.vars.set(name, v)
      v
    }
  }

  fun call(name: string, args: List<f64>): f64 throws EvalError {
    val arity = when (name) {
      "sqrt", "abs", "floor" => 1
      "max", "min", "pow"    => 2
      "sum"                  => -1
      else                   => throw EvalError(message: "unknown function '$name'")
    }
    if (arity >= 0 && args.len() != arity) throw EvalError(message: "'$name' takes $arity argument(s), got ${args.len()}")
    val a = args.at(0) ?: 0.0
    val b = args.at(1) ?: 0.0
    when (name) {
      "sqrt"  => if (a < 0.0) throw EvalError(message: "sqrt of a negative number") else a.sqrt()
      "abs"   => a.abs()
      "floor" => a.floor()
      "max"   => a.max(b)
      "min"   => a.min(b)
      "pow"   => a.pow(b)
      "sum"   => args.sum()
      else    => 0.0
    }
  }
}

fun show(e: Expr): string = when (e) {
  is Literal(value)          => if (value == value.trunc()) "${value as i64}" else "$value"
  is Variable(name)          => name
  is Unary(op, operand)      => "($op${show(*operand)})"
  is Binary(op, left, right) => "(${show(*left)} $op ${show(*right)})"
  is Call(name, args)        => "$name(${args.map(a => show(a)).join(", ")})"
  is Assign(name, value)     => "$name = ${show(*value)}"
}

fun render(x: f64): string = if (x == x.trunc() && x.abs() < 1.0e15) "${x as i64}" else x.toFixed(4)

fun main() {
  val script = [
    "1 + 2 * 3",
    "1 - 2 - 3 - 4",
    "2 * 3 * 4 - 5 - 6",
    "(1 + 2) * 3",
    "2 ^ 3 ^ 2",
    "-2 ^ 2",
    "10 % 4 + 7 / 2",
    "x = 5",
    "y = x * 2 + 1",
    "sqrt(x * x + y * y)",
    "max(x, y) - min(x, y)",
    "sum(1, 2, 3, 4) / 4",
    "z + 1",
    "1 / (x - 5)",
    "sqrt(-1)",
    "2 +",
    "3 $ 4",
    "pow(2, 10, 3)",
    "(1 + 2",
  ]
  var env = Env()
  loop (line in script) {
    val ast = parse(line)
    if (ast.err) {
      io.println("$line  => syntax error: ${ast.message()}")
      continue
    }
    val result = env.eval(ast)
    if (result.err) {
      io.println("${show(ast)}  => error: ${result.message()}")
      continue
    }
    io.println("${show(ast)}  => ${render(result)}")
  }
  io.println("variables: ${env.vars}")
}
