package fetch

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// Spec is what `veles add` is given: where a dependency comes from, and the
// version if the command line said one.
type Spec struct {
	Kind    string // "path", "git" or "registry"
	Source  string // a directory, a repository URL, or owner/name
	Version string // "" when none was given
	Name    string // the local name the manifest will use
}

var registryNameRE = regexp.MustCompile(`^[a-z0-9_-]+/[a-z0-9_-]+$`)

// ParseSpec reads the argument of `veles add`:
//
//	../mathlib                          a directory
//	https://github.com/veles-db/pg[@2.1.0]   a git repository (so does github.com/veles-db/pg, git@host:x/y)
//	acme/httputil[@1.4.2]               a registry package
func ParseSpec(arg string) (Spec, error) {
	spec := Spec{}
	if arg == "" {
		return spec, fmt.Errorf("say what to add: a path, a git URL or owner/name")
	}
	if i := strings.LastIndex(arg, "@"); i > 0 && i > strings.LastIndex(arg, "/") && i > strings.LastIndex(arg, ":") {
		spec.Version = arg[i+1:]
		arg = arg[:i]
		v, err := sema.ParseVersion(spec.Version)
		if err != nil {
			return spec, err
		}
		spec.Version = v.String()
	}
	switch {
	case strings.HasPrefix(arg, ".") || strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "~") || strings.ContainsRune(arg, '\\') || (len(arg) > 1 && arg[1] == ':'):
		if spec.Version != "" {
			return spec, fmt.Errorf("a path has no version: it is the directory as it is")
		}
		spec.Kind, spec.Source = "path", arg
	case strings.Contains(arg, "://") || strings.HasPrefix(arg, "git@") || strings.HasSuffix(arg, ".git"):
		spec.Kind, spec.Source = "git", normaliseURL(arg)
	case strings.Contains(arg, "/") && strings.Contains(strings.SplitN(arg, "/", 2)[0], "."):
		spec.Kind, spec.Source = "git", normaliseURL("https://"+arg)
	case registryNameRE.MatchString(arg):
		spec.Kind, spec.Source = "registry", arg
	default:
		return spec, fmt.Errorf("%q is not a path, a git URL or a registry name (owner/name, lowercase)", arg)
	}
	if spec.Kind == "git" {
		if err := checkGitURL(spec.Source, false); err != nil {
			return spec, err
		}
	}
	spec.Name = localNameOf(spec.Source)
	return spec, nil
}

// localNameOf is the name code will write in `use`: the last part of the
// source, made an identifier.
func localNameOf(source string) string {
	base := path.Base(strings.ReplaceAll(strings.TrimRight(source, "/\\"), "\\", "/"))
	if i := strings.LastIndex(base, ":"); i >= 0 {
		base = base[i+1:]
	}
	base = strings.TrimSuffix(base, ".git")
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_':
			b.WriteRune(r)
		case r == '-' || r == '.':
			b.WriteByte('_')
		}
	}
	name := b.String()
	if name != "" && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}
	return name
}

var tagRE = regexp.MustCompile(`refs/tags/(v?[0-9][^/\s]*)$`)

// Versions lists the tagged versions of a git repository or a registry
// package, in no particular order. A tag that is not a version is ignored.
func (f *Fetcher) Versions(kind, source string) ([]sema.Version, error) {
	if f.Offline {
		return nil, fmt.Errorf("cannot list the versions of %s: fetching is off", source)
	}
	var names []string
	switch kind {
	case "git":
		out, err := git("", "ls-remote", "--tags", "--refs", "--", source)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(out, "\n") {
			if m := tagRE.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				names = append(names, m[1])
			}
		}
	case "registry":
		if f.Proxy == "" {
			return nil, fmt.Errorf("registry package '%s' needs a proxy: set VELES_PROXY (the registry service is not live yet), or add its git repository instead", source)
		}
		url := strings.TrimRight(f.Proxy, "/") + "/" + source + "/@v/list"
		resp, err := f.client().Get(url)
		if err != nil {
			return nil, fmt.Errorf("cannot list the versions of %s: %v", source, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("cannot list the versions of %s: %s answered %s", source, url, resp.Status)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		names = strings.Fields(string(data))
	default:
		return nil, fmt.Errorf("a %s dependency has no versions", kind)
	}
	var out []sema.Version
	seen := map[string]bool{}
	for _, n := range names {
		if v, err := sema.ParseVersion(n); err == nil && !seen[v.String()] {
			seen[v.String()] = true
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Compare(out[j]) < 0 })
	return out, nil
}

// Latest is the highest stable version, within one major when major is not
// "" (the major as sema.Version.MajorID spells it). A pre-release is never
// picked on its own: it is used only when someone writes it.
func Latest(vs []sema.Version, major string) (sema.Version, bool) {
	var best sema.Version
	found := false
	for _, v := range vs {
		if v.Pre != "" || (major != "" && v.MajorID() != major) {
			continue
		}
		if !found || v.Compare(best) > 0 {
			best, found = v, true
		}
	}
	return best, found
}

// LatestUsable is Latest that skips versions the registry has yanked (D139):
// a new resolution never picks one. A git tag cannot be yanked.
func (f *Fetcher) LatestUsable(kind, source string, vs []sema.Version, major string) (sema.Version, bool) {
	rest := append([]sema.Version{}, vs...)
	for {
		v, ok := Latest(rest, major)
		if !ok {
			return v, false
		}
		if kind != "registry" {
			return v, true
		}
		if info, err := f.info(source, v.String()); err != nil || !info.Yanked {
			return v, true // an unreadable answer is the fetch's to report
		}
		var next []sema.Version
		for _, r := range rest {
			if r.Compare(v) != 0 {
				next = append(next, r)
			}
		}
		rest = next
	}
}
