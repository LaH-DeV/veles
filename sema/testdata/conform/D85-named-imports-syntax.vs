// D85: what a braced import may not be.
use json { * } // error: 'use json { * }' does not exist: name what you use, so a reader sees where each name comes from (D85)
use random { } // error: 'use random { }' names nothing: list what to import, or write 'use random'
use time.{ sleep } // error: 'use time.{ … }' is spelled 'use time { … }' (D85)

fun main() { }
