// Classic algorithms, one file per topic: sorting.vs, searching.vs,
// numbers.vs, strings.vs, graph.vs, dp.vs, structures.vs. Every .vs in a
// directory shares one namespace, so main calls them directly.
use io { println }

fun main() {
  println("== sorting")
  val data = [5, 2, 9, 1, 5, 6, 0, 3]
  println("bubble    ${bubbleSort(data)}")
  println("insertion ${insertionSort(data)}")
  println("selection ${selectionSort(data)}")
  println("merge     ${mergeSort(data)}")
  println("quick     ${quickSort(data)}")
  println("counting  ${countingSort(data, 9)}")
  println("heap      ${heapSort(data)}")
  println("words     ${insertionSort(["pear", "apple", "fig"])}")
  println("sorted? ${isSorted(data)} ${isSorted(mergeSort(data))}")

  println("== searching")
  val sorted = mergeSort(data)
  println("linear 9 at ${linearSearch(data, 9)}, 7 at ${linearSearch(data, 7)}")
  println("binary 6 at ${binarySearch(sorted, 6)}, 4 at ${binarySearch(sorted, 4)}")
  println("binaryRec 0 at ${binarySearchRec(sorted, 0, 0, sorted.len() - 1)}")
  println("lowerBound 4 -> ${lowerBound(sorted, 4)}, 100 -> ${lowerBound(sorted, 100)}")
  println("prelude on $sorted: 6 at ${sorted.binarySearch(6)}, 4 at ${sorted.binarySearch(4)}")
  println("prelude 5 occupies [${sorted.lowerBound(5)}, ${sorted.upperBound(5)})")
  println("partitionPoint x <= 5 -> ${sorted.partitionPoint(x => x <= 5)}")
  val names = ["apple", "fig", "pear"]
  println("binarySearchBy fig -> ${names.binarySearchBy(w => w, "fig")}, kiwi -> ${names.binarySearchBy(w => w, "kiwi")}")
  println("binarySearchWith, descending: ${[9, 6, 5, 1].binarySearchWith(x => (5).compareTo(x))}")
  println("the two implementations agree: ${preludeAgrees(data)}")
  println("pair summing to 11: ${pairWithSum(sorted, 11)}, to 100: ${pairWithSum(sorted, 100)}")

  println("== numbers")
  println("gcd(84, 36) = ${gcd(84, 36)}, lcm(4, 6) = ${lcm(4, 6)}")
  println("primes to 50: ${sieve(50)}")
  println("isPrime 97 ${isPrime(97)}, 91 ${isPrime(91)}")
  val memo: MutableMap<i64, i64> = [:]
  println("fib(30) = ${fib(30)} memo ${fibMemo(30, memo)} (table ${memo.len()} entries)")
  println("2^10 = ${power(2, 10)}, 3^13 mod 7 = ${powMod(3, 13, 7)}, 10! = ${factorial(10)}")
  println("digits(90210) = ${digits(90210)}, isqrt(99) = ${isqrt(99)}")
  println("360 = ${factorize(360)}")

  println("== strings")
  println("reverse ${reverse("héllo")}")
  println("palindrome ${isPalindrome("A man a plan a canal Panama")} ${isPalindrome("veles")}")
  println("anagram ${isAnagram("Listen", "Silent")} ${isAnagram("abc", "abd")}")
  println("counts ${wordCounts("the cat and the hat and the bat")}")
  println("rle ${runLengthEncode("aaabccdddd")}")
  println("caesar ${caesar("Hello, World!", 3)} -> ${caesar(caesar("Hello, World!", 3), -3)}")
  println("find naive ${findNaive("the quick brown fox", "brown")} kmp ${findKmp("the quick brown fox", "brown")} missing ${findKmp("abc", "zz")}")
  println("kmp aabaaab in aaabaabaaab: ${findKmp("aaabaabaaab", "aabaaab")}")
  println("prefix ${commonPrefix(["interstellar", "internet", "interval"])}")

  println("== graph")
  val g = Graph.withNodes(7)
  g.addUndirected(0, 1)
  g.addUndirected(0, 2)
  g.addUndirected(1, 3)
  g.addUndirected(2, 3)
  g.addUndirected(3, 4)
  g.addUndirected(5, 6)
  println("bfs ${g.bfs(0)}")
  println("dfs ${g.dfs(0)}")
  println("dist ${g.distances(0)}")
  println("components ${g.components()}")
  val dag = Graph.withNodes(6)
  dag.addEdge(5, 2)
  dag.addEdge(5, 0)
  dag.addEdge(4, 0)
  dag.addEdge(4, 1)
  dag.addEdge(2, 3)
  dag.addEdge(3, 1)
  println("topo ${dag.topologicalOrder()}")
  dag.addEdge(1, 5)
  println("topo with cycle ${dag.topologicalOrder()}")
  val edges = [(0, 1, 4), (0, 2, 1), (2, 1, 2), (1, 3, 1), (2, 3, 5)]
  println("dijkstra ${dijkstra(5, edges, 0)}")

  println("== dp")
  println("lcs ${lcsLength("ABCBDAB", "BDCABA")}")
  println("edit ${editDistance("kitten", "sitting")}")
  println("knapsack ${knapsack([1, 3, 4, 5], [1, 4, 5, 7], 7)}")
  println("coins ${coinChange([1, 2, 5], 11)} ${coinChange([2], 3)}")
  println("lis ${longestIncreasing([10, 9, 2, 5, 3, 7, 101, 18])}")
  println("kadane ${maxSubarraySum([-2, 1, -3, 4, -1, 2, 1, -5, 4])}")
  println("stairs ${climbStairs(10)}")

  println("== structures")
  val q = Deque<string>()
  loop (s in ["a", "b", "c"]) q.addLast(s)
  q.addFirst("z")
  println("deque $q first ${q.removeFirst()} last ${q.removeLast()} -> $q len ${q.len()}")
  println("sliding max ${slidingMax([1, 3, -1, -3, 5, 3, 6, 7], 3)}")
  println("balanced ${balanced("{[()]}")} ${balanced("([)]")} ${balanced("((")}")
  var tree = Bst()
  loop (v in [50, 30, 70, 20, 40, 60, 80, 30]) tree.insert(v)
  println("bst ${tree.inOrder()} size ${tree.size} height ${tree.height()} min ${tree.min()} max ${tree.max()}")
  println("bst has 60 ${tree.contains(60)}, 65 ${tree.contains(65)}")
  val heap = MinHeap()
  loop (x in [7, 3, 9, 1, 4]) heap.push(x)
  println("heap peek ${heap.peek()} pop ${heap.pop()} ${heap.pop()} len ${heap.len()}")
  val words = PriorityQueue<string>.natural()
  loop (w in ["pear", "apple", "fig"]) words.push(w)
  println("priority ${words.pop()} ${words.pop()} ${words.pop()} ${words.pop()}")

  // `sorted()` on integers and strings sorts natively; it must agree with
  // the comparator path at every size around its runs of 32
  var seed: u64 = 12345
  var agree = 0
  loop (n in [0, 1, 2, 31, 32, 33, 64, 65, 1000, 4097]) {
    val ints: MutableList<i64> = []
    val bytes: MutableList<u8> = []
    val texts: MutableList<string> = []
    loop (_ in 0..<n) {
      seed = seed ^ (seed << 13)
      seed = seed ^ (seed >> 7)
      seed = seed ^ (seed << 17)
      ints.push((seed % 2001) as i64 - 1000)
      bytes.push((seed % 256) as u8)
      texts.push("ż${seed % 97}")
    }
    if (ints.sorted() == ints.sortedWith((a, b) => a.compareTo(b)) &&
      bytes.sorted() == bytes.sortedWith((a, b) => a.compareTo(b)) &&
      texts.sorted() == texts.sortedWith((a, b) => a.compareTo(b))) agree += 1
  }
  println("sorted agrees at $agree of 10 sizes: ${[3, -1, 2, -7].sorted()} ${["b", "ą", "a"].sorted()}")
}
