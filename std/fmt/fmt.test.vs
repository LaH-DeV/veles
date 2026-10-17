use fmt

test thousands "groups digits in threes" {
  expect(fmt.thousands(0) == "0")
  expect(fmt.thousands(999) == "999")
  expect(fmt.thousands(1000) == "1,000")
  expect(fmt.thousands(-1234567) == "-1,234,567")
}

test bytes "picks the largest binary unit" {
  expect(fmt.bytes(0) == "0 B")
  expect(fmt.bytes(12) == "12 B")
  expect(fmt.bytes(1024) == "1.0 KiB")
  expect(fmt.bytes(1536) == "1.5 KiB")
  expect(fmt.bytes(5 * 1024 * 1024) == "5.0 MiB")
  expect(fmt.bytes(-2048) == "-2.0 KiB")
}

test percent "scales a fraction" {
  expect(fmt.percent(0.256) == "25.6%")
  expect(fmt.percent(1.0, 0) == "100%")
  expect(fmt.percent(0.5, 2) == "50.00%")
}

test ordinal "uses st nd rd th" {
  expect(fmt.ordinal(1) == "1st")
  expect(fmt.ordinal(2) == "2nd")
  expect(fmt.ordinal(3) == "3rd")
  expect(fmt.ordinal(4) == "4th")
  expect(fmt.ordinal(11) == "11th")
  expect(fmt.ordinal(12) == "12th")
  expect(fmt.ordinal(22) == "22nd")
  expect(fmt.ordinal(113) == "113th")
}

test duration "writes the largest units" {
  expect(fmt.duration(Duration.zero) == "0s")
  expect(fmt.duration(Duration.nanos(7)) == "7ns")
  expect(fmt.duration(Duration.micros(12)) == "12us")
  expect(fmt.duration(Duration.millis(250)) == "250ms")
  expect(fmt.duration(Duration.seconds(5)) == "5s")
  expect(fmt.duration(Duration.seconds(3725)) == "1h 2m 5s")
  expect(fmt.duration(Duration.days(2)) == "2d")
  expect(fmt.duration(Duration.seconds(-90)) == "-1m 30s")
}

test table "aligns columns by characters" {
  val out = fmt.table([["name", "n"], ["café", "12"], ["tea", "3"]], right: [1], header: true)
  expect(out == "name   n\n----  --\ncafé  12\ntea    3")
}
