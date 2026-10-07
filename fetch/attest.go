package fetch

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Signed reviews (D139). A statement says, about the exact bytes of one
// package version, that a reviewer makes a claim:
//
//	attest v1 <git|registry> <source> <version|commit:full> <h1 hash> <claim> <time> <ed25519:key> <sig>
//
// The signature is ed25519 over `veles-attest-v1 <kind> <source> <ref> <hash>
// <claim> <time>`. It verifies on its own, so the statement can be kept
// anywhere: served by a registry, committed under `attestations/`, or
// emailed. Who is believed is the consumer's choice (`[policy] trust`).

const (
	keyPrefix    = "ed25519:"
	attestTag    = "attest"
	attestFormat = "v1"
)

var claimRE = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Statement is one signed review.
type Statement struct {
	Kind   string // "git" or "registry"
	Source string
	Ref    string // a version, or commit:<full>
	Hash   string // the h1 hash of the package's files
	Claim  string
	Time   string // RFC 3339, UTC
	Key    string // ed25519:<base64 public key>
	Sig    string // base64
}

func (s Statement) payload() string {
	return strings.Join([]string{"veles-attest-v1", s.Kind, s.Source, s.Ref, s.Hash, s.Claim, s.Time}, " ")
}

// String is the statement's line.
func (s Statement) String() string {
	return strings.Join([]string{attestTag, attestFormat, s.Kind, s.Source, s.Ref, s.Hash, s.Claim, s.Time, s.Key, s.Sig}, " ")
}

// Subject names the package version a statement is about, as veles.sum does.
func (s Statement) Subject() string { return s.Kind + " " + s.Source + " " + s.Ref }

// Sign makes the statement's signature with the private key and fills Key.
func (s Statement) Sign(priv ed25519.PrivateKey) Statement {
	s.Key = EncodePublic(priv.Public().(ed25519.PublicKey))
	s.Sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(s.payload())))
	return s
}

// Verify checks the shape of the statement and its signature.
func (s Statement) Verify() error {
	if s.Kind != "git" && s.Kind != "registry" {
		return fmt.Errorf("a statement is about a git or registry package, not %q", s.Kind)
	}
	// a statement is one line of space-separated fields: a field with a space in
	// it could not be read back, and would sign something other than it says
	for _, field := range []string{s.Source, s.Ref, s.Hash, s.Claim, s.Time, s.Key, s.Sig} {
		if field == "" || strings.ContainsAny(field, " \t\r\n") {
			return fmt.Errorf("every field of a statement is present and has no whitespace (%q)", field)
		}
	}
	if !strings.HasPrefix(s.Hash, "h1:") {
		return fmt.Errorf("the hash %q is not an h1: hash", s.Hash)
	}
	if !claimRE.MatchString(s.Claim) {
		return fmt.Errorf("the claim %q is a lowercase word", s.Claim)
	}
	pub, err := DecodePublic(s.Key)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(s.Sig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("the signature is not a base64 ed25519 signature")
	}
	if !ed25519.Verify(pub, []byte(s.payload()), sig) {
		return fmt.Errorf("the signature does not match the statement: it was changed after it was signed, or made by another key")
	}
	return nil
}

// ParseStatement reads one line and verifies it.
func ParseStatement(line string) (Statement, error) {
	f := strings.Fields(line)
	if len(f) != 10 || f[0] != attestTag || f[1] != attestFormat {
		return Statement{}, fmt.Errorf("a statement is 'attest v1 <git|registry> <source> <ref> <hash> <claim> <time> <key> <signature>'")
	}
	s := Statement{Kind: f[2], Source: f[3], Ref: f[4], Hash: f[5], Claim: f[6], Time: f[7], Key: f[8], Sig: f[9]}
	return s, s.Verify()
}

// ParseStatements reads a file of statements (comments and blank lines are
// skipped). A bad line is an error naming it; the good ones are returned.
func ParseStatements(name string, data []byte) ([]Statement, []error) {
	var out []Statement
	var errs []error
	for n, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		s, err := ParseStatement(line)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s:%d: %v", name, n+1, err))
			continue
		}
		out = append(out, s)
	}
	return out, errs
}

// attestationsDir is where a project keeps statements: beside its veles.sum.
func attestationsDir(root string) string {
	return filepath.Join(filepath.Dir(sumPathFor(root)), "attestations")
}

