package driver

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// std/db against a real PostgreSQL: the tests of std/db that need a server
// run when VELES_TEST_PG_URL is set, and this test sets it. It starts a
// throwaway cluster from the PostgreSQL binaries it finds — the directory in
// VELES_PG_BIN, or initdb on the PATH — with a trust login, a SCRAM role and
// TLS with a certificate of its own; without binaries it is skipped (std/db's
// own tests then pass without a server, which proves little).
func TestStdDbAgainstPostgres(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	bin := os.Getenv("VELES_PG_BIN")
	find := func(name string) string {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if bin != "" {
			p := filepath.Join(bin, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return ""
		}
		p, err := exec.LookPath(name)
		if err != nil {
			return ""
		}
		return p
	}
	initdb, pgctl, postgres := find("initdb"), find("pg_ctl"), find("postgres")
	if initdb == "" || pgctl == "" || postgres == "" {
		t.Skip("PostgreSQL binaries not found (set VELES_PG_BIN to the directory with initdb, pg_ctl and postgres)")
	}

	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s %v: %v\n%s", filepath.Base(name), args, err, out.String())
		}
	}
	run(initdb, "-D", data, "-U", "postgres", "-A", "trust", "-E", "UTF8", "--locale=C")

	// TLS: a certificate for 127.0.0.1 and localhost, signed by itself
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "veles test postgres"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:         true, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalPKCS8PrivateKey(key)
	crt := filepath.Join(dir, "server.crt")
	os.WriteFile(crt, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)
	os.WriteFile(filepath.Join(dir, "server.key"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	// the SCRAM role logs in over TCP with a password; everything else is trusted
	hba := "host all scram_user 127.0.0.1/32 scram-sha-256\nhost all all 127.0.0.1/32 trust\n"
	if err := os.WriteFile(filepath.Join(data, "pg_hba.conf"), []byte(hba), 0o600); err != nil {
		t.Fatal(err)
	}
	conf, _ := os.OpenFile(filepath.Join(data, "postgresql.conf"), os.O_APPEND|os.O_WRONLY, 0o600)
	fmt.Fprintf(conf, "\nlisten_addresses = '127.0.0.1'\nport = %d\nssl = on\nssl_cert_file = '%s'\nssl_key_file = '%s'\nmax_connections = 60\nfsync = off\n",
		port, filepath.ToSlash(crt), filepath.ToSlash(filepath.Join(dir, "server.key")))
	conf.Close()

	single := exec.Command(postgres, "--single", "-D", data, "postgres")
	single.Stdin = strings.NewReader("create role scram_user login password 'p@ss:w/rd'\n")
	if out, err := single.CombinedOutput(); err != nil {
		t.Fatalf("postgres --single: %v\n%s", err, out)
	}

	// the server inherits pg_ctl's output handles, so they must be files: a pipe would stay open for as long as it runs
	logFile, err := os.Create(filepath.Join(dir, "pg_ctl.out"))
	if err != nil {
		t.Fatal(err)
	}
	start := exec.Command(pgctl, "-D", data, "-l", filepath.Join(dir, "pg.log"), "-w", "start")
	start.Stdout, start.Stderr = logFile, logFile
	if err := start.Run(); err != nil {
		out, _ := os.ReadFile(filepath.Join(dir, "pg_ctl.out"))
		t.Fatalf("pg_ctl start: %v\n%s", err, out)
	}
	logFile.Close()
	t.Cleanup(func() { exec.Command(pgctl, "-D", data, "-m", "immediate", "stop").Run() })

	t.Setenv("VELES_TEST_PG_URL", fmt.Sprintf("postgres://postgres@127.0.0.1:%d/postgres", port))
	t.Setenv("VELES_TEST_PG_SCRAM_URL", fmt.Sprintf("postgres://scram_user:p%%40ss%%3Aw%%2Frd@127.0.0.1:%d/postgres", port))
	t.Setenv("VELES_TEST_PG_TLS_URL", fmt.Sprintf("postgres://postgres@127.0.0.1:%d/postgres", port))
	t.Setenv("VELES_TEST_PG_ROOT", crt)
	if strings.Contains(crt, "%") {
		t.Skip("a temp path with a % cannot go in the test URL")
	}
	if code := Run(Options{Path: filepath.Join("..", "std", "db"), Mode: "test"}); code != 0 {
		t.Fatalf("veles test std/db: exit %d", code)
	}
}
