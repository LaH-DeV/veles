test "toFixed preserves precision beyond the C formatting limit" {
  val thirtyOne = 1.5.toFixed(31)
  expect(thirtyOne.len() == 33)
  expect(thirtyOne.startsWith("1.5"))
  expect(thirtyOne.endsWith("0"))

  val beyondExact = 1.5.toFixed(1075)
  expect(beyondExact.len() == 1077)
  expect(beyondExact.startsWith("1.5"))
  expect(beyondExact.endsWith("0"))

  val single = 1.5.toF32().toFixed(31)
  expect(single.len() == 33)
  expect(single.startsWith("1.5"))
  expect(single.endsWith("0"))
}

test "toFixed refuses negative and unrepresentable precision" {
  expectPanics(() => {
    1.5.toFixed(-1)
  })
  expectPanics(() => {
    1.5.toF32().toFixed(-1)
  })
  expectPanics(() => {
    1.5.toFixed(9223372036854775807)
  })
}

test "isClose is math.isclose: relative to the larger magnitude, or absolute" {
  expect((0.1 + 0.2).isClose(0.3))
  expect(!1.0.isClose(1.0 + 1.0e-8))
  expect(1.0.isClose(1.0 + 1.0e-10))
  expect(100.0.isClose(110.0, relative: 0.1))
  expect(!100.0.isClose(112.0, relative: 0.1))
  // near zero a fraction of the magnitude is nothing: only absolute helps
  expect(!0.0.isClose(1.0e-20))
  expect(0.0.isClose(1.0e-20, absolute: 1.0e-12))
  expect((-2.0).isClose(-2.0 - 1.0e-12))
  val third: f32 = 1.0 / 3.0
  expect(third.isClose(0.33333334))
  expect(!third.isClose(0.3334))
}

test "isClose: an infinity is close only to itself, a NaN to nothing" {
  val inf = 1.0 / 0.0
  val nan = 0.0 / 0.0
  expect(inf.isClose(inf))
  expect(!inf.isClose(-inf))
  expect(!inf.isClose(1.0e308, relative: 1.0))
  expect(!nan.isClose(nan))
  expect(!nan.isClose(1.0, absolute: 1.0e308))
}

test "isClose refuses a negative tolerance" {
  expectPanics(() => {
    1.0.isClose(1.0, relative: -1.0)
  })
  expectPanics(() => {
    1.0.toF32().isClose(1.0, absolute: -1.0)
  })
}
