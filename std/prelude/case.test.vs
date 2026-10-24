// D205, D206: enums as text in a `Case`, and a predicate per member.

enum Os {
  Windows
  Linux
  MacOS
}

enum Proto {
  HTTPServer
  TcpIp
  Ipv6
}

enum Arch {
  X64
  Arm64
}

implement Enumerated for Arch {
  static fun parse(text: string, case: Case): Arch? => when (text) {
    "x86_64", "amd64" => Arch.X64
    "aarch64"         => Arch.Arm64
    else              => null
  }
  fun toString(case: Case): string => if (this == Arch.X64) "x86_64" else "aarch64"
}

test fun spellings<E: Enumerated>(case: Case): List<string> => E.values().map(v => v.toString(case))

test "words split at case changes, runs of capitals and separators" {
  expect(caseWords("HTTPServer") == ["http", "server"])
  expect(caseWords("passwordHash") == ["password", "hash"])
  expect(caseWords("mac-os_X") == ["mac", "os", "x"])
  expect(caseWords("Ipv6") == ["ipv6"])
}

test "a member written in every case" {
  expect(Os.MacOS.toString() == "MacOS")
  expect(Os.MacOS.toString(case: Case.AsWritten) == "MacOS")
  expect(Os.MacOS.toString(case: Case.Lower) == "macos")
  expect(Os.MacOS.toString(case: Case.Upper) == "MACOS")
  expect(Os.MacOS.toString(case: Case.Camel) == "macOs")
  expect(Os.MacOS.toString(case: Case.Pascal) == "MacOs")
  expect(Os.MacOS.toString(case: Case.Snake) == "mac_os")
  expect(Os.MacOS.toString(case: Case.Kebab) == "mac-os")
  expect(Os.MacOS.toString(case: Case.ScreamingSnake) == "MAC_OS")
  expect(Proto.HTTPServer.toString(case: Case.Kebab) == "http-server")
}

test "parse is exact by default, forgiving with Case.Any, one form with another case" {
  expect(Os.parse("MacOS") == Os.MacOS)
  expect(Os.parse("macos") == null)
  expect(Os.parse("macos", case: Case.Any) == Os.MacOS)
  expect(Os.parse("MAC_OS", case: Case.Any) == Os.MacOS)
  expect(Os.parse("mac-os", case: Case.Kebab) == Os.MacOS)
  expect(Os.parse("mac_os", case: Case.Kebab) == null)
  expect(Proto.parse("http_server", case: Case.Snake) == Proto.HTTPServer)
  expect(Os.parse("solaris", case: Case.Any) == null)
}

test "an enum's own Enumerated is used for parse and toString(case:)" {
  expect(Arch.parse("x86_64") == Arch.X64)
  expect(Arch.parse("aarch64", case: Case.Any) == Arch.Arm64)
  expect(Arch.Arm64.toString(case: Case.Kebab) == "aarch64")
  expect(Arch.Arm64.toString() == "Arm64")  // Display is unchanged
}

test "generic code reads any enum through Enumerated" {
  expect(spellings<Os>(Case.Kebab) == ["windows", "linux", "mac-os"])
  expect(spellings<Arch>(Case.Kebab) == ["x86_64", "aarch64"])
}

test "a predicate per member" {
  val os = Os.Linux
  expect(os.isLinux())
  expect(!os.isWindows())
  expect(!os.isMacOS())
}
