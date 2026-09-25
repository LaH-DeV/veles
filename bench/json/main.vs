// JSON: encode 2000 records and decode them back, 20 times (std/json).
use io, json, time

struct Record {
  id:    i64
  name:  string
  score: f64
  tags:  List<string>
  implement Codable
}

fun main() throws EncodeError | DecodeError {
  var records: MutableList<Record> = []
  loop (i in 0..<2000) records.push(Record(id: i, name: "record number $i", score: (i as f64) * 0.5, tags: ["a", "bb", "ccc"]))
  val input = records.toList()
  val sw = time.Stopwatch.start()
  var check: i64 = 0
  loop (_ in 0..<20) {
    val text = try json.encode(input)
    val back = try json.decode<List<Record>>(text)
    loop (r in back) check += r.id
  }
  io.println("BENCH json 40000 ${sw.elapsed().toNanos()} $check")
}
