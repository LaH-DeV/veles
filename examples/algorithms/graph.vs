// Graph algorithms over an adjacency list. Nodes are 0..<n; `Graph` is a
// value struct holding a reference-typed list (D25), so a `val` graph can
// still grow through `addEdge` — see the note in the spec on D22/D25.
// The queues are the prelude's Deque; the priority queue in Dijkstra is
// the prelude's PriorityQueue.

pub struct Graph {
  n:   i64
  adj: MutableList<MutableList<i64>>

  static fun withNodes(n: i64): Graph =
    Graph(n: n, adj: MutableList<MutableList<i64>>.make(n, _ => []))

  fun addEdge(a: i64, b: i64) {
    self.adj.atOrPanic(a).push(b)
  }

  fun addUndirected(a: i64, b: i64) {
    self.addEdge(a, b)
    self.addEdge(b, a)
  }

  fun neighbours(v: i64): List<i64> = self.adj.atOrPanic(v).toList()

  /// Breadth-first order from `start`.
  fun bfs(start: i64): List<i64> {
    val seen = self.flags()
    val order: MutableList<i64> = []
    val queue = deque<i64>()
    queue.addLast(start)
    seen.set(start, true)
    loop {
      val v = queue.removeFirst() ?: break
      order.push(v)
      loop (w in self.neighbours(v)) {
        if (seen.atOrPanic(w)) continue
        seen.set(w, true)
        queue.addLast(w)
      }
    }
    order.toList()
  }

  /// Depth-first order from `start`, recursive.
  fun dfs(start: i64): List<i64> {
    val seen = self.flags()
    val order: MutableList<i64> = []
    self.dfsFrom(start, seen, order)
    order.toList()
  }

  fun dfsFrom(v: i64, seen: MutableList<bool>, order: MutableList<i64>) {
    seen.set(v, true)
    order.push(v)
    loop (w in self.neighbours(v)) {
      if (!seen.atOrPanic(w)) self.dfsFrom(w, seen, order)
    }
  }

  /// Fewest edges from `start` to every node; null where unreachable.
  fun distances(start: i64): List<i64?> {
    val dist = MutableList<i64?>.repeat(null, self.n)
    dist.set(start, 0)
    val queue = deque<i64>()
    queue.addLast(start)
    loop {
      val v = queue.removeFirst() ?: break
      val d = dist.atOrPanic(v) ?: 0
      loop (w in self.neighbours(v)) {
        if (dist.atOrPanic(w) != null) continue
        dist.set(w, d + 1)
        queue.addLast(w)
      }
    }
    dist.toList()
  }

  /// Kahn's algorithm; null when the graph has a cycle.
  fun topologicalOrder(): List<i64>? {
    val indegree = MutableList<i64>.repeat(0, self.n)
    loop (v in 0..<self.n) {
      loop (w in self.neighbours(v)) *indegree.refOrPanic(w) += 1
    }
    val ready = deque<i64>()
    loop (v in 0..<self.n) {
      if (indegree.atOrPanic(v) == 0) ready.addLast(v)
    }
    val order: MutableList<i64> = []
    loop {
      val v = ready.removeFirst() ?: break
      order.push(v)
      loop (w in self.neighbours(v)) {
        *indegree.refOrPanic(w) -= 1
        if (indegree.atOrPanic(w) == 0) ready.addLast(w)
      }
    }
    if (order.len() == self.n) order.toList() else null
  }

  /// Number of connected components (treating edges as undirected).
  fun components(): i64 {
    val seen = self.flags()
    var count = 0
    loop (v in 0..<self.n) {
      if (seen.atOrPanic(v)) continue
      count += 1
      val order: MutableList<i64> = []
      self.dfsFrom(v, seen, order)
    }
    count
  }

  fun flags(): MutableList<bool> = MutableList<bool>.repeat(false, self.n)
}

/// A pending step for Dijkstra, ordered by distance so the priority queue
/// yields the closest node first.
struct Hop {
  dist: i64
  node: i64

  impl Comparable {
    fun compareTo(other: Hop): i64 = self.dist.compareTo(other.dist)
  }
}

/// Dijkstra on a weighted graph given as (from, to, weight) edges, with a
/// priority queue of pending hops: O((n + m) log n). An entry pushed before
/// a shorter path to its node was found is stale and skipped when it comes
/// out.
pub fun dijkstra(n: i64, edges: List<(i64, i64, i64)>, start: i64): List<i64?> {
  val adj = MutableList<MutableList<(i64, i64)>>.make(n, _ => [])
  loop ((from, to, weight) in edges) adj.atOrPanic(from).push((to, weight))
  val dist = MutableList<i64?>.repeat(null, n)
  dist.set(start, 0)
  val pending = priorityQueue<Hop>()
  pending.push(Hop(dist: 0, node: start))
  loop {
    val hop = pending.pop() ?: break
    val best = dist.atOrPanic(hop.node) ?: hop.dist
    if (hop.dist > best) continue
    loop ((to, weight) in adj.atOrPanic(hop.node)) {
      val candidate = hop.dist + weight
      val known = dist.atOrPanic(to)
      if (known != null && known <= candidate) continue
      dist.set(to, candidate)
      pending.push(Hop(dist: candidate, node: to))
    }
  }
  dist.toList()
}