// LoadAttestations reads every `attestations/*.attest` of the project. A line
// that does not verify is an error — a project should not hold a statement
// that vouches for nothing.
func LoadAttestations(root string) ([]Statement, error) {
	dir := attestationsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Statement
	var bad []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".attest") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		ss, errs := ParseStatements(filepath.Join("attestations", e.Name()), data)
		out = append(out, ss...)
		for _, e := range errs {
			bad = append(bad, e.Error())
		}
	}
	if len(bad) > 0 {
		return out, fmt.Errorf("%s", strings.Join(bad, "\n"))
	}
	return out, nil
}

// AddAttestations appends statements to attestations/<file>.attest, skipping
// ones already held, and returns how many were new.
func AddAttestations(root, file string, statements []Statement) (int, error) {
	dir := attestationsDir(root)
	have := map[string]bool{}
	if existing, err := LoadAttestations(root); err == nil {
		for _, s := range existing {
			have[s.String()] = true
		}
	}
	var lines []string
	for _, s := range statements {
		if !have[s.String()] {
			have[s.String()] = true
			lines = append(lines, s.String())
		}
	}
	if len(lines) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	path := filepath.Join(dir, file+".attest")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l + "\n"); err != nil {
			return 0, err
		}
	}
	return len(lines), nil
}

// TrustedReviewers counts, for a package version, the distinct trusted keys
// that made each claim about exactly these bytes. trust maps a reviewer's
// name to its key; the result maps a claim to the names that made it.
func TrustedReviewers(statements []Statement, trust map[string]string, kind, source, ref, hash string) map[string][]string {
	names := map[string]string{} // key -> reviewer name
	for who, key := range trust {
		names[key] = who
	}
	out := map[string][]string{}
	seen := map[string]bool{}
	for _, s := range statements {
		who, ok := names[s.Key]
		if !ok || s.Kind != kind || s.Source != source || s.Ref != ref || s.Hash != hash || seen[s.Claim+" "+s.Key] {
			continue
		}
		seen[s.Claim+" "+s.Key] = true
		out[s.Claim] = append(out[s.Claim], who)
	}
	for _, who := range out {
		sort.Strings(who)
	}
	return out
}

// --- keys ------------------------------------------------------------------

// EncodePublic writes a public key as ed25519:<base64>.
func EncodePublic(pub ed25519.PublicKey) string {
	return keyPrefix + base64.StdEncoding.EncodeToString(pub)
}

// DecodePublic reads ed25519:<base64>.
func DecodePublic(s string) (ed25519.PublicKey, error) {
	if !strings.HasPrefix(s, keyPrefix) {
		return nil, fmt.Errorf("a public key is written ed25519:<base64>, not %q", s)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, keyPrefix))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%q is not an ed25519 public key (32 bytes, base64)", s)
	}
	return ed25519.PublicKey(raw), nil
}

func keyDir() string { return filepath.Join(Home(), "keys") }

var keyNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// GenerateKey makes a key pair, stores the private key in
// ~/.veles/keys/<name>.key (readable by its owner only) and returns the
// public key as it is written in a policy.
func GenerateKey(name string) (string, string, error) {
	if !keyNameRE.MatchString(name) {
		return "", "", fmt.Errorf("a key name is letters, digits, '-' and '_'")
	}
	path := filepath.Join(keyDir(), name+".key")
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(keyDir(), 0o700); err != nil {
		return "", "", err
	}
	text := "veles-ed25519-seed " + base64.StdEncoding.EncodeToString(priv.Seed()) + "\n"
	// O_EXCL: creating the file is the check that it did not exist, so two
	// `keygen`s of one name cannot both succeed with one key replacing the other
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return "", "", fmt.Errorf("%s exists; keys are never overwritten (choose another name)", path)
		}
		return "", "", err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return "", "", err
	}
	if err := f.Close(); err != nil {
		return "", "", err
	}
	return EncodePublic(pub), path, nil
}

// LoadKey reads a private key by name (~/.veles/keys/<name>.key) or by path.
func LoadKey(nameOrPath string) (ed25519.PrivateKey, error) {
	path := nameOrPath
	if keyNameRE.MatchString(nameOrPath) {
		path = filepath.Join(keyDir(), nameOrPath+".key")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no key %q (looked for %s); `veles attest keygen %s` makes one", nameOrPath, path, nameOrPath)
		}
		return nil, err
	}
	f := strings.Fields(string(data))
	if len(f) != 2 || f[0] != "veles-ed25519-seed" {
		return nil, fmt.Errorf("%s is not a veles key file", path)
	}
	seed, err := base64.StdEncoding.DecodeString(f[1])
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("%s holds a damaged key", path)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}
