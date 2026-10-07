package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// Home is the directory the module cache lives under: $VELES_HOME, else
// ~/.veles. Packages are kept in <home>/pkg, one directory per version, and
// never changed after they are written.
func Home() string {
	if h := os.Getenv("VELES_HOME"); h != "" {
		return h
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".veles")
	}
	return ".veles"
}

// Fetcher gets packages into the cache and checks them against a Sum.
type Fetcher struct {
	Sum *Sum
	// Proxy is the base URL registry packages come from (VELES_PROXY).
	Proxy string
	// RegistryKey is the pinned public key of the registry (`[registry] key`):
	// when set, a version's metadata must be signed with it (D139).
	RegistryKey string
	// Refresh asks the registry again about packages already in the cache
	// (their tier and yank); `veles fetch` and `veles audit` set it.
	Refresh bool
	// Warnings are what a fetch wants the user told (a yanked version that
	// veles.sum still holds).
	Warnings []string
	// Offline refuses to fetch: a package not already in the cache is an
	// error that names `veles fetch`. The language server runs this way.
	Offline bool
	HTTP    *http.Client
	home    string
	// vendor is the project's vendor directory when it has one: packages are
	// read from there and nowhere else (`veles vendor` writes it).
	vendor string
}

func (f *Fetcher) useVendor(dir string) {
	f.vendor = dir
	f.Offline = true
}

// NewFetcher is a Fetcher with the environment's settings.
func NewFetcher(sum *Sum) *Fetcher {
	return &Fetcher{Sum: sum, Proxy: os.Getenv("VELES_PROXY"), Offline: Offline, home: Home()}
}

// Offline makes every resolution refuse to fetch (set by the language
// server, which must not block on the network).
var Offline bool

// A package version a manifest asks for.
type pin struct {
	kind    string // "git" or "registry"
	source  string // the git URL, or owner/name
	version string // a normalised tag version ("" for a commit pin)
	commit  string // the hex prefix given ("" for a tag)
}

// normaliseURL makes the spellings of one repository one source: no trailing
// slash and no `.git`.
func normaliseURL(u string) string {
	u = strings.TrimRight(strings.TrimSpace(u), "/")
	return strings.TrimSuffix(u, ".git")
}

func pinOf(d sema.Dependency, fromPackage bool) (pin, error) {
	p := pin{kind: d.Kind(), commit: d.Commit}
	switch p.kind {
	case "registry":
		p.source = d.Registry
	case "git":
		if err := checkGitURL(d.Git, fromPackage); err != nil {
			return p, err
		}
		p.source = normaliseURL(d.Git)
	default:
		return p, fmt.Errorf("a %s dependency is not fetched", p.kind)
	}
	if d.Version != "" {
		v, err := sema.ParseVersion(d.Version)
		if err != nil {
			return p, err
		}
		p.version = v.String()
	}
	return p, nil
}

func (p pin) ref() string {
	if p.commit != "" {
		return "commit-" + strings.ToLower(p.commit)
	}
	return p.version
}

func sanitise(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "._")
}

// cacheDir is where a pin lives: the source made safe for a file name, a
// short hash of it (two sources that differ only in odd characters stay
// apart), then the version. A registry package's hash also covers the
// registry's address: `acme/lib` at a private registry and at a public one are
// different packages, and one must never be served from the other's cache
// entry (dependency confusion). vendor/ belongs to one project, which has one
// registry, so it leaves the address out.
func (f *Fetcher) cacheDir(p pin) string {
	key := p.kind + " " + p.source
	if p.kind == "registry" && f.vendor == "" {
		key += " @ " + strings.TrimRight(f.Proxy, "/")
	}
	sum := sha256.Sum256([]byte(key))
	name := sanitise(p.source)
	if len(name) > 60 {
		name = name[len(name)-60:]
	}
	root := filepath.Join(f.home, "pkg")
	if f.vendor != "" {
		root = f.vendor
	}
	return filepath.Join(root, p.kind, name+"-"+hex.EncodeToString(sum[:4])+"@"+p.ref())
}

