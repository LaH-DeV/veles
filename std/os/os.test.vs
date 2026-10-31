test "output success depends only on the exit code" {
  expect(Output(code: 0, stdout: "", stderr: "warning").ok())
  expect(!Output(code: 1, stdout: "result").ok())
  expect(!Output(code: -1, stdout: "").ok())
  expect(Output(code: 0, stdout: "result").stderr.isEmpty())
}

test "stderr modes retain their runtime protocol values" {
  expect(Stderr.Capture.value == 0)
  expect(Stderr.Inherit.value == 1)
  expect(Stderr.Merge.value == 2)
}

test "shutdown signals retain their portable signal numbers" {
  expect(Signal.Interrupt.value == 2)
  expect(Signal.Terminate.value == 15)
  expect(Signal.fromValue(2) == Signal.Interrupt)
  expect(Signal.fromValue(15) == Signal.Terminate)
  expect(Signal.fromValue(0) == null)
}

test "environment names containing NUL are rejected rather than truncated" {
  loop (name in ["\u{0}", "PATH\u{0}", "PATH\u{0}ignored", "HOME\u{0}ignored"]) {
    expect(env(name) == null)
  }
}

test "empty and NUL containing program names fail before process creation" {
  loop (program in ["", "\u{0}", "ignored\u{0}program"]) {
    expect(when (run(program)) {
      is Err(error) => error.kind == IoKind.InvalidInput && error.code == 22 &&
        error.path == program && error.detail == "not a program name"
      is Ok(_)      => false
    })
  }
}

test "NUL containing arguments are not silently split into new arguments" {
  loop (argument in ["\u{0}", "safe\u{0}extra", "\u{E9}\u{0}extra"]) {
    expect(when (run("never-start-this-program", ["first", argument])) {
      is Err(error) => error.kind == IoKind.InvalidInput && error.code == 22 &&
        error.path == "never-start-this-program" && error.detail == "an argument holds a NUL byte"
      is Ok(_)      => false
    })
  }
}

test "process identity and temporary directory queries return usable values" {
  expect(pid() > 0)
  expect(pid() == pid())
  expect(!program().isEmpty())
  expect(!tempDir().isEmpty())
}

test "platform errors preserve the supplied path and native code" {
  val error = ioError(0, "\u{E9}/missing.txt")
  expect(error.path == "\u{E9}/missing.txt")
  expect(error.code == 0)
  expect(!error.detail.isEmpty())
}
