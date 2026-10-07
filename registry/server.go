// Package registry is a reference implementation of the Veles registry
// protocol (D139) that keeps everything in memory. It exists to test the
// client end to end and to run a small private registry; the hosted service,
// with the real listing gates and registration by repository, is separate.
//
//	GET    /<owner>/<name>/@v/list           the versions, one per line
//	GET    /<owner>/<name>/@v/<v>.info       signed JSON metadata
//	GET    /<owner>/<name>/@v/<v>.zip        the package's files
//	GET    /<owner>/<name>/@v/<v>.attest     signed reviews, one per line
//	PUT    /<owner>/<name>/@v/<v>.zip        publish (token of that owner)
//	POST   /<owner>/<name>/@v/<v>.attest     add a review (any token)
//	PUT    /<owner>/<name>/@v/<v>.yank       yank, body = the reason (token of that owner)
//	DELETE /<owner>/<name>/@v/<v>.yank       undo
package registry

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LaH-DeV/veles/fetch"
	"github.com/LaH-DeV/veles/sema"
)

const maxUpload = 64 << 20

var nameRE = regexp.MustCompile(`^[a-z0-9_-]+$`)

// Server is the registry. The zero value is not usable; use New.
type Server struct {
	// Now is the clock; tests fix it.
	Now func() time.Time
	// Listed decides the tier of a published version, given its unpacked
	// files. The default lists version 1.0.0 or later that is not a
	// pre-release; the hosted service will also build on Windows and Linux,
	// run the package's tests and diff its API.
	Listed func(name string, v sema.Version, dir string) bool

	key    ed25519.PrivateKey
	tokens map[string]string // token -> owner

	mu   sync.Mutex
	pkgs map[string]map[string]*entry // name -> version -> entry
}

type entry struct {
	zip     []byte
	hash    string
	time    string
	tier    string
	yanked  bool
	reason  string
	attests []string
}

// New makes a registry that signs metadata with key and accepts the tokens
// (token -> owner).
func New(key ed25519.PrivateKey, tokens map[string]string) *Server {
	return &Server{
		Now:    time.Now,
		Listed: func(_ string, v sema.Version, _ string) bool { return v.Major >= 1 && v.Pre == "" },
		key:    key,
		tokens: tokens,
		pkgs:   map[string]map[string]*entry{},
	}
}

// PublicKey is the key a project pins in `[registry] key`.
func (s *Server) PublicKey() string {
	return fetch.EncodePublic(s.key.Public().(ed25519.PublicKey))
}

func fail(w http.ResponseWriter, code int, msg string) {
	http.Error(w, msg, code)
}

// ServeHTTP implements the protocol.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "@v" {
		fail(w, http.StatusNotFound, "not a registry path")
		return
	}
	owner, pkg, file := parts[0], parts[1], parts[3]
	if !nameRE.MatchString(owner) || !nameRE.MatchString(pkg) {
		fail(w, http.StatusBadRequest, "a package is owner/name: lowercase letters, digits, '-' and '_'")
		return
	}
	name := owner + "/" + pkg
	// the body is read before the lock is taken: a slow client must not hold
	// every other request up
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxUpload+1))
		if err != nil {
			fail(w, http.StatusBadRequest, "cannot read the request: "+err.Error())
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if file == "list" {
		if r.Method != http.MethodGet {
			fail(w, http.StatusMethodNotAllowed, "GET only")
			return
		}
		s.list(w, name)
		return
	}
	dot := strings.LastIndex(file, ".")
	if dot < 0 {
		fail(w, http.StatusNotFound, "unknown file")
		return
	}
	version, kind := file[:dot], file[dot+1:]
	switch r.Method + " " + kind {
	case "GET info":
		s.info(w, name, version)
	case "GET zip":
		if e := s.find(name, version); e == nil {
			fail(w, http.StatusNotFound, "no such version")
		} else {
			w.Header().Set("Content-Type", "application/zip")
			w.Write(e.zip)
		}
	case "GET attest":
		if e := s.find(name, version); e == nil {
			fail(w, http.StatusNotFound, "no such version")
		} else {
			w.Write([]byte(strings.Join(e.attests, "\n") + "\n"))
		}
	case "PUT zip":
		s.publish(w, r, owner, name, version)
	case "POST attest":
		s.attest(w, r, name, version)
	case "PUT yank", "DELETE yank":
		s.yank(w, r, owner, name, version)
	default:
		fail(w, http.StatusNotFound, "unknown call")
	}
}

func (s *Server) find(name, version string) *entry {
	return s.pkgs[name][version]
}

