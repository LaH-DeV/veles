# The registry protocol

What a Veles registry serves and accepts (spec D139). A client needs only the
read calls; `veles publish`, `veles yank` and `veles attest sign --push` use
the write calls. The `registry` package in the compiler's repository is a
reference implementation that keeps everything in memory.

A registry is a base URL (`[registry] url` in `veles.toml`, or the
environment variable `VELES_PROXY`). A package is `owner/name`: lowercase
letters, digits, `-` and `_` on each side of one `/`. A version is
`major.minor.patch` with an optional `-pre-release`, written without a
leading `v`.

## Read

| call | answers |
|---|---|
| `GET <base>/<owner>/<name>/@v/list` | the versions, one per line, yanked ones included |
| `GET <base>/<owner>/<name>/@v/<version>.info` | the version's metadata, JSON, signed |
| `GET <base>/<owner>/<name>/@v/<version>.zip` | the package's files, at the root of a zip |
| `GET <base>/<owner>/<name>/@v/<version>.attest` | signed reviews of the version, one per line |

### `.info`

```json
{"version":"1.4.2","time":"2026-10-01T09:00:00Z","hash":"h1:3b6f…",
 "tier":"listed","yanked":false,"yankReason":"","key":"ed25519:…","sig":"…"}
```

- `hash` is the `h1:` hash of the archive's files (the one `veles.sum`
  records): SHA-256 over each file's slash path and the SHA-256 of its bytes,
  sorted by path.
- `tier` is `listed` or `unreviewed`. *Listed* is the registry's promise that
  the version passed its automated gates (version 1.0.0 or later; builds on
  Windows and Linux; its tests pass; an API diff against the previous
  version).
- `yanked` withdraws a version from new use; it stays downloadable.
- `sig` is base64 ed25519 over the text
  `veles-info-v1 <owner/name> <version> <hash> <tier> <0|1 yanked> <time>`,
  made with the private half of `key`. A project pins the registry's key
  (`[registry] key`); the client then refuses metadata signed by any other
  key. `yankReason` is for people and is not signed.

The client checks that the archive hashes to `hash` and that `hash` agrees
with `veles.sum`; a disagreement is a hard error and nothing is cached.

### `.attest`

One statement per line:

```text
attest v1 <git|registry> <source> <version|commit:full> <h1 hash> <claim> <time> <ed25519:key> <sig>
```

`sig` is base64 ed25519 over `veles-attest-v1 <kind> <source> <ref> <hash>
<claim> <time>`. A statement is about exact bytes, so it counts only for the
package whose hash it names. It verifies on its own, wherever it is kept.

## Write

Write calls carry `Authorization: Bearer <token>`. A token belongs to an
owner (`VELES_TOKEN`, or the file `~/.veles/token`).

| call | effect |
|---|---|
| `PUT <base>/<owner>/<name>/@v/<version>.zip` | publish; `201` and the hash, `409` if the version exists, `403` for another owner's name, `422` for an archive that is not a package |
| `POST <base>/<owner>/<name>/@v/<version>.attest` | add one statement (any token; the registry keeps it only if it is about this version's hash) |
| `PUT <base>/<owner>/<name>/@v/<version>.yank` | yank; the body is the reason |
| `DELETE <base>/<owner>/<name>/@v/<version>.yank` | undo it |

A published version is immutable. How a name is registered, and that a
registry may also publish a tag from a git repository on its own, is up to
the service and is outside this protocol.
