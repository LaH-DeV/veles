use toml { Table, Value }
use version { Version }
use settings { Section, SettingsError }
use text { quote }

/// Where a dependency comes from. Exactly one, by construction: a table that
/// names two sources never becomes a `Dependency`.
sealed trait Source

/// A directory, relative to the manifest: the project's own code.
struct PathSource : Source {
  path: string
}

/// A package of a registry (D139), pinned by an exact version as written.
struct RegistrySource : Source {
  name:    string
  version: string
}

/// A git repository, pinned by a tag (a version) or by a commit.
struct GitSource : Source {
  url: string
  pin: GitPin
}

sealed trait GitPin

struct TagPin : GitPin {
  version: string
}

struct CommitPin : GitPin {
  prefix: string
}

/// One entry of `[dependencies]` or `[dev-dependencies]`.
struct Dependency {
  /// The name code writes in `use`.
  name:   string
  source: Source
  line:   i64

  /// `path`, `registry` or `git`.
  fun kind(): string => when (this.source) {
    is PathSource     => "path"
    is RegistrySource => "registry"
    is GitSource      => "git"
  }
}

/// The keys of a dependency table, as written; `sourceOf` turns them into a
/// `Source` or says why they cannot be one.
struct DependencyKeys {
  var path:     string = ""
  var registry: string = ""
  var git:      string = ""
  var version:  string = ""
  var commit:   string = ""
}

/// Why a dependency is wrong: the reason half of a message.
error DependencyProblem {
  message: string
}

/// The source the keys name, or the reason they do not.
fun sourceOf(keys: DependencyKeys): Source throws DependencyProblem {
  val named = [keys.path, keys.registry, keys.git].count(source => !source.isEmpty())
  if (named != 1) throw DependencyProblem(message: "say where it comes from with exactly one of path, registry or git")
  if (!keys.version.isEmpty()) {
    Version.parse(keys.version) catch (e) {
      throw DependencyProblem(message: e.message)
    }
    if (keys.version.startsWith("v")) {
      throw DependencyProblem(message: "write the version without the 'v' (${quote(keys.version.substring(1, keys.version.len()) ?: "")})")
    }
  }
  if (!keys.commit.isEmpty() && !isCommitPrefix(keys.commit)) {
    throw DependencyProblem(message: "commit ${quote(keys.commit)} is a hex prefix of 7 to 40 digits")
  }
  if (!keys.path.isEmpty()) {
    if (!keys.version.isEmpty() || !keys.commit.isEmpty()) {
      throw DependencyProblem(message: "a path dependency has no version or commit: it is the directory as it is")
    }
    return PathSource(path: keys.path)
  }
  if (!keys.registry.isEmpty()) {
    if (!isRegistryName(keys.registry)) {
      throw DependencyProblem(message: "registry name ${quote(keys.registry)} is owner/name, lowercase letters, digits, '-' and '_'")
    }
    if (!keys.commit.isEmpty()) throw DependencyProblem(message: "a registry dependency is pinned by version; a commit pin is for git")
    if (keys.version.isEmpty()) throw DependencyProblem(message: "a registry dependency needs a version, e.g. version = \"1.0.0\"")
    return RegistrySource(name: keys.registry, version: keys.version)
  }
  if (keys.version.isEmpty() == keys.commit.isEmpty()) {
    throw DependencyProblem(message: "a git dependency is pinned by exactly one of version (a tag) or commit")
  }
  val pin: GitPin = if (keys.version.isEmpty()) CommitPin(prefix: keys.commit) else TagPin(version: keys.version)
  GitSource(url: keys.git, pin)
}

/// Reads a `[dependencies]` or `[dev-dependencies]` table.
fun readDependencies(section: Section): List<Dependency> throws SettingsError {
  val out: MutableList<Dependency> = []
  loop (entry in section.entries()) {
    val name = entry.key
    val value = entry.value
    if (!isLocalName(name)) {
      throw section.failAt(value, "dependency name ${quote(name)}: the name code writes in `use`, so letters, digits and '_' and not starting with a digit")
    }
    val source = try dependencySource(section, name, value)
    out.push(Dependency(name, source, line: value.line))
  }
  out
}

/// The source one entry names: `name = "../dir"`, or a table of keys.
fun dependencySource(section: Section, name: string, value: Value): Source throws SettingsError {
  if (val path = value.asString()) {
    // a bare string is the short form of `{ path = … }`
    if (looksLikeVersion(path)) {
      throw section.failAt(value, "dependency '$name' = ${quote(path)} looks like a version, but a bare string is a path: write { registry = \"owner/name\", version = ${quote(path)} }")
    }
    return try checked(section, name, value, DependencyKeys(path))
  }
  val table = value.asTable() ?: throw section.failAt(value, "dependency '$name' is a path string or a table such as { registry = \"owner/name\", version = \"1.0.0\" }")
  val keys = try dependencyKeys(section, name, table)
  try checked(section, name, value, keys)
}

fun checked(section: Section, name: string, value: Value, keys: DependencyKeys): Source throws SettingsError {
  sourceOf(keys) catch (problem) {
    throw section.failAt(value, "dependency '$name': ${problem.message}")
  }
}

fun dependencyKeys(section: Section, name: string, table: *Table): DependencyKeys throws SettingsError {
  var keys = DependencyKeys()
  loop (entry in section.within(table).entries()) {
    val text = when (entry.key) {
      "path", "registry", "git", "version", "commit" => entry.value.asString() ?: throw section.failAt(entry.value, "dependency '$name': ${entry.key} is a string")
      else => throw section.failAt(entry.value, "dependency '$name': unknown key ${quote(entry.key)} (path, registry, git, version, commit)")
    }
    when (entry.key) {
      "path"     => keys.path = text
      "registry" => keys.registry = text
      "git"      => keys.git = text
      "version"  => keys.version = text
      else       => keys.commit = text
    }
  }
  keys
}

/// Letters, digits and `_`, not starting with a digit: a name `use` can say.
fun isLocalName(text: string): bool {
  if (text.isEmpty() || text.byteAt(0).isAsciiDigit()) return false
  loop (i in 0..<text.len()) {
    val byte = text.byteAt(i)
    if (!byte.isAsciiAlphanumeric() && byte != '_') return false
  }
  true
}

/// `owner/name`, each lowercase letters, digits, `-` and `_`.
fun isRegistryName(text: string): bool {
  val (owner, name) = text.splitOnce("/") ?: return false
  loop (part in [owner, name]) {
    if (part.isEmpty()) return false
    loop (i in 0..<part.len()) {
      val byte = part.byteAt(i)
      val allowed = byte.isAsciiLower() || byte.isAsciiDigit() || byte == '-' || byte == '_'
      if (!allowed) return false
    }
  }
  true
}

/// A hex prefix of a commit, 7 to 40 digits.
fun isCommitPrefix(text: string): bool {
  if (text.len() < 7 || text.len() > 40) return false
  loop (i in 0..<text.len()) {
    val byte = text.byteAt(i)
    if (!byte.isAsciiHexDigit() || byte.isAsciiUpper()) return false
  }
  true
}

/// A bare "1.4.2" in `[dependencies]` is a mistake worth a pointed message,
/// since a path never looks like that.
fun looksLikeVersion(text: string): bool {
  Version.parse(text) catch (e) {
    return false
  }
  true
}
