test fun c(text: string): Constraint => Constraint.parse(text) catch (e) {
  panic("bad test constraint $text: ${e.message}")
}

/// The versions among `candidates` that `constraint` accepts, as text.
test fun accepted(constraint: string, candidates: List<string>): List<string> {
  val wanted = c(constraint)
  candidates.filter(text => wanted.matches(v(text)))
}

test fun whyConstraint(text: string): string {
  Constraint.parse(text) catch (e) {
    return e.message
  }
  "ok"
}

const sample: List<string> = ["0.0.3", "0.0.4", "0.2.3", "0.2.9", "0.3.0", "1.0.0", "1.2.2", "1.2.3", "1.2.9", "1.3.0", "1.9.0", "2.0.0", "2.0.0-rc.1", "3.1.0"]

test exact "a bare full version, or =, is exactly that version" {
  expect(accepted("1.2.3", sample) == ["1.2.3"])
  expect(accepted("=1.2.3", sample) == ["1.2.3"])
  expect(accepted("v1.2.3", sample) == ["1.2.3"])
}

test bounds "comparison operators" {
  expect(accepted(">=1.9.0", sample) == ["1.9.0", "2.0.0", "3.1.0"])
  expect(accepted(">1.9.0", sample) == ["2.0.0", "3.1.0"])
  expect(accepted("<0.2.3", sample) == ["0.0.3", "0.0.4"])
  expect(accepted("<=0.2.3", sample) == ["0.0.3", "0.0.4", "0.2.3"])
  expect(accepted(">= 1.9.0", sample) == ["1.9.0", "2.0.0", "3.1.0"])
  expect(accepted(">=1.2.3 <1.3.0", sample) == ["1.2.3", "1.2.9"])
  expect(accepted(">=1.2.3, <1.3.0", sample) == ["1.2.3", "1.2.9"])
}

test caret "^ keeps the first non-zero number" {
  expect(accepted("^1.2.3", sample) == ["1.2.3", "1.2.9", "1.3.0", "1.9.0"])
  expect(accepted("^1.2", sample) == ["1.2.2", "1.2.3", "1.2.9", "1.3.0", "1.9.0"])
  expect(accepted("^1", sample) == ["1.0.0", "1.2.2", "1.2.3", "1.2.9", "1.3.0", "1.9.0"])
  expect(accepted("^0.2.3", sample) == ["0.2.3", "0.2.9"])
  expect(accepted("^0.2", sample) == ["0.2.3", "0.2.9"])
  expect(accepted("^0.0.3", sample) == ["0.0.3"])
  expect(accepted("^0.0", sample) == ["0.0.3", "0.0.4"])
  expect(accepted("^0", sample) == ["0.0.3", "0.0.4", "0.2.3", "0.2.9", "0.3.0"])
}

test tilde "~ allows patch updates" {
  expect(accepted("~1.2.3", sample) == ["1.2.3", "1.2.9"])
  expect(accepted("~1.2", sample) == ["1.2.2", "1.2.3", "1.2.9"])
  expect(accepted("~1", sample) == ["1.0.0", "1.2.2", "1.2.3", "1.2.9", "1.3.0", "1.9.0"])
  expect(accepted("~0.2.3", sample) == ["0.2.3", "0.2.9"])
}

test ranges "partial versions and wildcards" {
  expect(accepted("1.2", sample) == ["1.2.2", "1.2.3", "1.2.9"])
  expect(accepted("1.2.x", sample) == ["1.2.2", "1.2.3", "1.2.9"])
  expect(accepted("1.*", sample) == ["1.0.0", "1.2.2", "1.2.3", "1.2.9", "1.3.0", "1.9.0"])
  expect(accepted("*", sample).len() == 13)
  expect(accepted("x", sample).len() == 13)
  expect(accepted(">1.2", sample) == ["1.3.0", "1.9.0", "2.0.0", "3.1.0"])
  expect(accepted("<=1.2", sample) == ["0.0.3", "0.0.4", "0.2.3", "0.2.9", "0.3.0", "1.0.0", "1.2.2", "1.2.3", "1.2.9"])
  expect(accepted(">=1.2", sample) == ["1.2.2", "1.2.3", "1.2.9", "1.3.0", "1.9.0", "2.0.0", "3.1.0"])
  expect(accepted("<1.2", sample) == ["0.0.3", "0.0.4", "0.2.3", "0.2.9", "0.3.0", "1.0.0"])
  expect(accepted(">1", sample) == ["2.0.0", "3.1.0"])
  expect(accepted("<=1", sample) == ["0.0.3", "0.0.4", "0.2.3", "0.2.9", "0.3.0", "1.0.0", "1.2.2", "1.2.3", "1.2.9", "1.3.0", "1.9.0"])
}

