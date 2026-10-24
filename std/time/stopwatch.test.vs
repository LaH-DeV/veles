// Tests of `measure`: the result comes back with the time, a
// suspending body works, and an error the body throws passes through.

error Broke {
  message: string
}

test fun fails(): i64 throws Broke {
  throw Broke(message: "no")
}

test "measure returns the result and the time" {
  val (value, took) = measure(() => {
    await sleep(Duration.millis(10))
    42
  })
  expect(value == 42)
  expect(took.toMillis() >= 9)
  expect(took.toMillis() < 1000)
}

test "measure passes an error through" {
  val r = measure(() => try fails())
  expect(r is Err)
}
