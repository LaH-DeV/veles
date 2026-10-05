package driver

import (
	"bufio"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
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
	"testing"
	"time"
)

// The server side of std/tls (D128): a Veles program serving TLS, and Go's
// crypto/tls as the clients — every private-key encoding a server is expected
// to read, ALPN, bulk transfer in both directions at once, the protocol floor,
// a silent client that must not stall the others, certificate reload, and a
// key that does not match its certificate.

const tlsServerProgram = `use io, os, time, tls

fun serveOne(conn: tls.Conn) suspends {
  with c = conn
  loop {
    val line = when (c.readLine(1000)) {
      is Ok(l)  => l ?: break
      is Err(_) => break
    }
    when (c.writeText("echo $line\n")) {
      is Ok(_)  => { }
      is Err(_) => break
    }
    if (line == "bye") break
  }
}

fun main() suspends throws IoError {
  val args = os.args()
  with certs = try tls.reloading(args.at(0) ?: "", args.at(1) ?: "", every: Duration.millis(100), alpn: ["h2", "http/1.1"])
  with listener = try tls.listen(cert: certs.certificate())
  io.println("port ${listener.port()}")
  scope {
    loop {
      val conn = try listener.accept()
      async serveOne(conn)
    }
  }
}
`

// keyKinds are the private-key encodings a server is expected to read.
var keyKinds = []string{"ecdsa-pkcs8", "ecdsa-sec1", "rsa-pkcs1", "rsa-pkcs8", "p384-pkcs8"}

func keyFor(t *testing.T, kind string) (crypto.Signer, []byte) {
	t.Helper()
	var key crypto.Signer
	var err error
	switch kind {
	case "rsa-pkcs1", "rsa-pkcs8":
		key, err = rsa.GenerateKey(rand.Reader, 2048)
	case "p384-pkcs8":
		key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	default:
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}
	if err != nil {
		t.Fatal(err)
	}
	var der []byte
	label := "PRIVATE KEY"
	switch kind {
	case "ecdsa-sec1":
		der, err = x509.MarshalECPrivateKey(key.(*ecdsa.PrivateKey))
		label = "EC PRIVATE KEY"
	case "rsa-pkcs1":
		der = x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey))
		label = "RSA PRIVATE KEY"
	default:
		der, err = x509.MarshalPKCS8PrivateKey(key)
	}
	if err != nil {
		t.Fatal(err)
	}
	return key, pem.EncodeToMemory(&pem.Block{Type: label, Bytes: der})
}

// issue makes a server certificate for key and returns it as PEM.
func (a *tlsAuthority) issue(t *testing.T, key crypto.Signer, serial int64) []byte {
	t.Helper()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "veles test server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, a.cert, key.Public(), a.key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// startTLSServer runs the program on the given files and waits for its port.
func startTLSServer(t *testing.T, exe, certPath, keyPath string) int {
	t.Helper()
	cmd := exec.Command(exe, certPath, keyPath)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	r := bufio.NewReader(out)
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("the server printed no port: %v\nstderr: %s", err, stderr.String())
	}
	go io.Copy(io.Discard, r)
	var port int
	if _, err := fmt.Sscanf(line, "port %d", &port); err != nil {
		t.Fatalf("unexpected first line %q", line)
	}
	return port
}

func dialVeles(t *testing.T, port int, cfg *tls.Config) *tls.Conn {
	t.Helper()
	d := &net.Dialer{Timeout: 10 * time.Second}
	c, err := tls.DialWithDialer(d, "tcp", fmt.Sprintf("127.0.0.1:%d", port), cfg)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c.SetDeadline(time.Now().Add(30 * time.Second))
	return c
}

