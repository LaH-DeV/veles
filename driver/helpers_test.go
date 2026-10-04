package driver

import (
	"os"
	"path/filepath"
	"testing"
)

// D110: the prelude's concurrency helpers (retry, Semaphore, channel
// drains) on eight threads — written out here, since the prelude cannot be
// compiled as a package of its own — and std/time's ticker.
func TestConcurrencyHelpers(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	t.Setenv("VELES_THREADS", "8")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "helpers.test.vs"), []byte(helpersSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Run(Options{Path: dir, Mode: "test"}); code != 0 {
		t.Fatalf("veles test (the helpers): exit %d", code)
	}
	if code := Run(Options{Path: filepath.Join("..", "std", "time"), Mode: "test"}); code != 0 {
		t.Fatalf("veles test std/time: exit %d", code)
	}
}

const helpersSrc = `error Flaky { }

// fails its first "failures" calls, then returns the number of the call
test fun flakyCall(calls: Atomic<i64>, failures: i64): i64 throws Flaky {
  val n = calls.update(c => c + 1)
  if (n <= failures) throw Flaky()
  n
}

test "retry calls again after an error and returns the first value" {
  val calls = Atomic(value: 0)
  val v = try retry(3, () => try flakyCall(calls, 2))
  expect(v == 3)
  expect(calls.load() == 3)
}

test "retry throws the last error after the last call" {
  val calls = Atomic(value: 0)
  expect(retry(2, () => try flakyCall(calls, 5)) is Err)
  expect(calls.load() == 2)
}

test fun threeSlowFailures(calls: Atomic<i64>): i64 throws Flaky {
  try retry(3, () => try flakyCall(calls, 9), delay: Duration.millis(40))
}

test "retry waits the delay between calls, not after the last one" {
  // two waits of 40ms: not done in 60, done well within 1000
  expect(withTimeout(Duration.millis(60), () => try threeSlowFailures(Atomic(value: 0))) is Err)
  val calls = Atomic(value: 0)
  val r = withTimeout(Duration.seconds(1), () => try threeSlowFailures(calls))
  expect(calls.load() == 3)
  expect(r is Err)
}

test fun retryForever(calls: Atomic<i64>) {
  val _ = retry(1000000, () => try flakyCall(calls, 1000000), delay: Duration.seconds(10))
}

test "a cancelled retry stops at the wait" {
  val calls = Atomic(value: 0)
  expect(withTimeout(Duration.millis(50), () => retryForever(calls)) is Err)
  expect(calls.load() == 1)
}

test fun retryNoCalls(): i64 = retry(0, () => 1)

test "retry panics for fewer than one call" {
  // retry suspends, and expectPanics takes a function that does not
  expect(gather {
    async retryNoCalls()
  } is Err)
}

test fun semaphoreWorker(sem: Semaphore, inside: Atomic<i64>, most: Atomic<i64>) {
  with sem.acquire()
  val now = inside.update(n => n + 1)
  most.update(m => m.max(now))
  await sleep(Duration.millis(1))
  inside.update(n => n - 1)
}

test "a Semaphore never lets more than its permits in" {
  val sem = Semaphore(permits: 3)
  val inside = Atomic(value: 0)
  val most = Atomic(value: 0)
  scope {
    loop (_ in 0..<64) {
      async semaphoreWorker(sem, inside, most)
    }
  }
  expect(most.load() <= 3)
  expect(most.load() >= 1)
  expect(sem.available() == 3)
}

test "tryAcquire takes a free permit or returns null, and a second close gives nothing back" {
  val sem = Semaphore(permits: 1)
  val p = sem.tryAcquire()
  expect(p is Permit)
  expect(sem.tryAcquire() !is Permit)
  p?.close()
  p?.close()
  expect(sem.available() == 1)
}

test "a Semaphore panics for fewer than one permit" {
  expectPanics(() => {
    val _ = Semaphore(permits: 0)
  })
}

test fun feedThree(ch: Channel<i64>) {
  loop (i in 1..3) {
    ch.send(i)
  }
  ch.close()
}

test "toList receives until the channel is closed and drained" {
  val ch = Channel<i64>(capacity: 1)
  scope {
    async feedThree(ch)
    expect(ch.toList() == [1, 2, 3])
  }
}

test "forEach calls f on every value until the close" {
  val ch = Channel<i64>()
  val seen = Atomic(value: 0)
  scope {
    async feedThree(ch)
    ch.forEach(v => {
      seen.update(n => n + v)
    })
  }
  expect(seen.load() == 6)
}

test "an error from forEach's f stops it" {
  val ch = Channel<i64>(capacity: 3)
  feedThree(ch)
  val calls = Atomic(value: 0)
  expect(ch.forEach(v => {
    val _ = try flakyCall(calls, 1)
  }) is Err)
  expect(calls.load() == 1)
}
`
