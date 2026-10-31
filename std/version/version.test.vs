test fun v(text: string): Version => Version.parse(text) catch (e) {
  panic("bad test version $text: ${e.message}")
}

/// The reason `text` is refused, or "ok".
test fun whyVersion(text: string): string {
  Version.parse(text) catch (e) {
    return e.message
  }
  "ok"
}

test parsing "versions are read and written back" {
  expect(v("1.4.2").toString() == "1.4.2")
  expect(v("v1.4.2").toString() == "1.4.2")
  expect(v("0.0.0").toString() == "0.0.0")
  expect(v("1.4.2-rc.1").toString() == "1.4.2-rc.1")
  expect(v("1.0.0-alpha-1.x-y").pre == ["alpha-1", "x-y"])
  expect(v("4294967295.0.0").major == 4294967295)
  expect(v("1.4.2").isPreRelease() == false && v("1.4.2-a").isPreRelease())
}

test refusals "what is not a version says why" {
  expect(whyVersion("1.2") == "\"1.2\" is not a version: write major.minor.patch, e.g. 1.4.2")
  expect(whyVersion("1.2.3.4").startsWith("\"1.2.3.4\" is not a version"))
  expect(whyVersion("1.02.3") == "\"1.02.3\" is not a version: \"02\" is not a number without leading zeros")
  expect(whyVersion("1.x.3").contains("\"x\" is not a number"))
  expect(whyVersion("1.2.-3").startsWith("\"1.2.-3\""))
  expect(whyVersion("4294967296.0.0").contains("is not a number"))
  expect(whyVersion("1.2.3+build") == "\"1.2.3+build\": build metadata ('+…') is not used in versions")
  expect(whyVersion("1.2.3-") == "\"1.2.3-\": the pre-release part after '-' is letters, digits, '.' and '-'")
  expect(whyVersion("1.2.3-a_b").contains("the pre-release part after '-'"))
  expect(whyVersion("1.2.3-a..b").contains("none empty"))
  expect(whyVersion("1.2.3-a.01").contains("without a leading zero"))
  expect(whyVersion("1.2.3-a.0") == "ok")
  expect(whyVersion("").startsWith("\"\" is not a version"))
  expect(whyVersion("\u{200B}").startsWith("\"\\u200b\" is not a version"))
}

test precedence "the order semver.org gives" {
  val ordered = ["1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.1.0", "2.0.0", "10.0.0"]
  loop (i in 0..<ordered.len() - 1) {
    val low = v(ordered.at(i) ?: "")
    val high = v(ordered.at(i + 1) ?: "")
    expect(low < high)
    expect(high > low)
    expect(low.compareTo(high) == Ordering.Less)
    expect(low != high)
  }
  expect(v("1.2.3").compareTo(v("1.2.3")) == Ordering.Equal)
  expect(v("1.9.0") < v("1.10.0"))
  expect(v("1.0.0-9") < v("1.0.0-10"))
  expect(v("1.0.0-99999999999999999999") < v("1.0.0-100000000000000000000"))
  expect(v("1.0.0-1") < v("1.0.0-a"))
  expect(v("1.0.0-Z") < v("1.0.0-a"))
}

test sorting "a list sorts by precedence" {
  val sorted = ["1.10.0", "1.2.0", "1.2.0-rc.1", "0.9.9"].map(text => v(text)).sorted().map(version => version.toString())
  expect(sorted == ["0.9.9", "1.2.0-rc.1", "1.2.0", "1.10.0"])
}

test ids "what makes two versions the same package" {
  expect(v("1.4.2").majorId() == "1")
  expect(v("0.3.1").majorId() == "0.3")
  expect(v("0.0.2").majorId() == "0.0.2")
  expect(v("2.0.0-rc.1").majorId() == "2")
}

test bumps "the next version" {
  expect(v("1.4.2").bumpMajor().toString() == "2.0.0")
  expect(v("1.4.2-rc.1").bumpMajor().toString() == "2.0.0")
  expect(v("1.4.2").bumpMinor().toString() == "1.5.0")
  expect(v("1.4.2").bumpPatch().toString() == "1.4.3")
  expect(v("1.4.2-rc.1").bumpPatch().toString() == "1.4.2")
}

test equality "versions are values" {
  expect(v("1.2.3") == v("v1.2.3"))
  expect(v("1.2.3-a") != v("1.2.3"))
  val seen: MutableSet<Version> = []
  seen.add(v("1.2.3"))
  expect(seen.contains(v("v1.2.3")))
}
