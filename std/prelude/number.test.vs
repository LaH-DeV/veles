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

test "toFixed of a short binary fraction is exact, as the long way gives it" {
  // values whose fraction has few binary digits take the runtime's integer
  // path when the digits cover them; the rest round the exact long way
  expect(5.0.toFixed(2) == "5.00")
  expect(12.25.toFixed(2) == "12.25")
  expect(12.25.toFixed(0) == "12")
  expect((-0.5).toFixed(3) == "-0.500")
  expect((-0.0).toFixed(1) == "-0.0")
  expect(0.0.toFixed(0) == "0")
  expect(1048576.0.toFixed(5) == "1048576.00000")
  expect(0.0000019073486328125.toFixed(19) == "0.0000019073486328125")
  expect(0.0000019073486328125.toFixed(22) == "0.0000019073486328125000")
  expect(12.75.toFixed(1) == "12.8")
}

test "toFixed and a precision spec round a tie to the even neighbour, on every platform (D240)" {
  expect(12.25.toFixed(1) == "12.2")
  expect(12.75.toFixed(1) == "12.8")
  expect(0.125.toFixed(2) == "0.12")
  expect(0.375.toFixed(2) == "0.38")
  expect(2.5.toFixed(0) == "2")
  expect(3.5.toFixed(0) == "4")
  expect((-2.5).toFixed(0) == "-2")
  expect((-0.5).toFixed(0) == "-0")
  expect(9.5.toFixed(0) == "10")
  expect((-99.5).toFixed(0) == "-100")
  expect(0.0000019073486328125.toFixed(18) == "0.000001907348632812")
  val share = 0.125
  expect("${share:.2}" == "0.12")
  expect(2.5.roundEven() == 2.0)
  expect(2.5.round() == 3.0)
  expect(0.1.toFixed(3) == "0.100")
  expect(2.675.toFixed(2) == "2.67")
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
