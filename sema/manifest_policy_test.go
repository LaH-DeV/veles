package sema

import (
	"strings"
	"testing"
)

func TestManifestPolicy(t *testing.T) {
	m := parseOK(t, `
[package]
name = "app"

[policy]
deny = ["net", "unsafe", "native"]
allow = { net = ["acme/httputil", "https://example.com/pg"], unsafe = ["pg"] }
`)
	p := m.Policy
	if strings.Join(p.Deny, ",") != "net,unsafe,native" || strings.Join(p.Allow["net"], ",") != "acme/httputil,https://example.com/pg" || p.Allow["unsafe"][0] != "pg" {
		t.Errorf("policy: %+v", p)
	}
	if p.Empty() {
		t.Error("a policy with deny is not empty")
	}
	if !parseOK(t, "[package]\nname = \"app\"\n").Policy.Empty() {
		t.Error("no [policy] is an empty policy")
	}
}

func TestManifestPolicyErrors(t *testing.T) {
	pkg := "[package]\nname = \"p\"\n"
	for _, c := range []struct{ text, want string }{
		{pkg + "[policy]\ndeny = [\"network\"]\n", "[policy] deny: unknown capability \"network\" (unsafe, extern, native, net, fs, os, ffi, unlisted)"},
		{pkg + "[policy]\ndeny = \"net\"\n", "[policy] deny is a list"},
		{pkg + "[policy]\nallow = [\"net\"]\n", "[policy] allow is a table"},
		{pkg + "[policy]\nallow = { web = [\"a/b\"] }\n", "[policy] allow: \"web\" is neither a capability"},
		{pkg + "[policy]\nallow = { net = \"a/b\" }\n", "[policy] allow.net is a list"},
		{pkg + "[policy]\ndeni = []\n", "unknown [policy] key \"deni\" (deny, allow, trust, require)"},
		{pkg + "[policy]\ntrust = [\"x\"]\n", "[policy] trust is a table"},
		{pkg + "[policy]\ntrust = { a = \"abc\" }\n", "[policy] trust.a is a public key written ed25519:<base64>"},
		{pkg + "[policy]\nrequire = { reviewed = 1 }\n", "[policy] require needs trust"},
		{pkg + "[policy]\ntrust = { a = \"ed25519:AAAA\" }\nrequire = { reviewed = 0 }\n", "[policy] require.reviewed is how many"},
		{pkg + "[policy]\ntrust = { a = \"ed25519:AAAA\" }\nrequire = { reviewed = \"yes\" }\n", "[policy] require.reviewed is how many"},
		{pkg + "[registry]\nurl = \"ftp://x\"\n", "[registry] url is an http:// or https:// address"},
		{pkg + "[registry]\nkey = \"abc\"\n", "[registry] key is a public key written ed25519:<base64>"},
		{pkg + "[registry]\nhost = \"x\"\n", "unknown [registry] key \"host\" (url, key)"},
		{"[package]\nname = \"p\"\nregistry = \"Acme/X\"\n", "[package] registry \"Acme/X\" is owner/name"},
	} {
		if _, err := parseManifest("veles.toml", ".", c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q:\n got %v\nwant %q", c.text, err, c.want)
		}
	}
}

func TestManifestReviewPolicyAndRegistry(t *testing.T) {
	m := parseOK(t, `
[package]
name = "lib"
registry = "acme/lib"

[registry]
url = "https://registry.example.com"
key = "ed25519:AAAA"

[policy]
deny = ["unlisted"]
trust = { alice = "ed25519:AAAA", bob = "ed25519:BBBB" }
require = { reviewed = 2 }
allow = { reviewed = ["acme/old"], unlisted = ["acme/old"] }
`)
	if m.RegistryName != "acme/lib" || m.Registry.URL != "https://registry.example.com" || m.Registry.Key != "ed25519:AAAA" {
		t.Errorf("registry: %+v %q", m.Registry, m.RegistryName)
	}
	p := m.Policy
	if len(p.Trust) != 2 || p.Require["reviewed"] != 2 || p.Empty() {
		t.Errorf("policy: %+v", p)
	}
	// reviews: two are needed, an allow exempts, enough is enough
	for _, c := range []struct {
		have  int
		names []string
		want  bool
	}{
		{0, []string{"acme/x"}, true},
		{1, []string{"acme/x"}, true},
		{2, []string{"acme/x"}, false},
		{0, []string{"acme/old"}, false},
	} {
		if got := p.ReviewMissing("reviewed", c.have, c.names...); got != c.want {
			t.Errorf("have %d %v: %v", c.have, c.names, got)
		}
	}
	if p.ReviewMissing("audited", 0, "acme/x") {
		t.Error("a claim that is not required is never missing")
	}
	if !p.Denied("unlisted", "acme/x") || p.Denied("unlisted", "acme/old") {
		t.Error("unlisted: deny and allow")
	}
}
