// D113: constants are computed by the compiler. A scalar is written in at
// each use; a table is laid out once, read-only, and every use points at
// it — no code builds it at start-up.
use io

const KB: i64 = 1024
const MB = KB * 1024
const LABEL = "size ${MB}"
const PRIMES: List<i64> = [2, 3, 5]
const NESTED: List<List<u8>> = [[1], []]
const WORDS: Map<string, i64> = ["one": 1, "two": 2]

fun size(): i64 = MB + PRIMES.at(1)

fun main() {
  io.println("${size()} ${LABEL} ${PRIMES.len()} ${PRIMES} ${NESTED} ${WORDS.get("two") ?: 0}")
}
