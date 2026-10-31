test "successful stdin status does not manufacture an error" {
  expect(checkStdinStatus(stdinOK, 123, 0, "input") is Ok)
}

test "invalid text becomes InvalidData with stdin context" {
  loop (what in ["line", "input"]) {
    expect(when (checkStdinStatus(stdinInvalidUTF8, 123, 64, what)) {
      is Err(error) => when (error) {
        is IoError => error.kind == IoKind.InvalidData && error.path == "stdin" &&
          error.code == 0 && error.detail == "standard input $what is not valid UTF-8"
        is TooLong => false
      }
      is Ok(_)      => false
    })
  }
}

test "overflow preserves the callers byte ceiling" {
  loop (limit in [0, 1, 65536, 1048576, 9223372036854775807]) {
    expect(when (checkStdinStatus(stdinTooLong, 0, limit, "line")) {
      is Err(error) => when (error) {
        is TooLong => error.limit == limit &&
          error.message == "standard input line exceeded its $limit-byte limit"
        is IoError => false
      }
      is Ok(_)      => false
    })
  }
}

test "native read failures retain their error code" {
  loop (code in [0, 5, 9]) {
    expect(when (checkStdinStatus(stdinIOError, code, 64, "input")) {
      is Err(error) => when (error) {
        is IoError => error.kind == IoKind.Other && error.path == "stdin" &&
          error.code == code && error.detail == "could not read standard input"
        is TooLong => false
      }
      is Ok(_)      => false
    })
  }
}

test "unexpected native statuses fail closed with diagnostic context" {
  loop (status in [-1, stdinEOF, 5, 9223372036854775807]) {
    expect(when (checkStdinStatus(status, 9, 64, "input")) {
      is Err(error) => when (error) {
        is IoError => error.kind == IoKind.Other && error.path == "stdin" &&
          error.code == 9 && error.detail == "unknown standard-input status $status"
        is TooLong => false
      }
      is Ok(_)      => false
    })
  }
}

test "text line rejects a negative ceiling before reading stdin" {
  expectPanics(() => readLine(max: -1))
}

test "byte line rejects a negative ceiling before reading stdin" {
  expectPanics(() => readLineBytes(max: -1))
}

test "text readAll rejects a negative ceiling before reading stdin" {
  expectPanics(() => readAll(max: -1))
}

test "byte readAll rejects a negative ceiling before reading stdin" {
  expectPanics(() => readAllBytes(max: -1))
}
