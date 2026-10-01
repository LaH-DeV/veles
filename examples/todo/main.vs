// A todo.txt manager: a command-line tool that keeps tasks in a plain text
// file, one per line, in the todo.txt format —
//
//   x 2026-09-19 (A) 2026-09-17 Call the bank +finance @phone due:2026-09-20
//   ^ done         ^ priority   ^ created     ^ +project @context key:value tags
//
// The file lives at $TODO_FILE, or todo.txt in the current directory. Tasks
// are numbered by line; `todo help` lists the commands.
use fs, io { print, println }, os, path, time

error UsageError {
  message: string
}

// ---------------------------------------------------------------------------
// the task line

struct Task {
  var done:        bool = false
  var completedOn: string = ""  // ISO dates, "" when absent
  var createdOn:   string = ""
  var priority:    string = ""  // "A".."Z" or ""
  var text:        string       // the description, tags included

  /// Parses one line of the file; anything is a task, so this cannot fail.
  static fun parse(line: string): Task {
    var words = line.split(" ").filter(w => !w.isEmpty())
    var t = Task(text: "")
    if (words.first() == "x") {
      t.done = true
      words = words.drop(1)
      val completed = words.first() ?: ""
      if (isDate(completed)) {
        t.completedOn = completed
        words = words.drop(1)
      }
    }
    val pri = words.first() ?: ""
    if (isPriority(pri)) {
      t.priority = pri.substring(1, 2) ?: ""
      words = words.drop(1)
    }
    val created = words.first() ?: ""
    if (isDate(created)) {
      t.createdOn = created
      words = words.drop(1)
    }
    t.text = words.join(" ")
    t
  }

  /// The line as it is stored: the inverse of `parse`.
  fun line(): string {
    val sb = StringBuilder()
    if (this.done) {
      sb.append("x ")
      if (!this.completedOn.isEmpty()) sb.append("${this.completedOn} ")
    }
    if (!this.priority.isEmpty()) sb.append("(${this.priority}) ")
    if (!this.createdOn.isEmpty()) sb.append("${this.createdOn} ")
    sb.append(this.text)
    sb.toString()
  }

  fun words(): List<string> = this.text.split(" ")
  fun projects(): List<string> = this.words().filter(w => w.startsWith("+") && w.len() > 1)
  fun contexts(): List<string> = this.words().filter(w => w.startsWith("@") && w.len() > 1)

  /// The value of a `key:value` tag, or null.
  fun tag(key: string): string? {
    val prefix = "$key:"
    val w = this.words().find(w => w.startsWith(prefix)) ?: return null
    w.substring(prefix.len(), w.len())
  }

  fun due(): string? {
    val d = this.tag("due") ?: return null
    if (isDate(d)) d else null
  }

  /// Every term must match: `+proj` and `@ctx` match a tag, `-word` excludes,
  /// anything else is a case-insensitive substring of the line.
  fun matches(terms: List<string>): bool = terms.all(term => when {
    term.startsWith("-") && term.len() > 1 => !this.matches([term.substring(1, term.len()) ?: ""])
    term.startsWith("+") => this.projects().contains(term)
    term.startsWith("@") => this.contexts().contains(term)
    else => this.line().toLower().contains(term.toLower())
  })
}

fun isDigit(b: u8): bool = b >= '0' && b <= '9'

/// `YYYY-MM-DD`, by shape only.
fun isDate(w: string?): bool {
  if (w == null) return false
  if (w.len() != 10) return false
  loop (i in 0..<10) {
    val b = w.byteAt(i)
    if (i == 4 || i == 7) {
      if (b != '-') return false
    } else if (!isDigit(b)) return false
  }
  true
}

/// `(A)` to `(Z)`.
fun isPriority(w: string): bool = w.len() == 3 && w.byteAt(0) == '(' && w.byteAt(2) == ')' && w.byteAt(1) >= 'A' && w.byteAt(1) <= 'Z'

// ---------------------------------------------------------------------------
// dates: only whole days matter here

/// Today as an ISO date; $TODO_TODAY overrides the clock for scripts and tests.
fun today(): string = os.env("TODO_TODAY") ?: time.now().local().date()

