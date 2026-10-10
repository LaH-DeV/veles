package driver

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// D128 / plan E1: std/tls against a real TLS stack — Go's crypto/tls serving
// from this test — on whatever backend the platform has (SChannel on Windows,
// OpenSSL elsewhere). One Veles program dials servers that behave in turn:
// a good certificate, a certificate for another name, one no root vouches
// for, a connection cut without close_notify, a protocol too old, an
// ALPN choice, a large echo in both directions, and the escape hatch.

type tlsAuthority struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  string
}

func newAuthority(t *testing.T) *tlsAuthority {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Veles test authority"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	return &tlsAuthority{cert, key, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))}
}

// leaf issues a server certificate valid for the given names and addresses.
func (a *tlsAuthority) leaf(t *testing.T, dns []string, ips []net.IP) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "veles test server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dns,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, a.cert, &key.PublicKey, a.key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// tlsServer accepts on a loopback port and runs behave on every connection
// that completes the handshake; it returns the port.
func tlsServer(t *testing.T, cfg *tls.Config, behave func(raw net.Conn, c *tls.Conn)) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			raw, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer raw.Close()
				c := tls.Server(raw, cfg)
				raw.SetDeadline(time.Now().Add(2 * time.Minute)) // generous: a loaded machine runs the 1 MB echo slowly
				if err := c.Handshake(); err != nil {
					return
				}
				behave(raw, c)
			}()
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

const tlsProgram = `use http, io, tls

const ca = "@CA@"

// dials, sends a line, and returns the first line of the answer
fun dial(port: i64, options: tls.Options, host: string): string suspends throws IoError | io.TooLong {
  with conn = try tls.connect(host, port, options)
  try conn.writeText("hi\n")
  val line = try conn.readLine(100) ?: "(end)"
  val after = try conn.read()
  "$line, then ${after.len()} bytes, alpn ${conn.protocol()}"
}

// closes as soon as the answer is in, before the server ends its side: only
// the close can tell the server the connection ended cleanly
fun answerOnly(port: i64, options: tls.Options): string suspends {
  quick(port, options) catch (e) { "error ${e.message()}" }
}

fun quick(port: i64, options: tls.Options): string suspends throws IoError | io.TooLong {
  with conn = try tls.connect("127.0.0.1", port, options)
  try conn.writeText("hi\n")
  try conn.readLine(100) ?: "(end)"
}

fun hello(port: i64, options: tls.Options, host: string = "127.0.0.1"): string suspends {
  dial(port, options, host) catch (e) { "error ${e.message()}" }
}

// reads to the end while another task writes: the server echoes as it reads, so
// a client that wrote everything before reading would stall both sides
fun drain(conn: tls.Conn): string suspends throws IoError {
  var got = 0
  var sum = 0
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    got += chunk.len()
    loop (b in chunk) {
      sum += b.toI64()
    }
  }
  "got $got, sum $sum"
}

fun echoed(port: i64): string suspends throws IoError {
  with conn = try tls.connect("127.0.0.1", port, tls.Options(roots: ca))
  with reader = async drain(conn)
  val block: MutableList<u8> = []
  loop (i in 0..<65536) {
    block.push((i * 7 % 251).wrapU8())
  }
  val data = block.toList()
  loop (_ in 0..<16) {
    try conn.write(data)
  }
  try conn.shutdownWrite()
  await reader
}

// two requests through one client: the second reuses the pooled TLS connection
fun web(port: i64, options: tls.Options): string suspends {
  with client = http.Client(tlsOptions: options)
  val first = client.get("https://127.0.0.1:$port/hello") catch (e) { return "error ${e.message()}" }
  val a = first.text() catch (e) { return "error ${e.message()}" }
  val second = client.get("https://127.0.0.1:$port/again") catch (e) { return "error ${e.message()}" }
  val b = second.text() catch (e) { return "error ${e.message()}" }
  "${first.status.code} $a / ${second.status.code} $b"
}

fun echo(port: i64): string suspends => echoed(port) catch (e) { "error ${e.message()}" }

fun main() suspends {
  io.println("good: ${hello(@GOOD@, tls.Options(roots: ca))}")
}
`

