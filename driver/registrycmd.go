package driver

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LaH-DeV/veles/fetch"
	"github.com/LaH-DeV/veles/sema"
)

// The registry commands (D139): publish, yank and attest. They talk to the
// registry named by `[registry] url` (VELES_PROXY overrides it) with the
// token in VELES_TOKEN or ~/.veles/token.

// Publish is `veles publish [dir]`: check the package, archive it and upload
// the version named in its manifest.
func Publish(o PkgOptions) int {
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	if man.Version == "" || man.RegistryName == "" {
		return fail(fmt.Errorf("to publish, veles.toml needs [package] version and registry = \"owner/name\""))
	}
	// what is uploaded must at least be a package that checks
	if code := Run(Options{Path: root, Mode: "check"}); code != 0 {
		return fail(fmt.Errorf("%s does not check, so it was not published", man.Name))
	}
	name, version, err := fetch.Publish(root, man, fetch.RegistryURL(root, man), fetch.Token())
	if err != nil {
		return fail(err)
	}
	fmt.Printf("published %s %s (versions are immutable; to withdraw it, `veles yank %s@%s`)\n", name, version, name, version)
	return 0
}

// Yank is `veles yank <owner/name>@<version> [--reason text] [--undo]`.
func Yank(o PkgOptions) int {
	if len(o.Args) != 1 || !strings.Contains(o.Args[0], "@") {
		return fail(fmt.Errorf("usage: veles yank <owner/name>@<version> [--reason text] [--undo] [--dir package]"))
	}
	i := strings.LastIndex(o.Args[0], "@")
	name, version := o.Args[0][:i], o.Args[0][i+1:]
	// the registry comes from the package's manifest when there is one; the
	// environment alone is enough to yank from anywhere
	proxy := os.Getenv("VELES_PROXY")
	if root, man, err := pkgRoot(o.Dir, false); err == nil {
		proxy = fetch.RegistryURL(root, man)
	} else if proxy == "" {
		return fail(err)
	}
	if err := fetch.Yank(proxy, name, version, o.Reason, fetch.Token(), o.Undo); err != nil {
		return fail(err)
	}
	if o.Undo {
		fmt.Printf("%s %s is no longer yanked\n", name, version)
	} else {
		fmt.Printf("yanked %s %s: new resolutions will not pick it; builds that already hold it in veles.sum go on with a warning\n", name, version)
	}
	return 0
}

// Attest is `veles attest keygen|sign|verify|import`.
func Attest(o PkgOptions) int {
	if len(o.Args) == 0 {
		return fail(fmt.Errorf("usage: veles attest keygen <name> | sign <package> [--claim reviewed] [--key name] [--push] | verify | import <file|url>"))
	}
	sub, args := o.Args[0], o.Args[1:]
	switch sub {
	case "keygen":
		if len(args) != 1 {
			return fail(fmt.Errorf("usage: veles attest keygen <name>"))
		}
		pub, path, err := fetch.GenerateKey(args[0])
		if err != nil {
			return fail(err)
		}
		fmt.Printf("wrote the private key to %s (keep it secret)\nthe public key, for [policy] trust in a project that believes your reviews:\n  %s = %q\n", path, args[0], pub)
		return 0
	case "sign":
		return attestSign(o, args)
	case "verify":
		return attestVerify(o)
	case "import":
		return attestImport(o, args)
	}
	return fail(fmt.Errorf("unknown attest command %q (keygen, sign, verify, import)", sub))
}

// resolved is the build of the package at o.Dir, for the attest commands.
func resolved(o PkgOptions) (string, *sema.Manifest, []fetch.PackageReport, error) {
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return "", nil, nil, err
	}
	res, err := fetch.ResolveWith(root, man, fetch.ResolveOptions{SkipPolicy: true})
	if err != nil {
		return "", nil, nil, err
	}
	report, err := res.Report()
	return root, man, report, err
}