/// Days since 1970-01-01 for an ISO date (Howard Hinnant's days_from_civil).
fun dayNumber(date: string): i64 {
  val [year, m, d] = date.split("-").map(p => p.toInt() ?: 0) else panic("dayNumber: '$date' is not an ISO date")
  var y = year
  if (m <= 2) y -= 1
  val era = (if (y >= 0) y else y - 399) / 400
  val yoe = y - era * 400
  val doy = (153 * (if (m > 2) m - 3 else m + 9) + 2) / 5 + d - 1
  val doe = yoe * 365 + yoe / 4 - yoe / 100 + doy
  era * 146097 + doe - 719468
}

/// "overdue by 2 days", "due today", "due in 3 days".
fun describeDue(due: string, today: string): string {
  val days = dayNumber(due) - dayNumber(today)
  when {
    days < 0  => "overdue by ${-days} ${if (days == -1) "day" else "days"}"
    days == 0 => "due today"
    else      => "due in $days ${if (days == 1) "day" else "days"}"
  }
}

// ---------------------------------------------------------------------------
// the file

struct TodoFile {
  file:  string
  tasks: MutableList<Task> = []

  static fun open(file: string): TodoFile throws IoError {
    val t = TodoFile(file)
    if (fs.exists(file)) {
      loop (line in try fs.readFile(file).lines()) {
        if (!line.trim().isEmpty()) t.tasks.push(Task.parse(line))
      }
    }
    t
  }

  fun save() throws IoError {
    try fs.writeFile(this.file, this.tasks.map(t => t.line() + "\n").join(""))
  }

  /// Task numbers are 1-based line numbers.
  fun number(arg: string): i64 throws UsageError {
    val n = arg.toInt() ?: throw UsageError(message: "'$arg' is not a task number")
    if (n < 1 || n > this.tasks.len()) throw UsageError(message: "no task $n (${this.tasks.len()} in ${path.base(this.file)})")
    n
  }

  /// Task `n`, counting from 1. Every `n` comes from `number`, which accepts
  /// only `1..len`.
  fun task(n: i64): *Task = this.tasks.ref(n - 1) ?: panic("todo: number() accepts only 1..len")

  /// The numbered tasks, open ones first by priority, then by due date, then by number.
  fun listed(terms: List<string>, all: bool): List<(i64, Task)> {
    val rows: MutableList<(i64, Task)> = []
    loop ((i, t) in this.tasks.iter().enumerate()) {
      if ((all || !t.done) && t.matches(terms)) rows.push((i + 1, t))
    }
    rows.sortedWith((a, b) => {
      val (x, y) = (a.1, b.1)
      if (x.done != y.done) return if (x.done) Ordering.Greater else Ordering.Less
      val pri = sortKey(x.priority).compareTo(sortKey(y.priority))
      if (pri != Ordering.Equal) return pri
      val due = (x.due() ?: "9999").compareTo(y.due() ?: "9999")
      if (due != Ordering.Equal) return due
      a.0.compareTo(b.0)
    })
  }
}

fun sortKey(priority: string): string = if (priority.isEmpty()) "ZZ" else priority

fun printRows(rows: List<(i64, Task)>, total: i64, width: i64) {
  val now = today()
  loop ((n, t) in rows) {
    val due = t.due()
    val note = if (due != null && !t.done) "  <- ${describeDue(due, now)}" else ""
    println("${"$n".padStart(width)} ${t.line()}$note")
  }
  println("--")
  println("${rows.len()} of $total ${if (total == 1) "task" else "tasks"} shown")
}

fun counts(tasks: List<Task>, pick: fun(Task): List<string>) {
  val tally: MutableMap<string, i64> = [:]
  loop (t in tasks) {
    if (t.done) continue
    loop (tag in pick(t).distinct()) {
      tally.set(tag, tally.getOrDefault(tag, 0) + 1)
    }
  }
  loop ((tag, n) in tally.entries().sortedWith((a, b) => if (a.1 != b.1) b.1.compareTo(a.1) else a.0.compareTo(b.0))) {
    println("${"$n".padStart(3)} $tag")
  }
}

// ---------------------------------------------------------------------------
// commands

