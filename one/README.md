# one

`one` is the HTTP layer. It takes [zero](../zero/README.md)'s compiled assets and [fx](../fx/README.md)'s processed `Value`s, serves each route as self-describing wire bytes, gzip-compresses everything once at startup, and serves it from memory: the shell on `:1000`, the wire gateway on `:1001`.

| File      | Role                                                                      |
| --------- | ------------------------------------------------------------------------- |
| `one.go`  | `One` — two `http.ServeMux`s, per-route `serve`, gzip, `Serve()`          |
| `wire.go` | `Encode`/`payload`/`sequence`/`typeTable` — the wire format, nothing else |

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
2. `serve`s the root as one bundle `Value` whose children are `[universe, frames-bundle, panels-bundle]`, registered on `circuit`.
3. `serve`s each `Routes` entry at `/`+key on `circuit`.
4. registers the shell's catch-all handler on `pathless`.

Every route — the root and each registered route alike — is **exactly one `*fx.Value`**, served as `Encode(v)`: its type, then its payload. The value's type travels **in-band**, so the HTTP response needs no meaningful `Content-Type` — it's opaque `application/octet-stream`, and the client reconstructs the `Value` from the bytes. The root is not special-cased; it is simply a bundle `Value` (`application/x-bundle`) whose children are the universe payload and the frame and panel pools. `universe` is `zero`'s payload bytes as a `text/html` leaf.

| Member                 | Behavior                                                                                                                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `handlePathless(w, r)` | serves the shell at `/` (gzip). Any other path or a non-empty query redirects to `/` — the shell is a single page, routing lives entirely on the wire side                                       |
| `serve(path, v)`       | gzip-compresses `Encode(v)` once and registers a handler on `circuit` that serves those fixed bytes as opaque `application/octet-stream` — the value's real type is carried in-band in the frame |
| `cors(next)`           | sets `Access-Control-Allow-Origin: origin`, short-circuits `OPTIONS` with `204` — wraps `circuit` only                                                                                           |
| `Serve()`              | starts `:1001` (wire, CORS-wrapped) in a goroutine, `:1000` (shell) blocking on the main goroutine                                                                                               |

Both muxes' routes are fixed by the time `NewOne` returns; nothing is registered after. Whether the panel pool is non-empty is up to the caller — `pathless` registers `Keyboard()` before serving if nothing else did.

### `zip(data) []byte`

Gzips `data` at `gzip.BestCompression`. Used for the shell (once) and every wire route (once) — never per request. Every response this package writes sets `Content-Encoding: gzip` to match.

---

## `wire.go` — wire format

Two matched framings, one for each cardinality — a route is **one** value; a bundle's children are **many**.

### `Encode(v *fx.Value) []byte` — one value (a route)

```
[1B typeLen][type][payload]
```

A route is exactly one `Value`, so `Encode` frames it as its type followed by its `payload` — no counts, no length, because the payload runs to the end of the buffer. The type rides in-band, so the HTTP response needs no `Content-Type`. `payload(v)` is a leaf's `Data`, or a bundle's children as a `sequence` (below). `serve` writes `Encode(v)` for every route, the root included.

### `sequence(values []*fx.Value) []byte` — many values (a bundle's children)

```
[1B typeCount]
typeCount × [1B typeLen][type]        // string table: distinct MIME types
[1B entryCount]
repeated entryCount times:
  [1B typeID]?[4B dataLen BE][data]   // typeID omitted when typeCount == 1
```

When a value holds `Children`, its `payload` is their `sequence`. Distinct MIME types are written once, up front, as a table — built by `typeTable`, which returns the encoded table, the type→id map, and whether the bundle is single-typed. `entryCount` lets the client allocate its result at the exact size up front, instead of growing dynamically. Each entry then costs a 4-byte length ahead of its data, plus a 1-byte type id — unless every value shares one type, in which case the id is implied by the table and omitted entirely (the common case: a `Frames`/`Panels` bundle, an image directory). A child that is itself a bundle carries its own `sequence` under `application/x-bundle`, decoded on the client through `Value.children`. Names are never encoded — order is the contract.

`Encode`/`sequence`/`typeTable` are the **sole codec** — the client mirrors them in reverse (`source`'s inline frame parse plus `#decode`/`#typeTable`); there is no server-side decode. This is purely the over-the-network format — gob-based persistence (`Value.Save`, in [fx](../fx/README.md#valuego--content-values)) is deliberately separate, so changing this format never invalidates anything already persisted.

---

