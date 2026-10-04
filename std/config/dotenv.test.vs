// Tests of the dotenv reader (D125): what a line may be, quoting, comments,
// what is taken literally, and that a bad line is a problem the rest of the
// file survives.

test fun vars(text: string): Map<string, string> = parseDotenv(text).vars

test fun problems(text: string): List<string> = parseDotenv(text).problems

test "names, values, comments and blank lines" {
  val got = vars("# settings\n\nPORT=8080\n  NAME = my app  \nexport MODE=fast\nEMPTY=\n")
  expect(got.get("PORT") == "8080")
  expect(got.get("NAME") == "my app")
  expect(got.get("MODE") == "fast")
  expect(got.get("EMPTY") == "")
  expect(got.len() == 4)
}

test "an unquoted value ends at a space and a hash, and keeps a bare hash" {
  val got = vars("A=one # a note\nB=one#two\nC= # nothing\nD=#tag\nE=one\t# tab\n")
  expect(got.get("A") == "one")
  expect(got.get("B") == "one#two")
  expect(got.get("C") == "")
  expect(got.get("D") == "#tag")
  expect(got.get("E") == "one")
}

test "single quotes are literal, double quotes read escapes" {
  val got = vars("A='a \\n \$HOME #not a comment'\nB=\"line\\none\\ttab \\\"q\\\" back\\\\slash\"\nC=\"keep # this\" # drop this\nD=\"unknown \\x stays\"\n")
  expect(got.get("A") == "a \\n \$HOME #not a comment")
  expect(got.get("B") == "line\none\ttab \"q\" back\\slash")
  expect(got.get("C") == "keep # this")
  expect(got.get("D") == "unknown \\x stays")
}

test "a dollar sign is just a dollar sign" {
  val got = vars("PASSWORD=pa\$\$w0rd\${HOME}\nURL=\"postgres://u:p\$ss@h/db\"\n")
  expect(got.get("PASSWORD") == "pa\$\$w0rd\${HOME}")
  expect(got.get("URL") == "postgres://u:p\$ss@h/db")
}

test "a quoted value may span lines" {
  val got = vars("KEY=\"-----BEGIN-----\nabc\n-----END-----\"\nAFTER=1\n")
  expect(got.get("KEY") == "-----BEGIN-----\nabc\n-----END-----")
  expect(got.get("AFTER") == "1")
}

test "windows line endings are read like unix ones" {
  val got = vars("A=1\r\nB=\"two\"\r\n# note\r\nC=3 # c\r\n")
  expect(got.get("A") == "1")
  expect(got.get("B") == "two")
  expect(got.get("C") == "3")
}

test "a later line for the same name wins" {
  expect(vars("A=1\nA=2\n").get("A") == "2")
}

test "a bad line is a problem and the others are read" {
  val found = parseDotenv("GOOD=1\nnot a variable\n1BAD=2\nALSO=\"open\nSTILL=3\n")
  expect(found.vars.get("GOOD") == "1")
  expect(found.problems.len() >= 3)
  expect(found.problems.at(0) == "line 2: expected NAME=value")
  expect(found.problems.at(1) == "line 3: '1BAD' is not a variable name")
  expect(found.problems.at(2) == "line 4: the quote opened here is never closed")
}

test "text after a closing quote is a problem" {
  val found = parseDotenv("A=\"x\" y\nB=2\n")
  expect(found.problems == ["line 1: text after the closing quote"])
  expect(found.vars.get("B") == "2")
  expect(found.vars.get("A") == null)
}

test "an empty file and a file of comments hold nothing" {
  expect(vars("").isEmpty())
  expect(vars("# a\n# b\n\n").isEmpty())
  expect(problems("# a\n").isEmpty())
}
