// Command bench runs the Veles benchmarks against a Go reference of the
// same workload, so each number reads as a multiple of Go rather than a
// time that means nothing on its own.
//
//	go run ./bench                 run every benchmark, print the table
//	go run ./bench -run json       only the ones whose name contains "json"
//	go run ./bench -record         also append the table to bench/results.md
//
// Every benchmark is a directory bench/<name>/main.vs that times its own
// workload and prints `BENCH <name> <ops> <nanoseconds> <checksum>`. The Go
// reference computes the same checksum, and a mismatch fails the run: the
// two sides must provably do the same work before their times are compared.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type result struct {
	name     string
	ops      int64
	veles    time.Duration
	goTime   time.Duration
	checksum string
}

func main() {
	filter := flag.String("run", "", "only benchmarks whose name contains this")
	record := flag.Bool("record", false, "append the results to bench/results.md")
	flag.Parse()

	root := moduleRoot()
	tmp, err := os.MkdirTemp("", "veles-bench-")
	must(err)
	defer os.RemoveAll(tmp)
	veles := filepath.Join(tmp, "veles"+exe())
	run(root, "go", "build", "-o", veles, ".")

	dirs, err := filepath.Glob(filepath.Join(root, "bench", "*", "main.vs"))
	must(err)
	var results []result
	failed := false
	for _, main := range dirs {
		dir := filepath.Dir(main)
		name := filepath.Base(dir)
		if !strings.Contains(name, *filter) {
			continue
		}
		ref, ok := references[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "bench: %s has no Go reference\n", name)
			failed = true
			continue
		}
		bin := filepath.Join(tmp, name+exe())
		run(root, veles, "build", dir, "-o", bin, "--release")
		out, err := exec.Command(bin).Output()
		must(err)
		r := parse(name, out)
		start := time.Now()
		sum := ref()
		r.goTime = time.Since(start)
		if sum != r.checksum {
			fmt.Fprintf(os.Stderr, "bench: %s: Veles computed %s, Go %s — not the same work\n", name, r.checksum, sum)
			failed = true
		}
		results = append(results, r)
		fmt.Fprintf(os.Stderr, "  %-10s done\n", name)
	}
	table := render(results)
	fmt.Print(table)
	if *record {
		f, err := os.OpenFile(filepath.Join(root, "bench", "results.md"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		must(err)
		commit, _ := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD").Output()
		fmt.Fprintf(f, "\n## %s — %s, %s/%s (HEAD %s, plus the working tree)\n\n%s", time.Now().Format("2006-01-02 15:04"), runtime.Version(), runtime.GOOS, runtime.GOARCH, strings.TrimSpace(string(commit)), table)
		must(f.Close())
	}
	if failed {
		os.Exit(1)
	}
}

func render(rs []result) string {
	var b strings.Builder
	b.WriteString("| benchmark | ops | Veles | Go | Veles / Go |\n|---|---:|---:|---:|---:|\n")
	for _, r := range rs {
		ratio := float64(r.veles) / float64(r.goTime)
		fmt.Fprintf(&b, "| %s | %d | %s | %s | %.1f× |\n", r.name, r.ops, round(r.veles), round(r.goTime), ratio)
	}
	return b.String()
}

func round(d time.Duration) time.Duration {
	if d > time.Second {
		return d.Round(time.Millisecond)
	}
	return d.Round(10 * time.Microsecond)
}

func parse(name string, out []byte) result {
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 5 && f[0] == "BENCH" && f[1] == name {
			ops, _ := strconv.ParseInt(f[2], 10, 64)
			ns, _ := strconv.ParseInt(f[3], 10, 64)
			return result{name: name, ops: ops, veles: time.Duration(ns), checksum: f[4]}
		}
	}
	fmt.Fprintf(os.Stderr, "bench: %s printed no BENCH line:\n%s", name, out)
	os.Exit(1)
	return result{}
}

// ---------------------------------------------------------------------------
// Go references: the same work as bench/<name>/main.vs, the same checksum.

