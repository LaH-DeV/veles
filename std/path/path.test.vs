use path

test "within rejects absolute and relative root mismatches" {
  expect(!path.within("C:", "C:/outside.txt"))
  expect(!path.within("relative", "/relative/file"))
}

test "within accepts children with matching roots" {
  expect(path.within("C:/root", "C:/root/file.txt"))
  expect(path.within("relative/root", "relative/root/file.txt"))
}

test "dir preserves drive and UNC roots" {
  expect(path.dir("C:/file.txt") == "C:/")
  expect(path.dir("C:/dir/file.txt") == "C:/dir")
  expect(path.dir("//server/share/file.txt") == "//server/share/")
  expect(path.dir("//server/share") == "//server/share")
  expect(path.dir("//server/share/dir/file.txt") == "//server/share/dir")
}

test "within treats UNC share roots with and without separators as the same path" {
  val root = "//server/share"
  val forms = [root, root + "/", root.replace("/", "\\"), (root + "/").replace("/", "\\")]
  loop (base in forms) {
    loop (candidate in forms) expect(path.within(base, candidate))
    expect(path.within(base, root + "/file.txt"))
  }
}

test "UNC root equivalence does not admit sibling servers shares or relative paths" {
  loop (root in ["//server/share", "//server/share/", "\\\\server\\share\\"]) {
    expect(!path.within(root, "//server/share2/file.txt"))
    expect(!path.within(root, "//server/other/file.txt"))
    expect(!path.within(root, "//server2/share/file.txt"))
    expect(!path.within(root, "server/share"))
    expect(!path.within(root, "C:/file.txt"))
  }
}

test "clean fast paths preserve canonical text and normalization fallback" {
  val inputs = [
    "/a/b", "a/b", ".hidden/a..", "/a/./b", "/a//b/", "/a/../b",
    "../a", "a/../../b", "a\\b", "C:\\a\\..\\b", "//server/share/dir/../x",
    "\u{E9}/../\u{65E5}\u{672C}", ".", "..", "/", "//server/share/", "C:/",
  ]
  val expected = [
    "/a/b", "a/b", ".hidden/a..", "/a/b", "/a/b", "/b",
    "../a", "../b", "a/b", "C:/b", "//server/share/x",
    "\u{65E5}\u{672C}", ".", "..", "/", "//server/share/", "C:/",
  ]
  loop ((input, output) in inputs.zip(expected)) {
    expect(path.clean(input) == output)
    expect(path.clean(path.clean(input)) == output)
  }
}
