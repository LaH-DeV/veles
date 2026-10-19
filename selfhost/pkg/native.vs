use settings { Schema, Section, Setting, SettingsError }
use text { quote }

/// `[native]` (D67): the C libraries the package's `extern` blocks need at link
/// time.
///
///     [native]
///     libs = ["pq"]                 # -lpq: the shared library (or its import library)
///     static-libs = ["z"]           # linked into the binary: no DLL/.so at run time
///     lib-paths = ["vendor/lib"]    # searched first; relative to veles.toml
///     pkg-config = ["libpq"]        # flags from `pkg-config --libs`
///
/// A `libs`/`static-libs` entry that names a file (it has a directory or an
/// archive extension) is linked as that file, relative to veles.toml.
struct NativeLibraries {
  var libs:       List<string> = []
  var staticLibs: List<string> = []
  var libPaths:   List<string> = []
  var pkgConfig:  List<string> = []
}

fun nativeSchema(): Schema<NativeLibraries> => Schema(
  table: "native",
  settings: [
    Setting<NativeLibraries>.names("libs", "shared libraries to link, by name", "libs = [\"z\"]", (_, _, names, native) => {
      native.libs = names
    }),
    Setting<NativeLibraries>.names("static-libs", "libraries linked into the binary", "static-libs = [\"z\"]", (_, _, names, native) => {
      native.staticLibs = names
    }),
    Setting<NativeLibraries>.names("lib-paths", "where to look first, relative to veles.toml", "lib-paths = [\"z\"]", (_, _, names, native) => {
      native.libPaths = names
    }),
    Setting<NativeLibraries>.names("pkg-config", "packages whose flags `pkg-config --libs` gives", "pkg-config = [\"z\"]", (section, value, names, native) => {
      // a name is passed to pkg-config as an argument: one that starts with '-' would be an option
      loop (name in names) {
        if (name.startsWith("-")) throw section.failAt(value, "[native] pkg-config: ${quote(name)} is a package name, not an option")
      }
      native.pkgConfig = names
    }),
  ],
)

fun readNative(section: Section): NativeLibraries throws SettingsError {
  val native = &NativeLibraries()
  try nativeSchema().apply(section, native)
  *native
}
