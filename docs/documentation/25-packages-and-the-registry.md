# 25. How packages and the registry work

[Chapter 11](11-modules-and-packages.md) shows how to *use* dependencies.
This chapter explains how the machinery works and why you can trust it: what
happens when you build, where every file lives, what is checked at each step,
what an attacker can and cannot do, and what to do when something is refused.
Read it once before you depend on code you did not write.

## The short version

- A **package** is a directory with a `veles.toml`. It can depend on other
  packages that come from a **path**, a **git repository** or a **registry**.
- Nothing runs when a dependency is fetched or built. There are no build
  scripts, no install hooks and no post-build steps. Fetching copies files;
  the only code that runs is yours, after it is compiled.
- Every package version you use is pinned by an **exact version** in
  `veles.toml` and by a **hash** in `veles.sum`. A package whose bytes do not
  match its hash is refused before anything is compiled.
- The compiler works out what a dependency can *do* — use `unsafe`, call C,
  reach the network, the file system or the process — from its source, and a
  `[policy]` can refuse what you do not want.
- A **registry** adds names (`owner/name`), immutable copies of every version,
  signed metadata, a "listed" quality tier, yanks and signed reviews. Git
  keeps working without one.

## The pieces

```text
  your project                                          the outside
  ┌──────────────────────────────┐
  │ veles.toml   what you ask for│      git repository ── tag v1.4.2 ──┐
  │ veles.sum    what it must be │                                     │
  │ vendor/      (optional) copy │      registry ── .info .zip .attest ┤
  │ attestations/ reviews you hold│                                    │
  └──────────────┬───────────────┘                                     │
                 │ resolve                                             │
                 ▼                                                     ▼
          ┌─────────────┐   fetch + verify    ┌───────────────────────────────┐
          │  resolver   │ ──────────────────► │ module cache  ~/.veles/pkg    │
          │ (fetch pkg) │ ◄────────────────── │ one directory per version     │
          └──────┬──────┘    hashes checked   └───────────────────────────────┘
                 │ a directory for each dependency
                 ▼
          ┌─────────────┐
          │  compiler   │  reads directories only; never touches the network
          └─────────────┘
```

The compiler itself knows nothing about git, registries or the network. A
separate part (`fetch`) resolves the dependency graph and hands the compiler a
directory for each dependency. The language server runs the same way but is
**offline**: it reads the cache and never downloads.

## What a build does with a dependency

For every `veles build`, `run`, `check` and `test`, in this order:

1. **Read the manifests.** Yours, then each dependency's, recursively. A
   path dependency is read where it is; a git or registry dependency is
   needed from the cache or the network.
2. **Select versions.** For each *(source, major)* the highest version any
   manifest asks for is chosen (see below). Nothing newer is ever picked.
3. **Fetch what the cache lacks.** A git tag or commit is checked out into a
   temporary directory with every line-ending and filter conversion turned
   off, `.git` is removed, and symbolic links are refused. A registry version
   is downloaded as a zip and unpacked with path, size and duplicate checks.
4. **Hash and compare.** The files are hashed (`h1:`, SHA-256 over each file's
   path and bytes) and compared with `veles.sum`. The first time a version is
   seen its hash is *recorded*; after that a different hash is a hard error.
   For a registry package the hash must also equal what the registry's signed
   metadata says.
5. **Apply the policy.** Capabilities are computed from the source of each
   registry and git package; `[policy]` can refuse them, and can require
   signed reviews.
6. **Hand directories to the compiler**, which type-checks and builds as
   usual.

If any step fails, **nothing is built and nothing is added to the cache or to
`veles.sum`**. A package that fails its checksum is deleted, not kept.

### Cached and offline

Every use of a cached package is re-hashed and compared with the hash recorded
when it was fetched, so an edited cache entry is a hard error too. With a warm
cache a build needs no network at all. `veles fetch` downloads everything and
records it (use it once in CI, then build offline); `veles vendor` copies the
build into the project (see below) so even the cache is not needed.

## Where a dependency comes from

```toml
[dependencies]
mathlib  = "../mathlib"                                       # a directory: your own code
pg       = { git = "https://github.com/veles-db/pg", version = "2.1.0" }   # a tag
fast     = { git = "https://example.com/fast", commit = "3f2a9c1" }        # a commit
httputil = { registry = "acme/httputil", version = "1.4.2" }               # a registry
```

