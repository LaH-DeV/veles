// D166: a wire shape names its fields, so a tuple — labelled or not — is not
// Codable; the derive names the field instead of a missing method.
struct Score {
  name: string
  best: (round: i64, points: i64)
  implement Encodable // error: cannot derive 'Encodable' for 'Score': field 'best' is a tuple, which has no wire form — make it a struct, whose fields carry the keys; mark it @skip (with a default) or write the impl (D58)
}

struct Board {
  rows: List<(string, i64)>
  implement Encodable // error: cannot derive 'Encodable' for 'Board': field 'rows' holds elements that each is a tuple, which has no wire form — make it a struct, whose fields carry the keys; mark it @skip (with a default) or write the impl (D58)
}

fun main() { }