const HELP = "usage: todo <command> [arguments]

  add TEXT...          add a task; (A) at the start sets its priority
  ls [TERMS...] [-a]   open tasks (all with -a) matching every term:
                       +project, @context, -word to exclude, a word to search
  due                  open tasks with a due:YYYY-MM-DD tag, soonest first
  done N...            complete tasks
  undo N               reopen a task
  pri N A-Z            set a priority; depri N removes it
  edit N TEXT...       replace a task's text
  rm N                 delete a task (later tasks are renumbered)
  projects, contexts   how many open tasks carry each tag
  archive              move completed tasks to done.txt next to the file

The file is \$TODO_FILE or ./todo.txt; \$TODO_TODAY overrides today's date.
"

fun run(args: List<string>) throws UsageError | IoError {
  val file = os.env("TODO_FILE") ?: "todo.txt"
  val todo = try TodoFile.open(file)
  val command = args.first() ?: "ls"
  val rest = args.drop(1)
  val width = "${todo.tasks.len() + 1}".len()

  when (command) {
    "help", "-h", "--help" => print(HELP)
    "add"                  => {
      if (rest.isEmpty()) throw UsageError(message: "add needs the task text")
      var t = Task.parse(rest.join(" "))
      t.createdOn = today()
      todo.tasks.push(t)
      try todo.save()
      println("${todo.tasks.len()} ${t.line()}")
    }
    "ls", "list"           => {
      val terms = rest.filter(a => a != "-a" && a != "--all")
      printRows(todo.listed(terms, rest.len() != terms.len()), todo.tasks.len(), width)
    }
    "due"                  => {
      val rows = todo.listed([], false).filter(r => r.1.due() != null)
      printRows(rows.sortedBy(r => r.1.due() ?: ""), todo.tasks.len(), width)
    }
    "done", "do"           => {
      if (rest.isEmpty()) throw UsageError(message: "done needs a task number")
      loop (arg in rest) {
        val n = try todo.number(arg)
        val t = todo.task(n)
        if (t.done) {
          println("$n is already done")
          continue
        }
        t.done = true
        t.completedOn = today()
        t.priority = ""
        println("$n ${t.line()}")
      }
      try todo.save()
    }
    "undo"                 => {
      val n = try todo.number(rest.first() ?: "")
      val t = todo.task(n)
      t.done = false
      t.completedOn = ""
      try todo.save()
      println("$n ${t.line()}")
    }
    "pri"                  => {
      val n = try todo.number(rest.first() ?: "")
      val p = (rest.at(1) ?: "").toUpper()
      if (!isPriority("($p)")) throw UsageError(message: "priority must be a letter A-Z, got '$p'")
      todo.task(n).priority = p
      try todo.save()
      println("$n ${todo.task(n).line()}")
    }
    "depri"                => {
      val n = try todo.number(rest.first() ?: "")
      todo.task(n).priority = ""
      try todo.save()
      println("$n ${todo.task(n).line()}")
    }
    "edit"                 => {
      val n = try todo.number(rest.first() ?: "")
      if (rest.len() < 2) throw UsageError(message: "edit needs the new text")
      todo.task(n).text = rest.drop(1).join(" ")
      try todo.save()
      println("$n ${todo.task(n).line()}")
    }
    "rm", "del"            => {
      val n = try todo.number(rest.first() ?: "")
      val t = todo.tasks.removeAt(n - 1)
      try todo.save()
      println("removed $n ${t.line()}")
    }
    "projects"             => counts(todo.tasks.toList(), t => t.projects())
    "contexts"             => counts(todo.tasks.toList(), t => t.contexts())
    "archive"              => {
      val done = todo.tasks.filter(t => t.done)
      if (done.isEmpty()) {
        println("nothing to archive")
        return
      }
      val archive = path.join(path.dir(file), "done.txt")
      try fs.appendFile(archive, done.map(t => t.line() + "\n").join(""))
      val open = todo.tasks.filter(t => !t.done)
      todo.tasks.clear()
      todo.tasks.addAll(open)
      try todo.save()
      println("archived ${done.len()} to ${path.base(archive)}, ${open.len()} left")
    }
    else                   => throw UsageError(message: "unknown command '$command' (try: todo help)")
  }
}

fun main() {
  when (val r = run(os.args())) {
    is Err => {
      println("todo: ${r.message()}")
      os.exit(if (r is UsageError) 2 else 1)
    }
    is Ok  => { }
  }
}