- **A git version is a tag.** `version = "2.1.0"` fetches the tag `v2.1.0` (or
  `2.1.0`). Publishing a release is pushing a tag. A pre-release such as
  `2.2.0-beta.1` is used only when a manifest writes exactly that.
- **A commit pin** names an untagged revision. It cannot take part in version
  selection, so a package that others depend on gets a warning if it pins one.
- **A registry version** is an immutable copy the registry holds, with
  signed metadata (below).
- **A path** is your own code in your own repository: it is not hashed, not
  policed and not fetched.

### Identity, majors and selection

A package is identified by its **source** (the repository address, or
`owner/name`) and its **major version**. The major is the first non-zero part:
`1.4.2` is major 1, `0.3.1` is major 0.3, `0.0.2` is major 0.0.2.

- Two versions with the same source and major are **the same package**: only
  one is built, shared by everything that uses it.
- Two majors of one source are **two packages**; they may be used together
  under two names (`pg1`, `pg2`).
- The name in `[dependencies]` is only your local name for it. Only a
  package's *own* dependencies can be imported by it: if `a` uses `util`,
  your code cannot `use util` unless your manifest lists it too.

**Selection** is minimal version selection: what a manifest writes is a
*minimum*, and the highest minimum anyone asks for in a major is built.

```text
app    asks  util 1.0.0
mid    asks  util 1.3.0           →  util 1.3.0 is built, for app and mid alike
app    asks  util2 = util 2.0.0   →  util 2.0.0 is built as well (another major)
```

Nothing is upgraded unless somebody *wrote* the higher version. A build does
not change because a library published something; moving up is an edit
(`veles update`).

## The files, and which to commit

In your project:

| file or directory | what it is | commit? |
|---|---|---|
| `veles.toml` | what you ask for | yes |
| `veles.sum` | one line per package version: its hash | **yes** — it is what makes a build repeatable |
| `attestations/*.attest` | signed reviews you hold | yes |
| `vendor/` | a copy of every package in the build (`veles vendor`) | yes, if you use it |
| `bin/` | programs built from a workspace root | no |

On your machine, outside any project:

| path | what it is |
|---|---|
| `~/.veles/pkg/` (`$VELES_HOME/pkg/`) | the **module cache**: one directory per package version, whose files never change after they are written, each with a small `.veles-fetched` marker holding its hash and what the registry said |
| `~/.veles/tmp/` | checkouts in progress; emptied when a fetch ends or fails |
| `~/.veles/keys/<name>.key` | your reviewer signing keys (private; mode 0600) |
| `~/.veles/token` | your registry token (or set `VELES_TOKEN`) |

The module cache is shared by all your projects. A registry package is cached
**per registry address**, so `acme/lib` from your company's registry and
`acme/lib` from a public one never share an entry.

`veles.sum` lines look like this:

```text
git https://github.com/veles-db/pg 2.1.0 h1:3b6f…
git https://example.com/fast commit:3f2a9c1d… h1:9a01…
registry acme/httputil 1.4.2 h1:77ce…
```