func TestTLSClient(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	auth := newAuthority(t)
	good := auth.leaf(t, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})
	other := auth.leaf(t, []string{"other.example"}, nil)
	rogue := newAuthority(t).leaf(t, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})

	answer := func(raw net.Conn, c *tls.Conn) {
		bufio.NewReader(c).ReadString('\n')
		io.WriteString(c, "hello over tls\n")
		c.Close() // close_notify
	}
	goodPort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{good}}, answer)
	otherPort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{other}}, answer)
	roguePort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{rogue}}, answer)
	tls12Port := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{good}, MaxVersion: tls.VersionTLS12}, answer)
	oldPort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{good}, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11}, answer)
	alpnPort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{good}, NextProtos: []string{"h2", "http/1.1"}}, answer)
	truncPort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{good}}, func(raw net.Conn, c *tls.Conn) {
		bufio.NewReader(c).ReadString('\n')
		io.WriteString(c, "hel")
		raw.Close() // no close_notify
	})
	echoPort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{good}}, func(raw net.Conn, c *tls.Conn) {
		io.Copy(c, c)
		c.Close()
	})
	// the client's close sends close_notify (D147 follow-up, as Go's Close
	// does). Go's reader ends cleanly at a record boundary either way, so
	// this server answers and then reads the socket itself: the client sends
	// nothing more, so whatever arrives before the end is the alert record
	ended := make(chan string, 1)
	notifyPort := tlsServer(t, &tls.Config{Certificates: []tls.Certificate{good}}, func(raw net.Conn, c *tls.Conn) {
		bufio.NewReader(c).ReadString('\n')
		io.WriteString(c, "hello over tls\n")
		rest, _ := io.ReadAll(raw)
		if len(rest) > 0 {
			ended <- "close_notify"
		} else {
			ended <- "the socket was closed with no alert"
		}
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "path %s", r.URL.Path) })
	wl, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { wl.Close() })
	webConns := &countingListener{Listener: wl}
	go http.Serve(tls.NewListener(webConns, &tls.Config{Certificates: []tls.Certificate{good}}), mux)
	webPort := wl.Addr().(*net.TCPAddr).Port

	var sum int64
	for i := 0; i < 16; i++ {
		for j := 0; j < 65536; j++ {
			sum += int64(j * 7 % 251)
		}
	}

	src := strings.ReplaceAll(tlsProgram, "@CA@", strings.ReplaceAll(auth.pem, "\n", "\\n"))
	src = strings.Replace(src, `  io.println("good: ${hello(@GOOD@, tls.Options(roots: ca))}")`, fmt.Sprintf(`  val trusting = tls.Options(roots: ca)
  io.println("good: ${hello(%[1]d, trusting)}")
  io.println("name: ${hello(%[1]d, tls.Options(roots: ca, serverName: "localhost"))}")
  io.println("wrong name: ${hello(%[2]d, trusting)}")
  io.println("unknown authority: ${hello(%[3]d, trusting)}")
  io.println("system roots: ${hello(%[1]d, tls.Options())}")
  io.println("accept any: ${hello(%[3]d, tls.Options(dangerouslyAcceptAnyCertificate: true))}")
  io.println("tls 1.2: ${hello(%[4]d, trusting)}")
  io.println("tls 1.1: ${hello(%[5]d, trusting)}")
  io.println("alpn: ${hello(%[6]d, tls.Options(roots: ca, alpn: ["http/1.1"]))}")
  io.println("truncated: ${hello(%[7]d, trusting)}")
  io.println("echo: ${echo(%[8]d)}")
  io.println("refused: ${hello(1, trusting)}")
  io.println("https: ${web(%[9]d, trusting)}")
  io.println("https untrusted: ${web(%[9]d, tls.Options())}")
  io.println("notify: ${answerOnly(%[10]d, trusting)}")
`, goodPort, otherPort, roguePort, tls12Port, oldPort, alpnPort, truncPort, echoPort, webPort, notifyPort), 1)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "tlsclient.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	out := runExe(t, exe)
	got := map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if name, rest, ok := strings.Cut(line, ": "); ok {
			got[name] = rest
		}
	}
	want := map[string]string{
		"good":              "hello over tls, then 0 bytes, alpn null",
		"name":              "hello over tls, then 0 bytes, alpn null",
		"wrong name":        "error tls: certificate verification failed",
		"unknown authority": "error tls: certificate verification failed",
		"system roots":      "error tls: certificate verification failed",
		"accept any":        "hello over tls, then 0 bytes, alpn null",
		"tls 1.2":           "hello over tls, then 0 bytes, alpn null",
		"tls 1.1":           "error tls:",
		"alpn":              "hello over tls, then 0 bytes, alpn http/1.1",
		"truncated":         "error tls: the connection was closed without a close_notify",
		"echo":              fmt.Sprintf("got %d, sum %d", 16*65536, sum),
		"refused":           "error",
		"https":             "200 path /hello / 200 path /again",
		"https untrusted":   "error",
		"notify":            "hello over tls",
	}
	for name, prefix := range want {
		if g, ok := got[name]; !ok || !strings.HasPrefix(g, prefix) {
			t.Errorf("%s: got %q, want it to start with %q", name, g, prefix)
		}
	}

	select {
	case how := <-ended:
		if how != "close_notify" {
			t.Errorf("the client's close ended the connection with %q, want a close_notify first", how)
		}
	case <-time.After(10 * time.Second):
		t.Errorf("the server never saw the client's connection end")
	}
	if n := webConns.count.Load(); n != 2 { // the pooled connection of the first client, then the untrusted attempt's
		t.Errorf("the https server accepted %d connections, want 2 (one pooled and reused)", n)
	}
	if t.Failed() {
		t.Logf("program output:\n%s", out)
	}
}

// runExe runs a built program and returns its standard output.
func runExe(t *testing.T, exe string) string {
	t.Helper()
	cmd := exec.Command(exe)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s: %v\nstdout:\n%s\nstderr:\n%s", exe, err, out, stderr.String())
	}
	return string(out)
}

// countingListener counts the connections it accepts.
type countingListener struct {
	net.Listener
	count atomic.Int64
}

func (l *countingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		l.count.Add(1)
	}
	return c, err
}
