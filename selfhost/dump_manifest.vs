// The text the harness compares with the bootstrap compiler's: what the manifest
// reader makes of a `veles.toml`, field by field, or its error (plan R1).
use formatter
use linter
use pkg
use text { quote }

/// The dump of `text`, read as the manifest at `path`; `lints` are the
/// registry's lints.
fun manifestDump(path: string, text: string): string {
  val manifest = pkg.parseManifest(path, text) catch (e) {
    return "error ${e.message}\n"
  }
  val out = StringBuilder()
  val identity = manifest.identity
  out.appendLine("name ${quote(identity.name)}")
  out.appendLine("version ${quote(identity.version)}")
  out.appendLine("description ${quote(identity.description)}")
  out.appendLine("license ${quote(identity.license)}")
  out.appendLine("veles ${quote(identity.minimumCompiler)}")
  out.appendLine("registry-name ${quote(identity.registry)}")
  val paths = manifest.paths()
  loop (name in paths.keys().sorted()) out.appendLine("dep ${quote(name)} ${quote(paths.get(name) ?: "")}")
  loop (dependency in manifest.requires) out.appendLine("require ${dependencyText(dependency)}")
  loop (dependency in manifest.devRequires) out.appendLine("dev-require ${dependencyText(dependency)}")
  if (val members = manifest.workspace) out.appendLine("workspace ${quoteAll(members)}") else out.appendLine("workspace none")
  out.appendLine(formatText(manifest.format))
  out.appendLine("lint implicit-return=${manifest.lint.implicitReturn.value} max-width=${manifest.lint.maxWidth ?: 0}")
  loop (name in manifest.lint.levels.keys().sorted()) out.appendLine("lint-level ${quote(name)} ${levelText(manifest.lint.levels.get(name))}")
  loop (name in manifest.lint.library.keys().sorted()) out.appendLine("lint-library ${quote(name)} ${levelText(manifest.lint.library.get(name))}")
  val native = manifest.native
  out.appendLine("native libs=${quoteAll(native.libs)} static-libs=${quoteAll(native.staticLibs)} lib-paths=${quoteAll(native.libPaths)} pkg-config=${quoteAll(native.pkgConfig)}")
  val policy = manifest.policy
  out.appendLine("policy deny=${quoteAll(policy.deny.map(capability => pkg.capabilityName(capability)))}")
  loop (name in policy.allow.keys().sorted()) out.appendLine("policy-allow ${quote(name)} ${quoteAll(policy.allow.get(name) ?: [])}")
  loop (name in policy.trust.keys().sorted()) out.appendLine("policy-trust ${quote(name)} ${quote(policy.trust.get(name) ?: "")}")
  loop (name in policy.require.keys().sorted()) out.appendLine("policy-require ${quote(name)} ${policy.require.get(name) ?: 0}")
  out.appendLine("registry url=${quote(manifest.registry.url)} key=${quote(manifest.registry.key)}")
  out.appendLine("runtime threads=${manifest.runtime.threads ?: 0} gc-min-heap=${manifest.runtime.gcMinHeap ?: 0} gc-goal=${manifest.runtime.gcGoal ?: 0}")
  out.toString()
}

/// The registry of lints: name, default level, documentation, one per line.
fun lintsDump(): string {
  val out = StringBuilder()
  loop (lint in linter.lints()) out.appendLine("${lint.name}	${linter.lintLevelName(lint.default)}	${lint.doc}")
  out.toString()
}

fun formatText(format: formatter.FormatOptions): string {
  val indent = when (format.indent) {
    is formatter.DefaultIndent       => ""
    is formatter.TabIndent           => "	"
    is formatter.SpacesIndent(width) => " ".repeat(width)
  }
  val lineEndings = when (format.lineEndings) {
    formatter.LineEndings.Unset => ""
    formatter.LineEndings.Lf    => "lf"
    formatter.LineEndings.Crlf  => "crlf"
    formatter.LineEndings.Auto  => "auto"
  }
  "format indent=${quote(indent)} max-blank-lines=${format.maxBlankLines ?: 0} merge-imports=${format.imports == formatter.ImportStyle.Merged} no-trailing-commas=${format.commas == formatter.CommaStyle.Never} line-endings=${quote(lineEndings)}"
}

fun levelText(level: linter.LintLevel?): string => when (level) {
  linter.LintLevel.Allow => "allow"
  linter.LintLevel.Deny  => "deny"
  else                   => "warn"
}

fun dependencyText(dependency: pkg.Dependency): string {
  val (path, registry, git, version, commit) = sourceFields(dependency.source)
  "${quote(dependency.name)} ${dependency.kind()} path=${quote(path)} registry=${quote(registry)} git=${quote(git)} version=${quote(version)} commit=${quote(commit)} line=${dependency.line}"
}

/// A source as the five keys a dependency table writes: path, registry, git,
/// version, commit.
fun sourceFields(source: pkg.Source): (string, string, string, string, string) => when (source) {
  is pkg.PathSource(path)              => (path, "", "", "", "")
  is pkg.RegistrySource(name, version) => ("", name, "", version, "")
  is pkg.GitSource(url, pin)           => when (pin) {
    is pkg.TagPin(version)   => ("", "", url, version, "")
    is pkg.CommitPin(prefix) => ("", "", url, "", prefix)
  }
}

fun quoteAll(items: List<string>): string => "[" + items.map(item => quote(item)).join(",") + "]"