It sits beside the root manifest of a workspace (shared by its members), else
beside your own. If you ever must accept different contents for a version —
you trust that a tag was moved on purpose — delete that one line (and the
version's directory in `~/.veles/pkg`, if it is there) and the next build
records the new hash. Look at what changed in the dependency first.

### Vendoring

`veles vendor` copies every version the build mentions into `vendor/`. While
`vendor/` exists the build reads packages from there **and nowhere else** —
no network, no module cache — and still checks each against `veles.sum`. Use
it for repeatable builds in places without network, or to review dependency
code in a pull request. Delete `vendor/` to go back to the cache. Do not
reformat vendored files (`veles fmt` skips `vendor/`): their bytes are hashed.

## The trust model

What each defence is for, and what it does not cover:

| threat | what stops it | where |
|---|---|---|
| a download is altered in transit or on the host | the hash in `veles.sum`; for a registry also the signed metadata | every fetch |
| a maintainer moves a tag to new code | `veles.sum` still has the old hash: *checksum mismatch*, nothing built | every fetch |
| the registry serves different bytes than it promised | the archive must hash to the signed `info.hash` | registry fetch |
| the registry's host is hijacked | metadata signed with a key you **pinned** in `[registry] key` | registry fetch |
| a package is deleted upstream | the registry keeps every version; `vendor/` keeps your own copy | — |
| a malicious version is published | exact versions, no automatic upgrades; yank; reviews | resolution |
| a dependency runs code while it is installed | there are no install or build scripts | — |
| a dependency uses `unsafe`, C, the network, files or processes | capabilities computed from its source; `[policy] deny` | resolution |
| a dependency pulls in something worse | policy covers every package in the build, not just the ones you named | resolution |
| a name is taken over by a look-alike (dependency confusion) | names are `owner/name`; the cache is per registry address; a fetched package's git address must be a network address | resolution |
| a package's manifest makes git run a command or read your disk | git addresses are validated: no options, no `ext::` helpers, no local paths from fetched packages; git only speaks file/http/https/ssh/git | fetch |
| an archive writes outside its directory | absolute paths, `..`, `:`, `\`, `.git` parts, links and case-twins are refused | unpack |
| a registry token is stolen on the wire | tokens are sent only over `https://` (or to localhost) | publish, yank |
| a review is forged or reused for other code | statements are signed and name the exact hash | review check |
| the editor downloads something by surprise | the language server is offline | — |

What is **not** protected, so you know:

- The **first** fetch of a version from a registry whose key you have not
  pinned is trusted. Pin the key (`[registry] key`) to close this.
- A registry that is **compromised and signs lies** is not detected (there is
  no transparency log yet), and an *old* signed `.info` can be replayed.
- A brand-new malicious package that does **only** what its capabilities
  allow is not caught by tools: capabilities say what code *can* reach, not
  what it *does*. Reviews are how people vouch for the latter.
- Capabilities are read from `use` lines and `unsafe`/`extern` syntax. They
  are conservative about imports, but they cannot see a dependency's *own*
  dependencies' behaviour — which is why the policy applies to every package
  in the build, and why `veles audit` lists them all.
- Code you pull in at a **path** is yours and is trusted completely.
- Version selection trusts the versions written in manifests: a library that
  asks for a vulnerable version of something makes you build it. `veles deps`
  shows the whole tree.

## A registry from the inside

A registry stores, for every `owner/name` and version:

- the **archive** — a zip of the package's files, immutable once published;
- its **hash**, computed by the registry (`h1:` as above);
- a **tier**: `listed` or `unreviewed`;
- a **yank** flag and reason;
- the **signed reviews** people have submitted.

It answers four read requests and accepts four write requests (the full
protocol is in [the registry protocol](reference/registry-protocol.md)):

```text
GET  …/@v/list          the versions
GET  …/@v/1.4.2.info    {"version","time","hash","tier","yanked","yankReason","key","sig"}
GET  …/@v/1.4.2.zip     the files
GET  …/@v/1.4.2.attest  signed reviews
PUT  …/@v/1.4.2.zip     publish          (token of that owner)
POST …/@v/1.4.2.attest  add a review     (any token)
PUT/DELETE …/@v/1.4.2.yank   yank / undo (token of that owner)
```

**Owners and names.** A token belongs to an owner and can write only below
that owner (`acme/…`). Who may register an owner name is the service's rule;
the protocol only guarantees that a version, once published, never changes.

**Tiers.** *Listed* is the registry's promise that a version passed automated
gates: version 1.0.0 or later; it builds on Windows and Linux; its tests pass;
no unexplained API break since the previous version. Everything else —
0.x versions, pre-releases, anything that failed a gate — is *unreviewed* and
fully usable; the tier is information, and `deny = ["unlisted"]` in
`[policy]` is how a project says "listed only". (Git packages are always
unlisted.) The gates run in the registry service, not in your compiler; the
reference server in the compiler's repository lists any stable version 1.0.0
or later and does not run them.

**Signed metadata.** The registry signs `hash`, `tier`, yank state and time
of every `.info` with an ed25519 key. You pin the public key in
`[registry] key`; the compiler then refuses metadata signed with any other
key. The signature is over one line of text, so there is nothing ambiguous to
canonicalise.

**Yanks.** A yanked version stays downloadable — so builds do not break — but
no *new* resolution picks it: `add`, `update`, or a fresh clone without a
`veles.sum` line for it are refused with the reason, and `update` skips it.
A project whose `veles.sum` already holds it keeps building and is warned.

**Reviews.** A review is a signed sentence about exact bytes: *"key K says
`reviewed` about `acme/httputil` 1.4.2 with hash H"*. Anyone can sign one; the
consumer decides whose count. See the walk-through below.

**Two ways in.** The author can `veles publish` (an upload with a token), and
a registry service may also pull a tag from a git repository on its own. The
protocol covers only the upload.

> **Status.** The `registry` package in the compiler's repository is a
> *reference implementation* that keeps everything in memory; it exists for the
> tests and to pin the protocol down. The hosted service — persistence,
> registering owners, the real listing gates, a transparency log — is not built
> yet, and there is no `veles registry serve` command. Until a registry is
> running, use git dependencies; everything else in this chapter works with
> them.

## Walk-throughs

### Add a dependency

```text
$ veles add github.com/veles-db/pg
added pg = https://github.com/veles-db/pg 2.1.0 to [dependencies]
pg can use: net, unsafe, extern, native (with its dependencies); tier: unreviewed; `veles audit --detail` shows where
```

`add` finds the newest stable tag, edits one line of `veles.toml`, resolves
the build (fetching, hashing, applying the policy) and prints what the
package can do. If anything fails, `veles.toml` is put back byte for byte.
Read the capabilities: a date-formatting library that can use the network is
worth a look.

### Check what you depend on

```text
$ veles deps                    # the tree; "util 1.0.0 -> 1.3.0" where selection raised it
$ veles deps --why util         # every chain from your package to util
$ veles audit --detail          # each package: tier, capabilities, where they come from, reviewers
```

`audit` exits 1 when the policy is broken, so it works as a CI gate.

### Set a policy for a project or a team

```toml
[registry]
url = "https://registry.example.com"
key = "ed25519:…"

[policy]
deny = ["unsafe", "net", "unlisted"]
allow = { net = ["acme/httputil"], unlisted = ["acme/old"] }
trust = { alice = "ed25519:…", bob = "ed25519:…" }
require = { reviewed = 2 }
```

Every build now refuses a registry or git package that uses `unsafe` or the
network (except `acme/httputil`'s network), is not listed (except
`acme/old`), or has fewer than two `reviewed` statements from Alice and Bob
for its exact bytes. The error names the package and the line that would allow
it. In a workspace, members without their own policy follow the root's.

### Review a package and share the review

```text
$ veles attest keygen alice
wrote the private key to ~/.veles/keys/alice.key (keep it secret)
the public key, for [policy] trust in a project that believes your reviews:
  alice = "ed25519:…"

$ # read the code (veles vendor puts it in vendor/ for you to read)
$ veles attest sign pg --key alice            # the exact bytes in your build
$ veles attest sign pg --key alice --push     # …and send it to the registry
$ veles attest verify                         # your statements against the current build
$ veles attest import reviews.attest          # statements from a colleague, verified first
```

A statement is kept in `attestations/alice.attest`; commit it. When the
package changes (a new version, or a moved tag), old statements no longer match
and stop counting — you review again.

### Publish a package

```toml
[package]
name = "httputil"
version = "1.4.2"
registry = "acme/httputil"
```

```text
$ export VELES_TOKEN=…           # or put it in ~/.veles/token
$ veles publish
published acme/httputil 1.4.2 (versions are immutable; to withdraw it, `veles yank acme/httputil@1.4.2`)
```

`publish` first checks the package, then archives its files — in a git
repository, exactly the files git tracks or would add, so build outputs and
ignored files stay out; elsewhere, the directory without hidden files,
`vendor/`, `bin/` and `veles.sum` — and uploads them. A version cannot be
replaced. If it was a mistake: publish the next one and `veles yank` the bad
one, with a reason.

### Build in CI

```text
veles fetch                 # once: download, verify, record
veles audit                 # fail the job if the policy is broken
veles test                  # a workspace root tests every member
```

Commit `veles.sum`. With `vendor/` committed, the CI job needs no network and
no cache at all.

## Reference

### `veles.toml`

| table | key | meaning |
|---|---|---|
| `[package]` | `name`, `version`, `description`, `license`, `veles` | identity; `veles` is the oldest compiler that builds it |
| `[package]` | `registry = "owner/name"` | where `veles publish` publishes |
| `[dependencies]` | `name = "path"` or `{ path \| git \| registry, version \| commit }` | what the package uses |
| `[dev-dependencies]` | same | only for tests (declared; not yet resolved) |
| `[workspace]` | `members = ["dir", …]` | a repository of several packages |
| `[registry]` | `url`, `key` | the registry's address and its pinned public key |
| `[policy]` | `deny`, `allow`, `trust`, `require` | what dependencies may do and need |
| `[native]` | `libs`, `static-libs`, `lib-paths`, `pkg-config` | C libraries to link |
| `[format]` | `indent`, `max_blank_lines`, `imports` | `veles fmt` settings |
| `[lint]` | `implicit_return` | lints of the package's own code |

Unknown tables and keys are errors, with the list of valid ones.

### Capabilities

| name | the source… |
|---|---|
| `unsafe` | has an `unsafe` block or function |
| `extern` | has an `extern` block, or a function C can call |
| `native` | has a `[native]` table |
| `net` | imports `net`, `tls`, `http`, `db` or `otel` |
| `fs` | imports `fs` or `config` |
| `os` | imports `os` or `config` |
| `ffi` | imports `ffi` |
| `unlisted` | is not in the registry's listed tier |

Every directory below a package's root counts, including ones named `vendor`
or holding a `veles.toml` of their own (the compiler reads them as modules);
test files and hidden directories do not.

### Environment

| variable | meaning |
|---|---|
| `VELES_HOME` | where the module cache, keys and token live (default `~/.veles`) |
| `VELES_PROXY` | the registry's address; overrides `[registry] url` |
| `VELES_TOKEN` | your registry token; overrides `~/.veles/token` |
| `VELES_ALLOW_LOCAL_GIT` | let a *fetched* package's manifest name local paths and `file://` repositories (a private mirror on disk, an air-gapped build, tests); leave unset otherwise |

### Commands

| command | what it does |
|---|---|
| `veles fetch [dir]` | resolve, download and record; asks the registry again about cached packages |
| `veles add <spec>[@version] [--as name] [--dev]` | add a dependency (path, git URL or `owner/name`) |
| `veles update <name>[@version]… \| --all` | newest release of the same major; a newer major is only mentioned |
| `veles remove <name>…` | remove, and prune unused `veles.sum` lines |
| `veles deps [--why name]` | the build as a tree |
| `veles audit [--detail]` | tier, capabilities and reviews of every package; exit 1 if the policy is broken |
| `veles vendor` | copy the build into `vendor/` |
| `veles publish` | upload the package to its registry |
| `veles yank <owner/name>@<version> [--reason t] [--undo]` | withdraw a version from new use |
| `veles attest keygen\|sign\|verify\|import` | create keys, sign, check and import reviews |

## When something is refused

| message (shortened) | what it means | what to do |
|---|---|---|
| `checksum mismatch for git … 1.4.2` | the files for that version are not the ones `veles.sum` recorded: the tag was moved, the archive changed, or the cache was edited | find out why (`git log`, the upstream issue tracker); if you trust the new contents, delete that line from `veles.sum` (and the version's directory in `~/.veles/pkg` if it is there) |
| `the module cache entry … was modified after it was fetched` | something edited a file in `~/.veles/pkg` | delete that directory; it is fetched again |
| `… is not in the module cache; run veles fetch` | offline (the language server, `vendor/`) and the package was never downloaded | `veles fetch` |
| `[policy] refuses N use(s)` | a package breaks your `[policy]` | read the line it prints; remove the dependency, or add it to `allow` after reviewing it |
| `has 0 of the 2 trusted "reviewed" review(s)` | `require` wants reviews you do not hold | have a trusted reviewer `veles attest sign` it, or `veles attest import` their file |
| `was yanked from the registry: <reason>` | the author withdrew that version | choose another (`veles update <name>`); an existing build only warns |
| `signed with … not the key this project pins` | the registry's metadata is not signed by the key in `[registry] key` | do not proceed until you know why: a hijacked host, or the registry rotated its key |
| `the archive hashes to … but the registry's metadata says …` | the registry served bytes that contradict its own metadata | do not use that registry until it explains |
| `needs a proxy … set VELES_PROXY` | a registry dependency but no registry address | set `[registry] url` or `VELES_PROXY`, or depend on the git repository |
| `refusing to send a token … over http` | publish/yank to a non-local `http://` address | use `https://` |
| `is pinned by tag … and by commit …` | one repository, one major, two pins | keep one |
| `a fetched package cannot depend on a directory` | a dependency's manifest uses a `path` | it must publish that dependency and require it by version |
| `the address … is a local path, and a fetched package's manifest may not name one` | a dependency names a repository on your disk | see `VELES_ALLOW_LOCAL_GIT` if that is deliberate |
| `dependency 'io' … has the name of a standard module` | a local name that would make `use io` ambiguous | choose another name (`--as`) |

## Not built yet

Recorded in the project's checklist so they are built later: the hosted
registry service and its real listing gates; a runnable private registry
(`veles registry serve`) with persistence; a transparency log and replay
protection for signed metadata; more than one registry per project;
`veles update --major`; resolving `[dev-dependencies]`; publishing the members
of a workspace; revoking or expiring a review; `package.vs` (the manifest as a
typed constant).
