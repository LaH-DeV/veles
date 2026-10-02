// D112: deriving Encodable over a Secret field is refused with the way out;
// Decodable is fine, and a skipped Secret encodes.
use io

struct Login {
  user: string
  password: Secret<string>
  implement Codable // error: cannot derive 'Encodable' for 'Login': field 'password' is a Secret, which is never written out (D112); mark it @skip (with a default) or write 'encode' by hand (D58)
}

struct Settings {
  apiKey: Secret<string>
  implement Decodable
}

fun main() {
  io.println("ok")
}