func attestSign(o PkgOptions, args []string) int {
	if len(args) != 1 {
		return fail(fmt.Errorf("usage: veles attest sign <package> [--claim reviewed] [--key name] [--push]  (a local dependency name, a source, or a package name)"))
	}
	keyName := o.Key
	if keyName == "" {
		return fail(fmt.Errorf("say whose review this is: --key <name> (`veles attest keygen <name>` makes a key)"))
	}
	priv, err := fetch.LoadKey(keyName)
	if err != nil {
		return fail(err)
	}
	root, man, report, err := resolved(o)
	if err != nil {
		return fail(err)
	}
	var found *fetch.PackageReport
	for i, p := range report {
		if p.Remote() && (p.Local == args[0] || p.Source == args[0] || p.Name == args[0]) {
			if found != nil {
				return fail(fmt.Errorf("%q names more than one package in the build; use its source", args[0]))
			}
			found = &report[i]
		}
	}
	if found == nil {
		return fail(fmt.Errorf("no registry or git package called %q is in the build of %s (`veles deps` lists them)", args[0], man.Name))
	}
	claim := o.Claim
	if claim == "" {
		claim = "reviewed"
	}
	st := fetch.Statement{Kind: found.Kind, Source: found.Source, Ref: found.Ref, Hash: found.Hash, Claim: claim, Time: time.Now().UTC().Format(time.RFC3339)}.Sign(priv)
	if err := st.Verify(); err != nil {
		return fail(err)
	}
	if _, err := fetch.AddAttestations(root, keyName, []fetch.Statement{st}); err != nil {
		return fail(err)
	}
	fmt.Printf("signed %q for %s %s (%s) with key %s; kept in attestations/%s.attest\n", claim, found.Source, found.Version, found.Hash, keyName, keyName)
	if o.Push {
		if err := fetch.PushStatement(fetch.RegistryURL(root, man), st, fetch.Token()); err != nil {
			return fail(err)
		}
		fmt.Println("pushed to the registry")
	}
	return 0
}

func attestVerify(o PkgOptions) int {
	root, man, report, err := resolved(o)
	if err != nil {
		return fail(err)
	}
	stmts, loadErr := fetch.LoadAttestations(root)
	if loadErr != nil {
		fmt.Fprintln(os.Stderr, loadErr)
	}
	trust := map[string]string{}
	for who, key := range man.Policy.Trust {
		trust[key] = who
	}
	bad := loadErr != nil
	for _, s := range stmts {
		who := trust[s.Key]
		if who == "" {
			who = "an untrusted key"
		}
		state := "not in this build"
		for _, p := range report {
			if p.Kind == s.Kind && p.Source == s.Source {
				switch {
				case p.Ref == s.Ref && p.Hash == s.Hash:
					state = "ok"
				default:
					state = "stale: the build has " + p.Ref + " " + p.Hash
				}
			}
		}
		if strings.HasPrefix(state, "stale") {
			bad = true
		}
		fmt.Printf("%-6s %s %s %q by %s (%s)\n", map[bool]string{true: "FAIL", false: "ok"}[strings.HasPrefix(state, "stale")], s.Source, s.Ref, s.Claim, who, state)
	}
	if len(stmts) == 0 && loadErr == nil {
		fmt.Println("no attestations in attestations/")
	}
	if bad {
		return 1
	}
	return 0
}

func attestImport(o PkgOptions, args []string) int {
	if len(args) != 1 {
		return fail(fmt.Errorf("usage: veles attest import <file | https://url>"))
	}
	root, _, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	var data []byte
	if strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://") {
		resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(args[0])
		if err != nil {
			return fail(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fail(fmt.Errorf("%s answered %s", args[0], resp.Status))
		}
		if data, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20)); err != nil {
			return fail(err)
		}
	} else if data, err = os.ReadFile(args[0]); err != nil {
		return fail(err)
	}
	good, errs := fetch.ParseStatements(args[0], data)
	for _, e := range errs {
		fmt.Fprintln(os.Stderr, "not imported:", e)
	}
	file := strings.TrimSuffix(filepath.Base(args[0]), ".attest")
	if file == "" || strings.ContainsAny(file, `/\:?&=`) || strings.HasPrefix(args[0], "http") {
		file = "imported"
	}
	n, err := fetch.AddAttestations(root, file, good)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("imported %d new statement(s) into attestations/%s.attest (%d already held or skipped)\n", n, file, len(good)-n+len(errs))
	if len(errs) > 0 {
		return 1
	}
	return 0
}