// tokenOwner is the owner the request's bearer token belongs to.
func (s *Server) tokenOwner(r *http.Request) (string, bool) {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	owner, ok := s.tokens[tok]
	return owner, ok && tok != ""
}

func (s *Server) list(w http.ResponseWriter, name string) {
	var vs []sema.Version
	for v := range s.pkgs[name] {
		if pv, err := sema.ParseVersion(v); err == nil {
			vs = append(vs, pv)
		}
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].Compare(vs[j]) < 0 })
	var b strings.Builder
	for _, v := range vs {
		b.WriteString(v.String() + "\n")
	}
	w.Write([]byte(b.String()))
}

func (s *Server) info(w http.ResponseWriter, name, version string) {
	e := s.find(name, version)
	if e == nil {
		fail(w, http.StatusNotFound, "no such version")
		return
	}
	i := fetch.SignInfo(s.key, name, fetch.Info{Version: version, Time: e.time, Hash: e.hash, Tier: e.tier, Yanked: e.yanked, YankReason: e.reason})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(i)
}

func (s *Server) publish(w http.ResponseWriter, r *http.Request, owner, name, version string) {
	who, ok := s.tokenOwner(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "a token is needed to publish")
		return
	}
	if who != owner {
		fail(w, http.StatusForbidden, "this token may publish below "+who+"/, not "+owner+"/")
		return
	}
	v, err := sema.ParseVersion(version)
	if err != nil || v.String() != version {
		fail(w, http.StatusBadRequest, "the version is major.minor.patch[-pre], without a leading v")
		return
	}
	if _, dup := s.pkgs[name][version]; dup {
		fail(w, http.StatusConflict, name+" "+version+" is published; versions are immutable (publish a new one)")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxUpload+1))
	if err != nil || len(body) > maxUpload {
		fail(w, http.StatusRequestEntityTooLarge, "the archive is too large")
		return
	}
	tmp, err := os.MkdirTemp("", "registry-")
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.RemoveAll(tmp)
	zp := tmp + string(os.PathSeparator) + "upload.zip"
	if err := os.WriteFile(zp, body, 0o600); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	dir, err := fetch.Unpack(tmp, zp)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	man, err := sema.ReadManifestIn(dir)
	if _, statErr := os.Stat(dir + string(os.PathSeparator) + "veles.toml"); statErr != nil || err != nil || man == nil || man.Name == "" {
		fail(w, http.StatusUnprocessableEntity, "the archive needs a veles.toml with a [package] name")
		return
	}
	if man.Version != "" && man.Version != version {
		fail(w, http.StatusUnprocessableEntity, "veles.toml says version "+man.Version+", the upload is "+version)
		return
	}
	hash, err := fetch.TreeHash(dir)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	tier := "unreviewed"
	if s.Listed(name, v, dir) {
		tier = "listed"
	}
	if s.pkgs[name] == nil {
		s.pkgs[name] = map[string]*entry{}
	}
	s.pkgs[name][version] = &entry{zip: body, hash: hash, time: s.Now().UTC().Format(time.RFC3339), tier: tier}
	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, hash+"\n")
}

func (s *Server) attest(w http.ResponseWriter, r *http.Request, name, version string) {
	if _, ok := s.tokenOwner(r); !ok {
		fail(w, http.StatusUnauthorized, "a token is needed to add a review")
		return
	}
	e := s.find(name, version)
	if e == nil {
		fail(w, http.StatusNotFound, "no such version")
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<10))
	st, err := fetch.ParseStatement(strings.TrimSpace(string(body)))
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if st.Kind != "registry" || st.Source != name || st.Ref != version || st.Hash != e.hash {
		fail(w, http.StatusUnprocessableEntity, "the statement is not about "+name+" "+version+" with hash "+e.hash)
		return
	}
	for _, have := range e.attests {
		if have == st.String() {
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	e.attests = append(e.attests, st.String())
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) yank(w http.ResponseWriter, r *http.Request, owner, name, version string) {
	who, ok := s.tokenOwner(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "a token is needed")
		return
	}
	if who != owner {
		fail(w, http.StatusForbidden, "this token may change only "+who+"/")
		return
	}
	e := s.find(name, version)
	if e == nil {
		fail(w, http.StatusNotFound, "no such version")
		return
	}
	if r.Method == http.MethodDelete {
		e.yanked, e.reason = false, ""
	} else {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 2<<10))
		e.yanked, e.reason = true, strings.TrimSpace(string(body))
	}
	w.WriteHeader(http.StatusOK)
}