func TestTLSServer(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(tlsServerProgram), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "tlsserver.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	auth := newAuthority(t)
	pool := x509.NewCertPool()
	pool.AddCert(auth.cert)
	cfg := func(mut func(*tls.Config)) *tls.Config {
		c := &tls.Config{RootCAs: pool, ServerName: "localhost"}
		if mut != nil {
			mut(c)
		}
		return c
	}

	for _, kind := range keyKinds {
		t.Run(kind, func(t *testing.T) {
			key, keyPEM := keyFor(t, kind)
			certPath, keyPath := filepath.Join(dir, kind+".pem"), filepath.Join(dir, kind+".key")
			os.WriteFile(certPath, auth.issue(t, key, 100), 0o644)
			os.WriteFile(keyPath, keyPEM, 0o600)
			port := startTLSServer(t, exe, certPath, keyPath)

			// a verified connection: ALPN, an echo, and the bulk in both directions at once
			c := dialVeles(t, port, cfg(func(c *tls.Config) { c.NextProtos = []string{"http/1.1", "h2"} }))
			defer c.Close()
			if got := c.ConnectionState().NegotiatedProtocol; got != "h2" { // the server's preference wins
				t.Errorf("alpn: got %q, want h2", got)
			}
			fmt.Fprintf(c, "ping\n")
			br := bufio.NewReader(c)
			if line, _ := br.ReadString('\n'); line != "echo ping\n" {
				t.Errorf("echo: got %q", line)
			}
			const lines = 20000
			go func() {
				w := bufio.NewWriter(c)
				for i := 0; i < lines; i++ {
					fmt.Fprintf(w, "line %05d %s\n", i, strings.Repeat("x", 40))
				}
				w.Flush()
			}()
			for i := 0; i < lines; i++ {
				line, err := br.ReadString('\n')
				want := fmt.Sprintf("echo line %05d %s\n", i, strings.Repeat("x", 40))
				if err != nil || line != want {
					t.Fatalf("bulk line %d: got %q, %v", i, line, err)
				}
			}
			fmt.Fprintf(c, "bye\n")
			if line, _ := br.ReadString('\n'); line != "echo bye\n" {
				t.Errorf("bye: got %q", line)
			}
			if _, err := br.ReadByte(); err != io.EOF {
				t.Errorf("after bye: %v, want a clean end of stream", err)
			}

			// TLS 1.2 works; anything older is refused
			c12 := dialVeles(t, port, cfg(func(c *tls.Config) { c.MaxVersion = tls.VersionTLS12 }))
			if got := c12.ConnectionState().Version; got != tls.VersionTLS12 {
				t.Errorf("version: got %x", got)
			}
			c12.Close()
			old := cfg(func(c *tls.Config) { c.MinVersion = tls.VersionTLS10; c.MaxVersion = tls.VersionTLS11 })
			if c11, err := tls.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port), old); err == nil {
				c11.Close()
				t.Errorf("a TLS 1.1 client was accepted")
			}

			// a client that connects and says nothing does not stall the others
			idle, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				t.Fatal(err)
			}
			defer idle.Close()
			ok := dialVeles(t, port, cfg(nil))
			fmt.Fprintf(ok, "still here\n")
			if line, _ := bufio.NewReader(ok).ReadString('\n'); line != "echo still here\n" {
				t.Errorf("with an idle client connected: got %q", line)
			}
			ok.Close()
		})
	}

	t.Run("reload", func(t *testing.T) {
		key, keyPEM := keyFor(t, "ecdsa-pkcs8")
		certPath, keyPath := filepath.Join(dir, "r.pem"), filepath.Join(dir, "r.key")
		os.WriteFile(certPath, auth.issue(t, key, 1), 0o644)
		os.WriteFile(keyPath, keyPEM, 0o600)
		port := startTLSServer(t, exe, certPath, keyPath)
		serial := func(c *tls.Conn) int64 { return c.ConnectionState().PeerCertificates[0].SerialNumber.Int64() }
		old := dialVeles(t, port, cfg(nil))
		defer old.Close()
		if got := serial(old); got != 1 {
			t.Fatalf("serial before the reload: %d", got)
		}
		// the key stays, the certificate is renewed
		os.WriteFile(certPath, auth.issue(t, key, 2), 0o644)
		deadline := time.Now().Add(20 * time.Second)
		for {
			c := dialVeles(t, port, cfg(nil))
			s := serial(c)
			c.Close()
			if s == 2 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("the renewed certificate was not picked up")
			}
			time.Sleep(100 * time.Millisecond)
		}
		// a half-written renewal is refused and the old one stays
		os.WriteFile(certPath, []byte("-----BEGIN CERTIFICATE-----\nbm90IGEgY2VydA==\n"), 0o644)
		time.Sleep(600 * time.Millisecond)
		c := dialVeles(t, port, cfg(nil))
		if got := serial(c); got != 2 {
			t.Errorf("after a bad renewal the serial is %d, want 2", got)
		}
		c.Close()
		// the connection opened before the reload still works
		fmt.Fprintf(old, "old\n")
		if line, _ := bufio.NewReader(old).ReadString('\n'); line != "echo old\n" {
			t.Errorf("the old connection: got %q", line)
		}
	})

	t.Run("a key that does not match is refused at the start", func(t *testing.T) {
		keyA, _ := keyFor(t, "ecdsa-pkcs8")
		_, keyB := keyFor(t, "ecdsa-pkcs8")
		certPath, keyPath := filepath.Join(dir, "m.pem"), filepath.Join(dir, "m.key")
		os.WriteFile(certPath, auth.issue(t, keyA, 1), 0o644)
		os.WriteFile(keyPath, keyB, 0o600)
		cmd := exec.Command(exe, certPath, keyPath)
		out, _ := cmd.CombinedOutput()
		if cmd.ProcessState.Success() || !strings.Contains(string(out), "tls:") {
			t.Errorf("a mismatched key: exit %v, output %q", cmd.ProcessState, out)
		}
	})
}