test alternatives "|| joins sets" {
  expect(accepted("1.0.0 || 3.1.0", sample) == ["1.0.0", "3.1.0"])
  expect(accepted("^0.2 || >=3", sample) == ["0.2.3", "0.2.9", "3.1.0"])
  expect(accepted("<0.0.4||>=2.0.0", sample) == ["0.0.3", "2.0.0", "3.1.0"])
}

test prereleases "a pre-release is offered only to a constraint that names one of its release" {
  expect(accepted("^1.2.0", ["1.3.0-beta.1", "1.3.0"]) == ["1.3.0"])
  expect(accepted(">=2.0.0-rc.1", ["2.0.0-rc.1", "2.0.0-rc.2", "2.0.0", "2.0.1-rc.1"]) == ["2.0.0-rc.1", "2.0.0-rc.2", "2.0.0"])
  expect(accepted("<2.0.0", ["1.9.9", "2.0.0-rc.1"]) == ["1.9.9"])
  expect(accepted("*", ["1.0.0-alpha"]) == [])
  expect(accepted("=1.0.0-alpha", ["1.0.0-alpha", "1.0.0"]) == ["1.0.0-alpha"])
}

test newest "the highest version that qualifies" {
  val all = sample.map(text => v(text))
  expect(c("^1.2").newest(all)?.toString() == "1.9.0")
  expect(c("~0.2").newest(all)?.toString() == "0.2.9")
  expect(c("<1.0.0").newest(all)?.toString() == "0.3.0")
  expect(c(">=9").newest(all) == null)
  expect(c("*").newest([]) == null)
}

test exactly "a constraint made from a version" {
  expect(Constraint.exactly(v("1.2.3")).matches(v("1.2.3")))
  expect(!Constraint.exactly(v("1.2.3")).matches(v("1.2.4")))
  expect(Constraint.exactly(v("1.2.3-rc.1")).matches(v("1.2.3-rc.1")))
  expect(Constraint.exactly(v("1.2.3")).toString() == "=1.2.3")
}

test text "a constraint keeps the text it was read from" {
  expect(c("^1.2 || >=3").toString() == "^1.2 || >=3")
  expect(c("^1.2 || >=3").text == "^1.2 || >=3")
}

test refusals "what is not a constraint says why" {
  expect(whyConstraint("") == "\"\": a constraint is a version, a bound like '>=1.2', or '*' for any version")
  expect(whyConstraint("1.2 ||").contains("a constraint is a version"))
  expect(whyConstraint(">=") == "\">=\": '>=' needs a version after it")
  expect(whyConstraint("^").contains("'^' needs a version after it"))
  expect(whyConstraint("1.2.3.4").contains("has more than three numbers"))
  expect(whyConstraint("1.x.3").contains("has a number after an 'x'"))
  expect(whyConstraint("a.b.c").contains("is not a number"))
  expect(whyConstraint("1.02").contains("is not a number without leading zeros"))
  expect(whyConstraint(">*").contains("matches no version"))
  expect(whyConstraint("<*").contains("matches no version"))
  expect(whyConstraint("1.2-rc.1").contains("a pre-release goes with all three numbers"))
  expect(whyConstraint("1.x-rc.1").contains("pre-release"))
  expect(whyConstraint("1.2.3-").contains("the pre-release part"))
  expect(whyConstraint("^1.2.3-rc.1") == "ok")
}
