// D82: `os.run`'s `mergeStderr:` became `stderr: os.Stderr`; the old flag
// still parses, and its fix keeps what it meant.
use io, os

fun merged(): string throws = (try os.run("git", ["--version"], mergeStderr: true)).stdout // error: 'mergeStderr' was removed: write 'stderr: os.Stderr.Merge' (D82)
fun passedThrough(): string throws = (try os.run("git", ["--version"], mergeStderr: false)).stdout // error: 'mergeStderr' was removed: write 'stderr: os.Stderr.Inherit' (D82)

fun main() {
  io.println("")
}