var references = map[string]func() string{
	"sha256": func() string {
		data := make([]byte, 1048576)
		for i := range data {
			data[i] = byte(i * 31 % 256)
		}
		var last [32]byte
		for i := 0; i < 16; i++ {
			last = sha256.Sum256(data)
		}
		return hex.EncodeToString(last[:])[:16]
	},
	"maps": func() string {
		var sum int64
		for r := 0; r < 5; r++ {
			m := map[int64]int64{}
			for k := int64(0); k < 200000; k++ {
				m[k*2654435761%1000000007] = k
			}
			for k := int64(0); k < 200000; k++ {
				sum += m[k*2654435761%1000000007]
			}
		}
		return strconv.FormatInt(sum, 10)
	},
	"sort": func() string {
		x := uint64(88172645463325252)
		input := make([]int64, 0, 300000)
		for i := 0; i < 300000; i++ {
			x ^= x << 13
			x ^= x >> 7
			x ^= x << 17
			input = append(input, int64(x%1000000))
		}
		var check int64
		for r := 0; r < 3; r++ {
			s := slices.Clone(input)
			slices.Sort(s)
			check += s[0] + s[150000] + s[299999]
		}
		return strconv.FormatInt(check, 10)
	},
	"json": func() string {
		type Record struct {
			ID    int64    `json:"id"`
			Name  string   `json:"name"`
			Score float64  `json:"score"`
			Tags  []string `json:"tags"`
		}
		input := make([]Record, 2000)
		for i := range input {
			input[i] = Record{ID: int64(i), Name: "record number " + strconv.Itoa(i), Score: float64(i) * 0.5, Tags: []string{"a", "bb", "ccc"}}
		}
		var check int64
		for r := 0; r < 20; r++ {
			text, err := json.Marshal(input)
			must(err)
			var back []Record
			must(json.Unmarshal(text, &back))
			for _, rec := range back {
				check += rec.ID
			}
		}
		return strconv.FormatInt(check, 10)
	},
	"strings": func() string {
		var check int64
		for r := 0; r < 5; r++ {
			var sb strings.Builder
			for i := 0; i < 200000; i++ {
				sb.WriteString("item-")
				sb.WriteString(strconv.Itoa(i))
				sb.WriteString(",")
			}
			parts := strings.Split(sb.String(), ",")
			check += int64(len(parts) + len(parts[123456]))
		}
		return strconv.FormatInt(check, 10)
	},
	"trees": func() string {
		type node struct{ left, right *node }
		var build func(d int) *node
		build = func(d int) *node {
			if d == 0 {
				return &node{}
			}
			return &node{build(d - 1), build(d - 1)}
		}
		var count func(n *node) int64
		count = func(n *node) int64 {
			if n.left == nil || n.right == nil {
				return 1
			}
			return 1 + count(n.left) + count(n.right)
		}
		var check int64
		for d := 4; d <= 16; d += 2 {
			for t := 0; t < 1<<(16-d+4); t++ {
				check += count(build(d))
			}
		}
		return strconv.FormatInt(check, 10)
	},
	"channels": func() string {
		ch := make(chan int64, 64)
		go func() {
			for i := int64(0); i < 200000; i++ {
				ch <- i
			}
			close(ch)
		}()
		var sum int64
		for v := range ch {
			sum += v
		}
		return strconv.FormatInt(sum, 10)
	},
	"spawn": func() string {
		square := func(i int64) int64 {
			var acc int64
			for k := int64(0); k < 50; k++ {
				acc += (i + k) % 7
			}
			return acc
		}
		var sum int64
		for b := int64(0); b < 100; b++ {
			res := make([]int64, 1000)
			var wg sync.WaitGroup
			for j := 0; j < 1000; j++ {
				wg.Add(1)
				go func(j int) {
					defer wg.Done()
					res[j] = square(b*1000 + int64(j))
				}(j)
			}
			wg.Wait()
			for _, r := range res {
				sum += r
			}
		}
		return strconv.FormatInt(sum, 10)
	},
	"parallel": func() string {
		work := func(seed int64) int64 {
			var total int64
			words := map[string]int64{}
			for round := int64(0); round < 200; round++ {
				var xs []int64
				for i := int64(0); i < 200; i++ {
					xs = append(xs, (seed*31+i*7+round)%1000)
				}
				sorted := slices.Clone(xs)
				slices.Sort(sorted)
				total += sorted[100]
				words["k"+strconv.FormatInt((seed+round)%17, 10)]++
			}
			return total + int64(len(words))
		}
		out := make(chan int64, 64)
		for s := int64(0); s < 64; s++ {
			go func(s int64) { out <- work(s) }(s)
		}
		var sum int64
		for i := 0; i < 64; i++ {
			sum += <-out
		}
		return strconv.FormatInt(sum, 10)
	},
}

// ---------------------------------------------------------------------------

func moduleRoot() string {
	dir, err := os.Getwd()
	must(err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintln(os.Stderr, "bench: run it inside the repository")
			os.Exit(2)
		}
		dir = parent
	}
}

func run(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "bench: %s %s: %v\n%s", name, strings.Join(args, " "), err, out)
		os.Exit(1)
	}
}

func exe() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		os.Exit(1)
	}
}
