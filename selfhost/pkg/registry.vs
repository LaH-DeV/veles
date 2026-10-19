use settings { Schema, Section, Setting, SettingsError }
/// `[registry]` (D139): where registry packages come from, and the key their
/// metadata is signed with. VELES_PROXY overrides the address. Without a key
/// the metadata is still checked against the archive and veles.sum, but the
/// registry itself is trusted on first use.
struct RegistrySettings {
  var url: string = ""
  var key: string = ""
}

fun registrySchema(): Schema<RegistrySettings> => Schema(
  table: "registry",
  settings: [
    Setting<RegistrySettings>.text("url", "the registry's http:// or https:// address", expects: "is a string", set: (section, value, text, registry) => {
      if (!text.startsWith("http://") && !text.startsWith("https://")) {
        throw section.failAt(value, "[registry] url is an http:// or https:// address")
      }
      registry.url = text
    }),
    Setting<RegistrySettings>.text("key", "the public key its metadata is signed with: ed25519:<base64>", expects: "is a string", set: (section, value, text, registry) => {
      if (!text.startsWith("ed25519:")) throw section.failAt(value, "[registry] key is a public key written ed25519:<base64>")
      registry.key = text
    }),
  ],
)

fun readRegistry(section: Section): RegistrySettings throws SettingsError {
  val registry = &RegistrySettings()
  try registrySchema().apply(section, registry)
  *registry
}
