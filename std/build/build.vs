/// What the compiler knows about the build, as constants (D160, D204): the
/// target it compiles for and the program it compiles. Every name here is
/// known while compiling, so an `if` on one compiles only the branch taken
/// and `static assert` can test it.
///
/// ```veles
/// use build
///
/// val separator = if (build.target.os.isWindows()) "\\" else "/"
/// io.println("${build.program.name} ${build.program.version}")
/// ```

/// An operating system a program can be built for.
public enum Os {
  Windows
  Linux
  MacOS
}

/// A processor architecture a program can be built for.
public enum Arch {
  X64
  Arm64
}

/// The target of a build: where the program will run. A cross-compile
/// describes the target, never the machine that compiles.
public struct Target {
  /// The operating system.
  public os: Os
  /// The processor architecture.
  public arch: Arch
  /// Whether this is a release build (`--release`).
  public release: bool
}

/// The program being built: the package whose `veles.toml` the build
/// started from — also when a library reads it.
public struct Program {
  /// The package's name.
  public name: string
  /// Its version, `""` when the manifest gives none.
  public version: string
}

/// The target this build compiles for.
public const target: Target = Target(
  os: Os.parse(compilerTarget("os"), case: Case.Any) ?: Os.Linux,
  arch: Arch.parse(compilerTarget("arch"), case: Case.Any) ?: Arch.X64,
  release: compilerTarget("release") == "true",
)  // compilerTarget and compilerProgram are answered by the compiler while compiling

/// The program this build compiles.
public const program: Program = Program(name: compilerProgram("name"), version: compilerProgram("version"))
