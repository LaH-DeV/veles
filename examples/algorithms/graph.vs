// Graph algorithms over an adjacency list. Nodes are 0..<n; `Graph` is a
// value struct holding a reference-typed list (D25), so a `val` graph can
// still grow through `addEdge` — see the note in the spec on D22/D25.

pub struct Graph {
  n:   i64
  adj: MutableList<MutableList<i64>>

  static fun withNodes(n: i64): Graph {
    var adj: MutableList<MutableList<i64>> = []
    loop (_ in 0..<n) adj.push([])
    Graph(n: n, adj: adj)
  }

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
    var seen = self.flags()
    var order: MutableList<i64> = []
    var queue: MutableList<i64> = [start]
    var head = 0
    seen.set(start, true)
    loop (head < queue.len()) {
      val v = queue.atOrPanic(head)
      head += 1
      order.push(v)
      loop (w in self.neighbours(v)) {
        if (!seen.atOrPanic(w)) {
          seen.set(w, true)
          queue.push(w)
        }
      }
    }
    order.toList()
  }

  /// Depth-first order from `start`, recursive.
  fun dfs(start: i64): List<i64> {
    var seen = self.flags()
    var order: MutableList<i64> = []
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
    var dist: MutableList<i64?> = []
    loop (_ in 0..<self.n) dist.push(null)
    dist.set(start, 0)
    var queue: MutableList<i64> = [start]
    var head = 0
    loop (head < queue.len()) {
      val v = queue.atOrPanic(head)
      head += 1
      val d = dist.atOrPanic(v) ?: 0
      loop (w in self.neighbours(v)) {
        if (dist.atOrPanic(w) == null) {
          dist.set(w, d + 1)
          queue.push(w)
        }
      }
    }
    dist.toList()
  }

  /// Kahn's algorithm; null when the graph has a cycle.
  fun topologicalOrder(): List<i64>? {
    var indegree: MutableList<i64> = []
    loop (_ in 0..<self.n) indegree.push(0)
    loop (v in 0..<self.n) {
      loop (w in self.neighbours(v)) indegree.set(w, indegree.atOrPanic(w) + 1)
    }
    var ready: MutableList<i64> = []
    loop (v in 0..<self.n) {
      if (indegree.atOrPanic(v) == 0) ready.push(v)
    }
    var order: MutableList<i64> = []
    var head = 0
    loop (head < ready.len()) {
      val v = ready.atOrPanic(head)
      head += 1
      order.push(v)
      loop (w in self.neighbours(v)) {
        indegree.set(w, indegree.atOrPanic(w) - 1)
        if (indegree.atOrPanic(w) == 0) ready.push(w)
      }
    }
    if (order.len() == self.n) order.toList() else null
  }

  /// Number of connected components (treating edges as undirected).
  fun components(): i64 {
    var seen = self.flags()
    var count = 0
    loop (v in 0..<self.n) {
      if (seen.atOrPanic(v)) continue
      count += 1
      var order: MutableList<i64> = []
      self.dfsFrom(v, seen, order)
    }
    count
  }

  fun flags(): MutableList<bool> {
    var out: MutableList<bool> = []
    loop (_ in 0..<self.n) out.push(false)
    out
  }
}

/// Dijkstra on a weighted graph given as (from, to, weight) edges, with an
/// O(n²) "pick the closest unvisited node" scan — fine for small graphs and
/// needs no priority queue.
pub fun dijkstra(n: i64, edges: List<(i64, i64, i64)>, start: i64): List<i64?> {
  var dist: MutableList<i64?> = []
  var done: MutableList<bool> = []
  loop (_ in 0..<n) {
    dist.push(null)
    done.push(false)
  }
  dist.set(start, 0)
  loop (_ in 0..<n) {
    // closest unvisited node
    var best: i64? = null
    loop (v in 0..<n) {
      if (done.atOrPanic(v)) continue
      val dv = dist.atOrPanic(v)
      if (dv == null) continue
      val b = best
      if (b == null || dv < (dist.atOrPanic(b) ?: 0)) best = v
    }
    val u = best ?: break
    done.set(u, true)
    val du = dist.atOrPanic(u) ?: 0
    loop ((from, to, w) in edges) {
      if (from != u) continue
      val candidate = du + w
      val current = dist.atOrPanic(to)
      if (current == null || candidate < current) dist.set(to, candidate)
    }
  }
  dist.toList()
}
