// Tests of `time.ticker` (D110): ticks arrive each period, a slow reader
// finds one waiting rather than a queue, and a closed ticker stops.

test "ticks arrive about once a period" {
  with clock = ticker(Duration.millis(10))
  val sw = Stopwatch.start()
  loop (_ in 0..<5) {
    val at = await clock.ticks.recv()
    expect(at != null)
  }
  val ms = sw.elapsed().toMillis()
  expect(ms >= 40)
  expect(ms < 1000)
}

test "a slow reader misses ticks instead of queueing them" {
  with clock = ticker(Duration.millis(5))
  await sleep(Duration.millis(60))
  var waiting = 0
  loop {
    val _ = clock.ticks.tryRecv() ?: break
    waiting += 1
  }
  expect(waiting == 1)
}

test "closing a ticker stops it and closes its channel, once" {
  val clock = ticker(Duration.millis(5))
  await sleep(Duration.millis(20))
  clock.close()
  // a tick already waiting is still received, as from any closed channel;
  // nothing comes after it
  var after = 0
  loop {
    val _ = await clock.ticks.recv() ?: break
    after += 1
  }
  expect(after <= 1)
  await sleep(Duration.millis(20))
  expect(clock.ticks.tryRecv() == null)
  clock.close()
}

test "a tick is the time it was sent" {
  val before = now()
  with clock = ticker(Duration.millis(5))
  val at = await clock.ticks.recv() ?: Timestamp.epoch
  expect(at >= before)
  expect(at <= now())
}

test "a ticker panics for a period that is not positive" {
  expectPanics(() => {
    val _ = ticker(Duration.zero)
  })
}