const httpsServerProgram = `use http, io, net, os, tls

fun main() suspends throws IoError {
  val args = os.args()
  with certs = try tls.reloading(args.at(0) ?: "", args.at(1) ?: "", every: Duration.seconds(60))
  with listener = try net.listen()
  io.println("port ${listener.port()}")
  val router = http.Router()
  router.get("/hi", req => http.Response.text("hello tls"))
  router.post("/echo", req => http.Response.text(try req.text()))
  http.serve(listener, router.handler(), log: false, tls: certs.certificate())
}
`

// http.serve(tls:) serves HTTPS: the handler sees plain requests, keep-alive
// and bodies work as over TCP, and a client that speaks no TLS is dropped.
func TestHTTPSServe(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(httpsServerProgram), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "https.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	auth := newAuthority(t)
	key, keyPEM := keyFor(t, "ecdsa-pkcs8")
	certPath, keyPath := filepath.Join(dir, "s.pem"), filepath.Join(dir, "s.key")
	os.WriteFile(certPath, auth.issue(t, key, 7), 0o644)
	os.WriteFile(keyPath, keyPEM, 0o600)
	port := startTLSServer(t, exe, certPath, keyPath)

	pool := x509.NewCertPool()
	pool.AddCert(auth.cert)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "localhost"}}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	base := fmt.Sprintf("https://127.0.0.1:%d", port)

	for i := 0; i < 3; i++ { // keep-alive: the same connection again
		resp, err := client.Get(base + "/hi")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "hello tls" || resp.TLS == nil {
			t.Fatalf("GET /hi: %q, tls %v", body, resp.TLS != nil)
		}
	}
	big := strings.Repeat("0123456789abcdef", 6000)
	resp, err := client.Post(base+"/echo", "text/plain", strings.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != big {
		t.Fatalf("echo: got %d bytes, want %d", len(body), len(big))
	}

	// plain HTTP to the TLS port is dropped, and the server carries on
	plain, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	plain.SetDeadline(time.Now().Add(10 * time.Second))
	fmt.Fprintf(plain, "GET /hi HTTP/1.1\r\nHost: x\r\n\r\n")
	if got, _ := io.ReadAll(plain); strings.Contains(string(got), "hello") {
		t.Errorf("a cleartext request was answered: %q", got)
	}
	plain.Close()
	resp, err = client.Get(base + "/hi")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
