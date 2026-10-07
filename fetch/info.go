package fetch

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Info is a registry's metadata about one version (D139), served as JSON at
// `<base>/<owner>/<name>/@v/<version>.info`. The signature covers the fields
// that decide what a client does — the hash, the tier, the yank — as one
// line of text, so there is no JSON canonicalisation to get wrong.
type Info struct {
	Version    string `json:"version"`
	Time       string `json:"time"`
	Hash       string `json:"hash"`
	Tier       string `json:"tier"`
	Yanked     bool   `json:"yanked"`
	YankReason string `json:"yankReason,omitempty"`
	Key        string `json:"key,omitempty"`
	Sig        string `json:"sig,omitempty"`
}

// Payload is the text the registry signs.
func (i Info) Payload(name string) string {
	yanked := "0"
	if i.Yanked {
		yanked = "1"
	}
	return strings.Join([]string{"veles-info-v1", name, i.Version, i.Hash, i.Tier, yanked, i.Time}, " ")
}

// SignInfo fills Key and Sig.
func SignInfo(priv ed25519.PrivateKey, name string, i Info) Info {
	i.Key = EncodePublic(priv.Public().(ed25519.PublicKey))
	i.Sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(i.Payload(name))))
	return i
}

// VerifyInfo checks the signature against a pinned key.
func VerifyInfo(pinned, name string, i Info) error {
	if i.Key != pinned {
		return fmt.Errorf("the metadata is signed with %s, not the key this project pins (%s)", orNone2(i.Key, "no key"), pinned)
	}
	pub, err := DecodePublic(pinned)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(i.Sig)
	if err != nil || !ed25519.Verify(pub, []byte(i.Payload(name)), sig) {
		return fmt.Errorf("the metadata's signature does not verify")
	}
	return nil
}

func orNone2(s, none string) string {
	if s == "" {
		return none
	}
	return s
}

// httpGet reads a small response body.
func (f *Fetcher) httpGet(url string) ([]byte, error) {
	resp, err := f.client().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// info fetches and checks the metadata of a registry version.
func (f *Fetcher) info(name, version string) (*Info, error) {
	if f.Proxy == "" {
		return nil, fmt.Errorf("registry package '%s' needs a proxy: the registry service is not live yet, so set VELES_PROXY (or [registry] url) to a registry that serves it, or depend on its git repository instead", name)
	}
	url := strings.TrimRight(f.Proxy, "/") + "/" + name + "/@v/" + version + ".info"
	data, err := f.httpGet(url)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch the metadata of %s %s: %v", name, version, err)
	}
	var i Info
	if err := json.Unmarshal(data, &i); err != nil {
		return nil, fmt.Errorf("the metadata of %s %s is not JSON: %v", name, version, err)
	}
	if i.Version != version || !strings.HasPrefix(i.Hash, "h1:") || (i.Tier != "listed" && i.Tier != "unreviewed") {
		return nil, fmt.Errorf("the metadata of %s %s is malformed (version %q, hash %q, tier %q)", name, version, i.Version, i.Hash, i.Tier)
	}
	if f.RegistryKey != "" {
		if err := VerifyInfo(f.RegistryKey, name, i); err != nil {
			return nil, fmt.Errorf("%s %s: %v; nothing was fetched", name, version, err)
		}
	}
	return &i, nil
}

// registryAttestations reads the statements a registry holds about a version.
func (f *Fetcher) registryAttestations(name, version string) ([]Statement, error) {
	if f.Proxy == "" || f.Offline {
		return nil, nil
	}
	data, err := f.httpGet(strings.TrimRight(f.Proxy, "/") + "/" + name + "/@v/" + version + ".attest")
	if err != nil {
		return nil, err
	}
	good, _ := ParseStatements(name+" "+version, data) // a statement that does not verify counts for nothing
	return good, nil
}
