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
// A benchmark may also print `MEMORY <name> <bytes>` — bytes per operation
// (bench/idle: per parked task) — and its reference sets goMemory to the
// same measure; those get a table of their own.
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type result struct {
	name     string
	ops      int64
	veles    time.Duration
	goTime   time.Duration
	checksum string
	memory   int64 // bytes per operation the Veles side reported; 0 = none
	goMemory int64
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
		goSetup = 0
		goMemory = 0
		start := time.Now()
		sum := ref()
		r.goTime = time.Since(start) - goSetup
		r.goMemory = goMemory
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
	head := false
	for _, r := range rs {
		if r.memory == 0 || r.goMemory == 0 {
			continue
		}
		if !head {
			b.WriteString("\n| benchmark | Veles bytes/op | Go bytes/op | Veles / Go |\n|---|---:|---:|---:|\n")
			head = true
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %.2f× |\n", r.name, r.memory, r.goMemory, float64(r.memory)/float64(r.goMemory))
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
	var r result
	found := false
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 5 && f[0] == "BENCH" && f[1] == name {
			ops, _ := strconv.ParseInt(f[2], 10, 64)
			ns, _ := strconv.ParseInt(f[3], 10, 64)
			r = result{name: name, ops: ops, veles: time.Duration(ns), checksum: f[4], memory: r.memory}
			found = true
		}
		if len(f) == 3 && f[0] == "MEMORY" && f[1] == name {
			r.memory, _ = strconv.ParseInt(f[2], 10, 64)
		}
	}
	if found {
		return r
	}
	fmt.Fprintf(os.Stderr, "bench: %s printed no BENCH line:\n%s", name, out)
	os.Exit(1)
	return result{}
}

// ---------------------------------------------------------------------------
// Go references: the same work as bench/<name>/main.vs, the same checksum.

// goSetup is the time a reference spent on work the Veles side does before
// its clock starts (building the input text); the runner subtracts it.
var goSetup time.Duration

// goMemory is what a reference measured that reports memory: bytes per
// operation, as its Veles side prints them on its MEMORY line.
var goMemory int64

var references = map[string]func() string{
	"gzip": func() string {
		begin := time.Now()
		data := gzipText(2097152)
		var packed bytes.Buffer
		start := time.Now()
		w, _ := gzip.NewWriterLevel(&packed, 6)
		w.Write(data)
		w.Close()
		timed := time.Since(start)
		r, _ := gzip.NewReader(&packed)
		back, _ := io.ReadAll(r)
		// the text and the check on the way back are not the work measured
		goSetup = time.Since(begin) - timed
		return checksumBytes(back)
	},
	"gunzip": func() string {
		begin := time.Now()
		data := gzipText(2097152)
		var packed bytes.Buffer
		w, _ := gzip.NewWriterLevel(&packed, 6)
		w.Write(data)
		w.Close()
		start := time.Now()
		var back []byte
		for i := 0; i < 8; i++ {
			r, _ := gzip.NewReader(bytes.NewReader(packed.Bytes()))
			back, _ = io.ReadAll(r)
		}
		timed := time.Since(start)
		goSetup = time.Since(begin) - timed
		return checksumBytes(back)
	},
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
	"adapters": func() string {
		// the same loops the eager adapters run, each element through a
		// closure as Veles calls its lambda
		x := uint64(88172645463325252)
		input := make([]int64, 0, 200000)
		for i := 0; i < 200000; i++ {
			x ^= x << 13
			x ^= x >> 7
			x ^= x << 17
			input = append(input, int64(x%1000000))
		}
		mapF := func(xs []int64, f func(int64) int64) []int64 {
			out := make([]int64, 0, len(xs))
			for _, v := range xs {
				out = append(out, f(v))
			}
			return out
		}
		filterF := func(xs []int64, f func(int64) bool) []int64 {
			var out []int64
			for _, v := range xs {
				if f(v) {
					out = append(out, v)
				}
			}
			return out
		}
		var check int64
		for round := int64(0); round < 20; round++ {
			scaled := mapF(input, func(n int64) int64 { return n*3 + round })
			even := filterF(scaled, func(n int64) bool { return n%2 == 0 })
			var acc int64
			for _, n := range even {
				acc += n % 1000
			}
			check += acc
			for _, n := range even {
				if n > 2999990 {
					check++
					break
				}
			}
			all := true
			for _, n := range scaled {
				if n < 0 {
					all = false
					break
				}
			}
			if all {
				check += 2
			}
			found := int64(-1)
			for _, n := range even {
				if n%7 == 3 {
					found = n
					break
				}
			}
			check += found
		}
		return strconv.FormatInt(check, 10)
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
	"httphello": func() string {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		must(err)
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, "hello, world")
		})}
		go srv.Serve(ln)
		defer srv.Close()
		url := "http://" + ln.Addr().String() + "/"
		sums := make([]int64, 64)
		var wg sync.WaitGroup
		for c := 0; c < 64; c++ {
			wg.Add(1)
			go func(c int) {
				defer wg.Done()
				tr := &http.Transport{MaxIdleConnsPerHost: 1}
				defer tr.CloseIdleConnections()
				client := &http.Client{Transport: tr}
				for i := 0; i < 250; i++ {
					res, err := client.Get(url)
					must(err)
					body, err := io.ReadAll(res.Body)
					must(err)
					res.Body.Close()
					sums[c] += int64(len(body))
				}
			}(c)
		}
		wg.Wait()
		var total int64
		for _, s := range sums {
			total += s
		}
		return strconv.FormatInt(total, 10)
	},
	"pipes": func() string {
		sums := make([]int64, 8)
		var wg sync.WaitGroup
		for p := 0; p < 8; p++ {
			ch := make(chan int64, 64)
			go func() {
				for i := int64(0); i < 200000; i++ {
					ch <- i
				}
				close(ch)
			}()
			wg.Add(1)
			go func(p int) {
				defer wg.Done()
				for v := range ch {
					sums[p] += v
				}
			}(p)
		}
		wg.Wait()
		var total int64
		for _, s := range sums {
			total += s
		}
		return strconv.FormatInt(total, 10)
	},
	"suscall": func() string {
		var sum int64
		for i := int64(0); i < 10000000; i++ {
			sum += suscallTop(i)
		}
		return strconv.FormatInt(sum, 10)
	},
	"idle": func() string {
		// 100k goroutines parked on one channel: the heap and the stacks
		// they add, per goroutine, after a collection
		const n = 100000
		var ms runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&ms)
		before := ms.HeapInuse + ms.StackInuse
		ch := make(chan int64)
		results := make(chan int64, n)
		var parked atomic.Int64
		for i := 0; i < n; i++ {
			go func() {
				parked.Add(1)
				results <- <-ch
			}()
		}
		for parked.Load() < n {
			runtime.Gosched()
		}
		runtime.GC()
		runtime.ReadMemStats(&ms)
		goMemory = int64(ms.HeapInuse+ms.StackInuse-before) / n
		var sum int64
		for i := int64(0); i < n; i++ {
			ch <- i
		}
		for i := 0; i < n; i++ {
			sum += <-results
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

var gzipWords = []string{"the", "of", "and", "stream", "request", "value", "error", "handler", "buffer", "index", "result", "compile", "token", "branch", "memory", "thread", "socket", "length", "window", "symbol", "a", "to", "in", "is"}

// gzipText is bench/gzip's input: words picked by a xorshift generator
func gzipText(size int) []byte {
	out := make([]byte, 0, size+16)
	x := uint64(88172645463325252)
	for len(out) < size {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		out = append(out, gzipWords[x%24]...)
		if x%11 == 0 {
			out = append(out, 10)
		} else {
			out = append(out, 32)
		}
	}
	return out
}

func checksumBytes(b []byte) string {
	var sum int64
	for _, c := range b {
		sum = (sum*31 + int64(c)) % 1000000007
	}
	return strconv.FormatInt(sum, 10)
}

// the suscall reference: three ordinary functions, as Go writes them
func suscallLeaf(i int64) int64 {
	if i < 0 {
		time.Sleep(time.Millisecond)
	}
	return i % 7
}

func suscallMiddle(i int64) int64 { return suscallLeaf(i) + 1 }

func suscallTop(i int64) int64 { return suscallMiddle(i) + 1 }