// A fetched package.
type fetched struct {
	pin    pin
	dir    string
	hash   string
	commit string // the full commit of a git checkout
	man    *sema.Manifest
	// tier is "listed" or "unreviewed" (D139): what the registry said when
	// the version was fetched; a git package is always unreviewed.
	tier       string
	yanked     bool // as the registry said when it was fetched
	yankReason string
	known      bool // veles.sum already had a line for it before this resolution
	// signedBy is the registry key the metadata above was verified with; ""
	// when it was not (no key pinned), in which case a project that pins a key
	// does not believe the tier or the yank state in the cache.
	signedBy string
}

// sumKey is the package's line in veles.sum, without the hash.
func (fp *fetched) sumKey() string {
	ref := fp.pin.version
	if fp.pin.commit != "" {
		ref = "commit:" + fp.commit
	}
	return fp.pin.kind + " " + fp.pin.source + " " + ref
}

// get returns the package in the cache, fetching it when it is not there, and
// verifies it: the files must hash to what was recorded when they were
// fetched, and to what veles.sum says.
func (f *Fetcher) get(p pin) (*fetched, error) {
	dir := f.cacheDir(p)
	fp := &fetched{pin: p, dir: dir, tier: "unreviewed"}
	if f.Sum != nil && p.commit == "" {
		fp.known = f.Sum.Has(p.kind + " " + p.source + " " + p.version)
	}
	if _, err := os.Stat(dir); err == nil {
		marker, merr := readMarker(dir)
		hash, herr := TreeHash(dir)
		if merr != nil || herr != nil {
			return nil, fmt.Errorf("the cache entry %s is damaged (%v); delete the directory to fetch it again", dir, firstErr(merr, herr))
		}
		if hash != marker.hash {
			return nil, fmt.Errorf("the module cache entry %s was modified after it was fetched (it hashes to %s, it was %s); delete the directory to fetch it again", dir, hash, marker.hash)
		}
		fp.hash, fp.commit = hash, marker.commit
		if marker.tier != "" {
			fp.tier = marker.tier
		}
		fp.yanked, fp.yankReason, fp.signedBy = marker.yanked, marker.yankReason, marker.signedBy
		unverified := f.RegistryKey != "" && p.kind == "registry" && marker.signedBy != f.RegistryKey
		if unverified {
			fp.tier, fp.yanked, fp.yankReason = "unreviewed", false, "" // not believed until the registry says it again
		}
		// The cached copy remembers the registry's answer from the day it was
		// fetched. A new resolution (no veles.sum line), and the commands that
		// are about the network anyway, ask again; an unreachable registry
		// leaves the cached answer standing.
		if p.kind == "registry" && f.Proxy != "" && !f.Offline && f.vendor == "" && (!fp.known || f.Refresh || unverified) {
			if info, err := f.info(p.source, p.version); err == nil && info.Hash == hash {
				signed := ""
				if f.RegistryKey != "" {
					signed = info.Key // f.info verified it against the pinned key
				}
				if info.Tier != fp.tier || info.Yanked != fp.yanked || info.YankReason != fp.yankReason || signed != fp.signedBy {
					fp.tier, fp.yanked, fp.yankReason, fp.signedBy = info.Tier, info.Yanked, info.YankReason, signed
					writeMarker(dir, fp)
				}
			}
		}
	} else {
		if f.vendor != "" {
			return nil, fmt.Errorf("%s %s is not in the vendor directory %s; run `veles vendor` to refresh it, or delete vendor/ to use the module cache", p.source, orCommit(p), f.vendor)
		}
		if f.Offline {
			return nil, fmt.Errorf("%s %s is not in the module cache; run `veles fetch` to download it", p.source, orCommit(p))
		}
		if err := f.fetchInto(fp); err != nil {
			return nil, err
		}
	}
	if err := f.Sum.Check(fp.sumKey(), fp.hash); err != nil {
		return nil, err
	}
	if fp.yanked {
		// Cargo's rule: a build that already has the version in veles.sum goes
		// on, with a warning; a new resolution must choose another (D139)
		msg := fmt.Sprintf("%s %s was yanked from the registry: %s", p.source, p.version, orNone(fp.yankReason))
		if !fp.known {
			return nil, fmt.Errorf("%s; choose another version (`veles update %s` finds the newest that is not)", msg, localNameOf(p.source))
		}
		f.Warnings = append(f.Warnings, msg+" (veles.sum already holds it, so this build continues; move off it when you can)")
	}
	if _, err := os.Stat(filepath.Join(dir, "veles.toml")); err != nil {
		return nil, fmt.Errorf("%s %s has no veles.toml: a package is a directory with a manifest", p.source, orCommit(p))
	}
	man, err := sema.ReadManifestIn(dir)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %v", p.source, orCommit(p), err)
	}
	fp.man = man
	return fp, nil
}

