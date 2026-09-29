// D85: a braced import asks for what the module has, and what it may share.
use path { dirr } // error: module 'path' has no declaration 'dirr'; did you mean 'dir'?
use io { println, eprintln as println } // error: 'println' is already imported in this file: import one of them under another name, 'io { eprintln as … }'
use fs { readFile }, os { args as readFile } // error: 'readFile' is already imported in this file: import one of them under another name, 'os { args as … }'
use utf8 { isScalar } // error: 'isScalar' is already declared in this module (at
use json { Duration } // error: module 'json' has no declaration 'Duration'; 'Duration' is global (the prelude): it needs no import
use hex { Value } // error: module 'hex' has no declaration 'Value'; it is 'codec.Value': import it from 'codec'

fun isScalar() { }

fun main() {
  println("")
}
