# one

`one` is the HTTP layer. It takes [zero](../zero/README.md)'s compiled assets and [fx](../fx/README.md)'s processed `Value`s, encodes them into the client wire format, gzip-compresses everything once at startup, and serves it from memory: the shell on `:1000`, the wire gateway on `:1001`.

| File      | Role                                                                   |
| --------- | ---------------------------------------------------------------------- |
| `one.go`  | `One` — two `http.ServeMux`s, `/` payload assembly, gzip, `Serve()`    |
| `wire.go` | `Encode`/`Decode` (the wire format) and `Save` (gob route persistence) |

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
2. assembles the `/` payload — `[1-byte frame-count header, universe, ...Frames, ...Panels]` — `Encode`s it, gzips it, and registers it on `circuit`.
3. `Encode`s + gzips each `Routes` entry and registers it at `/`+key on `circuit`.
4. registers the shell's catch-all handler on `pathless`.

The 1-byte header lets the client slice `Frames` from `Panels` without either needing its own wire entry; `universe` is `zero`'s payload bytes wrapped as a `text/html` `Value` here.

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

## `wire.go` — wire format & persistence

### `Encode(values ...*fx.Value) []byte` / `Decode(buf) []*fx.Value`

```
[1B typeCount]
typeCount × [1B typeLen][type]        // string table: distinct MIME types
repeated until EOF:
  [1B typeID][4B dataLen BE][data]
```

Distinct MIME types are written once, up front, as a table; each entry then costs just a 1-byte type id and a 4-byte length ahead of its data. There is no entry count — the response's length is the terminator. Names are never encoded — order is the contract.

A directory `Value` carries no bytes of its own: its wire payload is `Encode(Children)`, so the client decodes it exactly the way it decodes the top-level response (decode once, then decode a directory entry's data again). `Decode` is the one-level inverse. This is purely the over-the-network format — `Save` uses gob instead, so changing it never invalidates anything already persisted.

### `Save(key, v) error`

Gob-encodes the full `Value` tree (`Name`, `Type`, `Data`, `Children` — not the wire format) to `s3/<key>`, ready to sync to object storage. Because it's gob, `Encode`/`Decode` can change freely without ever requiring a re-save. `pathless.Save(key)` looks up `Routes[key]` and delegates here.

---

