// std/random: a seeded generator gives the same sequence every run, which
// is what this example relies on; without `seed` it starts from the clock.
// Also bitwise operators, which the generator is built from.
use io
use random

fun main() {
  random.seed(2026)
  val dice: List<i64> = (1..5).iter().map(i => random.range(1, 7)).toList()
  io.println("dice $dice")
  val deck = mut ["A", "K", "Q", "J", "10"]
  random.shuffle(deck)
  io.println("shuffled $deck, pick ${random.pick(deck)}")
  io.println("float ${random.float().toFixed(3)} coin ${random.boolean()}")

  var rng = random.Rng.seeded(7)
  var rng2 = random.Rng.seeded(7)
  io.println("own generators agree: ${rng.range(0, 1000) == rng2.range(0, 1000)}")

  val flags: u8 = 0b1010
  io.println("${flags & 0b0010} ${flags | 1} ${flags ^ 0xFF} ${~flags} ${flags << 4} ${flags >> 1} ${(-16) >> 2} ${(255).toString(radix: 2)}")
}
