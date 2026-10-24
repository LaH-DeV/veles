use path as paths
use toml { Value }
use settings { Schema, Section, Setting, SettingsError }
use text { quote }

/// `[workspace]` (D138): the member directories of a monorepo, relative to the
/// manifest that lists them, in the cleaned form they are compared by.
struct WorkspaceMembers {
  var directories: List<string> = []
}

fun workspaceSchema(): Schema<WorkspaceMembers> => Schema(
  table: "workspace",
  settings: [
    Setting<WorkspaceMembers>.names("members", "the member directories, below the root", "members = [\"app\", \"libs/mathlib\"]", (section, value, listed, workspace) => {
      val seen: MutableSet<string> = []
      val cleaned: MutableList<string> = []
      loop (member in listed) {
        val clean = try memberDirectory(section, value, member)
        if (seen.contains(clean)) throw section.failAt(value, "[workspace] members: ${quote(member)} is listed twice")
        seen.add(clean)
        cleaned.push(clean)
      }
      workspace.directories = workspace.directories.concat(cleaned)
    }),
  ],
)

fun readWorkspace(section: Section): List<string> throws SettingsError {
  val workspace = &WorkspaceMembers()
  try workspaceSchema().apply(section, workspace)
  if (workspace.directories.isEmpty()) throw section.fail(section.table.line, "[workspace] needs members = [\"dir\", ...]")
  workspace.directories
}

/// A member as a directory below the root manifest, cleaned; a refusal where it
/// is not one.
fun memberDirectory(section: Section, at: Value, member: string): string throws SettingsError {
  val clean = paths.clean(member)
  if (member.isEmpty() || clean == ".") {
    throw section.failAt(at, "[workspace] members: a member is a directory below the root, not ${quote(member)}")
  }
  if (paths.isAbsolute(member) || clean == ".." || clean.startsWith("../")) {
    throw section.failAt(at, "[workspace] members: ${quote(member)} leaves the workspace; a member is a directory below the root manifest")
  }
  if (member.contains("*") || member.contains("?") || member.contains("[")) {
    throw section.failAt(at, "[workspace] members: patterns such as ${quote(member)} are not supported yet; list each member directory")
  }
  clean
}
