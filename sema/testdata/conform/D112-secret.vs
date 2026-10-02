// D112: a Secret holds text or bytes; it is never hashed, ordered or
// written out, and says so.
use io, json

fun main() throws EncodeError {
  with s = Secret.of("pw")
  with b = Secret.of("key".bytes())
  val set: Set<Secret<string>> = [] // error: 'Secret<string>' cannot be a map key or set element: a Secret is never hashed, so its bytes cannot leak through a hash or a key's order (D112)
  io.println("${s < s}") // error: operator '<' is not defined for 'Secret<string>': a Secret is never ordered, only compared with '==' (D112)
  io.println(try json.encode(s)) // error: type 'Secret<string>' does not implement trait 'Encodable' required by parameter 'T' of 'encode'; a Secret is never written out (D112)
  val number = Secret.of(42) // error: a Secret holds text or bytes: 'Secret<string>' or 'Secret<List<u8>>', not 'Secret<i64>' (D112)
  val owned: Secret<MutableList<u8>>? = null // error: a Secret holds text or bytes: 'Secret<string>' or 'Secret<List<u8>>', not 'Secret<MutableList<u8>>' (D112)
  io.println("$s $b ${s == Secret.of(b.expose().decodeUtf8() ?: "")} ${set.len()} $number $owned")
}