func orCommit(p pin) string {
	if p.commit != "" {
		return "at commit " + p.commit
	}
	return p.version
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func orNone(s string) string {
	if s == "" {
		return "no reason given"
	}
	return s
}

type marker struct {
	hash, commit, tier string
	yanked             bool
	yankReason         string
	signedBy           string
}

func readMarker(dir string) (marker, error) {
	data, err := os.ReadFile(filepath.Join(dir, markerFile))
	if err != nil {
		return marker{}, err
	}
	var m marker
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "hash "):
			m.hash = strings.TrimPrefix(line, "hash ")
		case strings.HasPrefix(line, "commit "):
			m.commit = strings.TrimPrefix(line, "commit ")
		case strings.HasPrefix(line, "signed "):
			m.signedBy = strings.TrimPrefix(line, "signed ")
		case strings.HasPrefix(line, "tier "):
			m.tier = strings.TrimPrefix(line, "tier ")
		case strings.HasPrefix(line, "yanked "):
			m.yanked = true
			m.yankReason = strings.TrimPrefix(line, "yanked ")
		}
	}
	if m.hash == "" {
		return m, fmt.Errorf("no hash in %s", markerFile)
	}
	return m, nil
}

// writeMarker records what is known about a cached package: its hash, and
// what the registry or repository said when it was fetched.
func writeMarker(dir string, fp *fetched) error {
	text := "hash " + fp.hash + "\ntier " + fp.tier + "\n"
	if fp.commit != "" {
		text += "commit " + fp.commit + "\n"
	}
	if fp.yanked {
		text += "yanked " + strings.ReplaceAll(fp.yankReason, "\n", " ") + "\n"
	}
	if fp.signedBy != "" {
		text += "signed " + fp.signedBy + "\n"
	}
	return os.WriteFile(filepath.Join(dir, markerFile), []byte(text), 0o644)
}

// fetchInto downloads the package to a temporary directory inside the cache
// (so the final rename stays on one file system), hashes it, and moves it
// into place. A package that fails its checksum is never put in the cache.
func (f *Fetcher) fetchInto(fp *fetched) error {
	tmpRoot := filepath.Join(f.home, "tmp")
	if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
		return err
	}
	p := fp.pin
	var tmp, expected string // expected: the hash the registry says the archive has
	var err error
	switch p.kind {
	case "git":
		var tags []string
		if p.commit == "" {
			tags = []string{"v" + p.version, p.version}
		}
		tmp, fp.commit, err = gitCheckout(tmpRoot, p.source, tags, p.commit)
	default:
		var info *Info
		if info, err = f.info(p.source, p.version); err != nil {
			return err
		}
		fp.tier, fp.yanked, fp.yankReason = info.Tier, info.Yanked, info.YankReason
		if f.RegistryKey != "" {
			fp.signedBy = info.Key
		}
		if tmp, err = f.proxyCheckout(tmpRoot, p.source, p.version); err != nil {
			return err
		}
		expected = info.Hash
	}
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			removeAll(tmp)
		}
	}()
	if fp.hash, err = TreeHash(tmp); err != nil {
		return fmt.Errorf("%s %s: %v", p.source, orCommit(p), err)
	}
	if expected != "" && fp.hash != expected {
		return fmt.Errorf("%s %s: the archive hashes to %s but the registry's metadata says %s; the registry serves two different packages for one version, and nothing was built from either", p.source, p.version, fp.hash, expected)
	}
	if err = f.Sum.Check(fp.sumKey(), fp.hash); err != nil {
		return err
	}
	if err = writeMarker(tmp, fp); err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(fp.dir), 0o755); err != nil {
		return err
	}
	if err = os.Rename(tmp, fp.dir); err != nil {
		if _, statErr := os.Stat(fp.dir); statErr == nil {
			return nil // another build put the same version there first
		}
		return err
	}
	keep = true
	return nil
}
