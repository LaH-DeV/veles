// Graph algorithms over an adjacency list. Nodes are 0..<n; `Graph` is a
// value struct holding a reference-typed list (D25), so a `val` graph can
// still grow through `addEdge` — see the note in the spec on D22/D25.
// The queues are the prelude's Deque; the priority queue in Dijkstra is
// the prelude's PriorityQueue.

public struct Graph {
  n:   i64
  adj: MutableList<MutableList<i64>>

  static fun withNodes(n: i64): Graph =
    Graph(n, adj: MutableList<MutableList<i64>>.make(n, _ => []))

  fun addEdge(a: i64, b: i64) {
    if (b < 0 || b >= this.n) panic("graph: node $b is not in 0..<${this.n}")
    val edges = this.adj.at(a) ?: panic("graph: node $a is not in 0..<${this.n}")
    edges.push(b)
  }

  fun addUndirected(a: i64, b: i64) {
    this.addEdge(a, b)
    this.addEdge(b, a)
  }

  fun neighbours(v: i64): List<i64> = slot(this.adj, v).toList()

  /// Breadth-first order from `start`.
  fun bfs(start: i64): List<i64> {
    val seen = this.flags()
    val order: MutableList<i64> = []
    val queue = deque<i64>()
    queue.addLast(start)
    seen.set(start, true)
    loop {
      val v = queue.removeFirst() ?: break
      order.push(v)
      loop (w in this.neighbours(v)) {
        if (slot(seen, w)) continue
        seen.set(w, true)
        queue.addLast(w)
      }
    }
    order.toList()
  }

  /// Depth-first order from `start`, recursive.
  fun dfs(start: i64): List<i64> {
    val seen = this.flags()
    val order: MutableList<i64> = []
    this.dfsFrom(start, seen, order)
    order.toList()
  }

  fun dfsFrom(v: i64, seen: MutableList<bool>, order: MutableList<i64>) {
    seen.set(v, true)
    order.push(v)
    loop (w in this.neighbours(v)) {
      if (!slot(seen, w)) this.dfsFrom(w, seen, order)
    }
  }

  /// Fewest edges from `start` to every node; null where unreachable.
  fun distances(start: i64): List<i64?> {
    val dist = MutableList<i64?>.repeat(null, this.n)
    dist.set(start, 0)
    val queue = deque<i64>()
    queue.addLast(start)
    loop {
      val v = queue.removeFirst() ?: break
      val d = slot(dist, v) ?: 0
      loop (w in this.neighbours(v)) {
        if (slot(dist, w) != null) continue
        dist.set(w, d + 1)
        queue.addLast(w)
      }
    }
    dist.toList()
  }

  /// Kahn's algorithm; null when the graph has a cycle.
  fun topologicalOrder(): List<i64>? {
    val indegree = MutableList<i64>.repeat(0, this.n)
    loop (v in 0..<this.n) {
      loop (w in this.neighbours(v)) indegree.set(w, slot(indegree, w) + 1)
    }
    val ready = deque<i64>()
    loop (v in 0..<this.n) {
      if (slot(indegree, v) == 0) ready.addLast(v)
    }
    val order: MutableList<i64> = []
    loop {
      val v = ready.removeFirst() ?: break
      order.push(v)
      loop (w in this.neighbours(v)) {
        val left = slot(indegree, w) - 1
        indegree.set(w, left)
        if (left == 0) ready.addLast(w)
      }
    }
    if (order.len() == this.n) order.toList() else null
  }

  /// Number of connected components (treating edges as undirected).
  fun components(): i64 {
    val seen = this.flags()
    var count = 0
    loop (v in 0..<this.n) {
      if (slot(seen, v)) continue
      count += 1
      val order: MutableList<i64> = []
      this.dfsFrom(v, seen, order)
    }
    count
  }

  fun flags(): MutableList<bool> = MutableList<bool>.repeat(false, this.n)
}

/// Entry `v` of a table with one slot per node. Every node an edge names is
/// in `0..<n` — `addEdge` checks it — so this cannot fail.
fun slot<T>(table: MutableList<T>, v: i64): T = table.at(v) ?: panic("graph: every node is in 0..<n")

/// A pending step for Dijkstra, ordered by distance so the priority queue
/// yields the closest node first.
struct Hop {
  dist: i64
  node: i64

  implement Comparable {
    fun compareTo(other: Hop): Ordering = this.dist.compareTo(other.dist)
  }
}

/// Dijkstra on a weighted graph given as (from, to, weight) edges, with a
/// priority queue of pending hops: O((n + m) log n). An entry pushed before
/// a shorter path to its node was found is stale and skipped when it comes
/// out.
public fun dijkstra(n: i64, edges: List<(i64, i64, i64)>, start: i64): List<i64?> {
  val adj = MutableList<MutableList<(i64, i64)>>.make(n, _ => [])
  loop ((from, to, weight) in edges) {
    if (to < 0 || to >= n) panic("dijkstra: an edge goes to $to, outside 0..<$n")
    val out = adj.at(from) ?: panic("dijkstra: an edge comes from $from, outside 0..<$n")
    out.push((to, weight))
  }
  val dist = MutableList<i64?>.repeat(null, n)
  dist.set(start, 0)
  val pending = priorityQueue<Hop>()
  pending.push(Hop(dist: 0, node: start))
  loop {
    val hop = pending.pop() ?: break
    val best = slot(dist, hop.node) ?: hop.dist
    if (hop.dist > best) continue
    loop ((to, weight) in slot(adj, hop.node)) {
      val candidate = hop.dist + weight
      val known = slot(dist, to)
      if (known != null && known <= candidate) continue
      dist.set(to, candidate)
      pending.push(Hop(dist: candidate, node: to))
    }
  }
  dist.toList()
}
