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
  fun message(): string = "${this.message} (column ${this.col + 1})"
}

fun isDigit(b: u8): bool = b >= '0' && b <= '9'
fun isAlpha(b: u8): bool = (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'

type Position = i64

struct Lexer {
  src:     string
  var pos: Position = 0
  toks:    MutableList<(Token, Position)> = []

  fun run(): List<(Token, Position)> throws SyntaxError {
    loop (this.pos < this.src.len()) {
      val b = this.src.byteAt(this.pos)
      val start = this.pos
      when {
        b == ' ' || b == '\t' => this.pos += 1
        isDigit(b) || b == '.' => {
          loop (this.pos < this.src.len() && (isDigit(this.src.byteAt(this.pos)) || this.src.byteAt(this.pos) == '.')) this.pos += 1
          val text = this.src.substring(start, this.pos) ?: ""
          val v = text.toF64() ?: throw SyntaxError(message: "bad number '$text'", col: start)
          this.toks.push((Num(value: v), start))
        }
        isAlpha(b) => {
          loop (this.pos < this.src.len() && (isAlpha(this.src.byteAt(this.pos)) || isDigit(this.src.byteAt(this.pos)))) this.pos += 1
          this.toks.push((Name(text: this.src.substring(start, this.pos) ?: ""), start))
        }
        "+-*/^%(),=".contains(this.src.substring(start, start + 1) ?: "?") => {
          this.pos += 1
          this.toks.push((Op(text: this.src.substring(start, this.pos) ?: ""), start))
        }
        else => throw SyntaxError(message: "unexpected character '${this.src.substring(start, start + 1)}'", col: start)
      }
    }
    this.toks.push((End(), this.pos))
    this.toks.toList()
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
  is Op(text) => when (text) {
    "+", "-"      => (10, 11)
    "*", "/", "%" => (20, 21)
    "^"           => (31, 30)  // right-associative
    else          => (0, 0)
  }
  else        => (0, 0)
}

struct Parser {
  private val tokens: List<(Token, Position)>
  private var pos:    Position = 0

  // the token list ends with an end token, and the parser never moves past it
  private fun here(): (Token, Position) = this.tokens.at(this.pos) ?: panic("calc: the parser stops at the end token")
  private fun peek(): Token = this.here().0
  private fun col(): Position = this.here().1

  private fun next(): Token {
    val t = this.peek()
    if (this.pos < this.tokens.lastIndex()) this.pos += 1
    t
  }

  private fun expectOp(text: string) throws SyntaxError {
    val t = this.next()
    if (t !is Op || t.text != text) throw SyntaxError(message: "expected '$text'", col: this.col())
  }

  private fun parseExpr(minPower: i64): Expr throws SyntaxError {
    var left = try this.parsePrefix()
    loop {
      val t = this.peek()
      if (t is Op && t.text == "=") {
        val target = left
        if (target !is Variable) throw SyntaxError(message: "can only assign to a name", col: this.col())
        this.next()
        val value = try this.parseExpr(0)
        return Assign(name: target.name, value: &value)
      }
      val (lp, rp) = infixPower(t)
      if (lp == 0 || lp < minPower) break
      val op = this.next()
      if (op !is Op) break
      val right = try this.parseExpr(rp)
      // box the current `left`, not the variable: `&left` would point at
      // the variable being assigned, making the tree a cycle
      val boxed = left
      left = Binary(op: op.text, left: &boxed, right: &right)
    }
    left
  }

  private fun parsePrefix(): Expr throws SyntaxError {
    val col = this.col()
    val t = this.next()
    when (t) {
      is Num(value) => Literal(value)
      is Name(text) => {
        val nt = this.peek()
        if (nt is Op && nt.text == "(") {
          this.next()
          var args: MutableList<Expr> = []
          val first = this.peek()
          if (!(first is Op && first.text == ")")) {
            loop {
              args.push(try this.parseExpr(0))
              val sep = this.peek()
              if (sep is Op && sep.text == ",") {
                this.next()
                continue
              }
              break
            }
          }
          try this.expectOp(")")
          Call(name: text, args: args.toList())
        } else {
          Variable(name: text)
        }
      }
      is Op(text)   => {
        when (text) {
          "-"  => Unary(op: "-", operand: &(try this.parseExpr(25)))
          "("  => {
            val inner = try this.parseExpr(0)
            try this.expectOp(")")
            inner
          }
          else => throw SyntaxError(message: "unexpected token", col)
        }
      }
      is End        => throw SyntaxError(message: "unexpected end of expression", col)
    }
  }

  fun parseAll(): Expr throws SyntaxError {
    val e = try this.parseExpr(0)
    if (this.peek() !is End) throw SyntaxError(message: "unexpected token", col: this.col())
    e
  }
}

fun parse(src: string): Expr throws SyntaxError {
  var lexer = Lexer(src)
  val tokens = try lexer.run()
  var parser = Parser(tokens)
  try parser.parseAll()
}

// ---------------------------------------------------------------------------
// evaluation

error EvalError {
  message: string
}

struct Env {
  private vars: MutableMap<string, f64> = [:]

  fun eval(e: Expr): f64 throws EvalError = when (e) {
    is Literal(value)          => value
    is Variable(name)          => this.vars.get(name) ?: throw EvalError(message: "unknown variable '$name'")
    is Unary(operand)          => -(try this.eval(*operand))
    is Binary(op, left, right) => {
      val l = try this.eval(*left)
      val r = try this.eval(*right)
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
    is Call(name, args)        => {
      val vals = try args.map(a => try this.eval(a))
      try this.call(name, vals)
    }
    is Assign(name, value)     => {
      val v = try this.eval(*value)
      this.vars.set(name, v)
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

  fun peek(): Map<string, f64> = this.vars.toMap()
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
  io.println("variables: ${env.peek()}")
}
