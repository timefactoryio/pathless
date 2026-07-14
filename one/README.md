# one

`one` is the HTTP layer. It takes [zero](../zero/README.md)'s compiled assets and [fx](../fx/README.md)'s processed `Value`s, encodes them into the client wire format, gzip-compresses everything once at startup, and serves it from memory: the shell on `:1000`, the wire gateway on `:1001`.

| File      | Role                                                                |
| --------- | ------------------------------------------------------------------- |
| `one.go`  | `One` — two `http.ServeMux`s, `/` payload assembly, gzip, `Serve()` |
| `wire.go` | `Encode`/`Decode` — the wire format, nothing else                   |

---

## `one.go` — server

```go
type One struct {
    origin   string         // CORS allow-origin
    shell    []byte         // gzipped HTML shell, served at "/"
    pathless *http.ServeMux // ":1000" — the shell
    circuit  *http.ServeMux // ":1001" — wire routes
}

func NewOne(origin string, shell, universe []byte, f *fx.Fx) *One
```

`One` embeds no `zero`/`fx` structs — it receives their outputs as plain bytes plus a `*fx.Fx` to read the pools. `NewOne` does all assembly up front:

1. gzips the shell.
2. assembles the `/` payload — `[universe, frames-bundle, panels-bundle]` — `Encode`s it, gzips it, and registers it on `circuit`.
3. `Encode`s + gzips each `Routes` entry and registers it at `/`+key on `circuit`.
4. registers the shell's catch-all handler on `pathless`.

`Frames`/`Panels` are each wrapped as one directory-style `Value` (`Children: f.Frames`/`f.Panels`) rather than flattened into the list with a count prefix — the client decodes each bundle the same way it decodes any nested directory entry, no special-cased header required. `universe` is `zero`'s payload bytes wrapped as a `text/html` `Value` here.

| Member                 | Behavior                                                                                                                                                   |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `handlePathless(w, r)` | serves the shell at `/` (gzip). Any other path or a non-empty query redirects to `/` — the shell is a single page, routing lives entirely on the wire side |
| `wire(path, data)`     | gzip-compresses `data` once and registers a handler on `circuit` that serves those fixed bytes as `application/octet-stream`                               |
| `cors(next)`           | sets `Access-Control-Allow-Origin: origin`, short-circuits `OPTIONS` with `204` — wraps `circuit` only                                                     |
| `Serve()`              | starts `:1001` (wire, CORS-wrapped) in a goroutine, `:1000` (shell) blocking on the main goroutine                                                         |

Both muxes' routes are fixed by the time `NewOne` returns; nothing is registered after. Whether the panel pool is non-empty is up to the caller — `pathless` registers `Keyboard()` before serving if nothing else did.

### `zip(data) []byte`

Gzips `data` at `gzip.BestCompression`. Used for the shell (once) and every wire route (once) — never per request. Every response this package writes sets `Content-Encoding: gzip` to match.

---

## `wire.go` — wire format

### `Encode(values ...*fx.Value) []byte` / `Decode(buf) []*fx.Value`

```
[1B typeCount]
typeCount × [1B typeLen][type]        // string table: distinct MIME types
[1B entryCount]
repeated entryCount times:
  [1B typeID]?[4B dataLen BE][data]   // typeID omitted when typeCount == 1
```

Distinct MIME types are written once, up front, as a table. `entryCount` lets `Decode` allocate its result at the exact size up front, instead of growing dynamically. Each entry then costs a 4-byte length ahead of its data, plus a 1-byte type id — unless every value in this call shares one type, in which case the id is implied by the table and omitted entirely (the common case: a `Frames`/`Panels` bundle, an image directory). Names are never encoded — order is the contract.

A directory `Value` carries no bytes of its own: its wire payload is `Encode(Children)`, tagged with the `application/x-bundle` type so the client's `decode` recognizes it and recurses into it automatically — the caller never decodes an entry's data a second time by hand. `Decode` (this Go package's, mirrored by the client's `decode`) is the one-level inverse. This is purely the over-the-network format — gob-based persistence (`Value.Save`, in [fx](../fx/README.md#valuego--content-values)) is deliberately separate, so changing this format never invalidates anything already persisted.

---

