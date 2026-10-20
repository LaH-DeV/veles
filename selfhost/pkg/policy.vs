use settings { Schema, Section, Setting, SettingsError, stringList }
use text { quote }

/// What a package's source can do that a build should know about (D138),
/// computed from the source and never declared. `Unlisted` is not something the
/// source does: the package is not in the registry's listed tier (D139), so one
/// policy vocabulary says "deny unlisted" the way it says "deny net".
enum Capability {
  Unsafe
  Extern
  Native
  Net
  Fs
  Os
  Ffi
  Unlisted
}

/// The capability a policy file names (`"net"`), or null.
fun capabilityNamed(name: string): Capability? => if (name == name.toLower()) Capability.parse(name.capitalize()) else null

/// The lowercase name a manifest writes.
fun capabilityName(capability: Capability): string => capability.toString().toLower()

fun capabilityNames(): string => Capability.values().map(capability => capabilityName(capability)).join(", ")

/// `[policy]` (D138): which capabilities the packages this one depends on may
/// use.
///
///     [policy]
///     deny = ["net", "unsafe", "unlisted"]    # no dependency may use these…
///     allow = { net = ["acme/httputil"] }     # …except the packages named here
///     trust = { alice = "ed25519:…" }         # whose reviews count (D139)
///     require = { reviewed = 1 }              # how many trusted reviews a package needs
///
/// A package is named by its source — `owner/name` or the git URL as written in
/// the dependency — or by its own package name. The policy applies to registry
/// and git packages; a path dependency is the project's own code. `allow` takes
/// capabilities and the claims named in `require`.
struct Policy {
  var deny: List<Capability> = []
  allow:    MutableMap<string, List<string>> = [:]
  /// Reviewer name to `ed25519:<base64 public key>`.
  trust: MutableMap<string, string> = [:]
  /// Claim to the number of distinct trusted keys it needs.
  require: MutableMap<string, i64> = [:]
}

fun policySchema(): Schema<Policy> => Schema(
  table: "policy",
  settings: [
    Setting<Policy>.names("deny", "capabilities no dependency may use", "deny = [\"net\"]", (section, value, names, policy) => {
      val capabilities: MutableList<Capability> = []
      loop (name in names) {
        capabilities.push(capabilityNamed(name) ?: throw section.failAt(value, "[policy] deny: unknown capability ${quote(name)} (${capabilityNames()})"))
      }
      policy.deny = capabilities
    }),
    Setting<Policy>(key: "allow", doc: "the packages that may use a capability or claim despite deny", read: (section, value, policy) => {
      val table = value.asTable() ?: throw section.failAt(value, "[policy] allow is a table of capability = [packages], e.g. allow = { net = [\"acme/httputil\"] }")
      loop (claimant in section.within(table).entries()) {
        policy.allow.set(claimant.key, try stringList(section, claimant.value, "[policy] allow.${claimant.key}", "${claimant.key} = [\"owner/name\"]"))
      }
    }),
    Setting<Policy>(key: "trust", doc: "reviewers whose attestations count: name = \"ed25519:<base64 key>\"", read: (section, value, policy) => {
      val table = value.asTable() ?: throw section.failAt(value, "[policy] trust is a table of reviewer = key, e.g. trust = { alice = \"ed25519:…\" }")
      loop (reviewer in section.within(table).entries()) {
        val key = reviewer.value.asString()
        if (key == null || !key.startsWith("ed25519:")) {
          throw section.failAt(reviewer.value, "[policy] trust.${reviewer.key} is a public key written ed25519:<base64>")
        }
        policy.trust.set(reviewer.key, key)
      }
    }),
    Setting<Policy>(key: "require", doc: "claims and how many trusted reviewers must attest each", read: (section, value, policy) => {
      val table = value.asTable() ?: throw section.failAt(value, "[policy] require is a table of claim = count, e.g. require = { reviewed = 1 }")
      loop (claim in section.within(table).entries()) {
        val count = claim.value.asInt()
        if (count == null || count < 1) {
          throw section.failAt(claim.value, "[policy] require.${claim.key} is how many trusted reviewers must attest it, 1 or more")
        }
        policy.require.set(claim.key, count)
      }
    }),
  ],
)

fun readPolicy(section: Section): Policy throws SettingsError {
  val policy = &Policy()
  try policySchema().apply(section, policy)
  if (!policy.require.isEmpty() && policy.trust.isEmpty()) {
    throw section.fail(section.table.line, "[policy] require needs trust = { name = \"ed25519:…\" }: whose reviews count?")
  }
  if (val allowed = section.table.get("allow")) {
    // in a fixed order, so the same manifest always gives the same message
    loop (name in policy.allow.keys().sorted()) {
      if (capabilityNamed(name) == null && !policy.require.containsKey(name)) {
        throw section.failAt(allowed, "[policy] allow: ${quote(name)} is neither a capability (${capabilityNames()}) nor a claim in require")
      }
    }
  }
  *policy
}
