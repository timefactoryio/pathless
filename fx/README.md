# fx

`fx` encodes content into the wire format and registers routes and frames on top of it. `circuit.go` defines the `Value` tree, its wire encoding, and route storage/persistence; `fx.go` builds on `Circuit` to consolidate frame files into the shape the wire expects.

| File         | Role                                                                               |
| ------------ | ---------------------------------------------------------------------------------- |
| `circuit.go` | `Circuit`/`Value` — wire `Encode`/`Decode`, route storage, `Load`/`Save`/`ToBytes` |
| `fx.go`      | `Fx` (embeds `*Circuit`) — frame/panel registration, asset consolidation, bundling |

---

## `circuit.go` — wire format & routes

### `Value`

```go
type Value struct {
    Name   string   // server-internal only — sort.txt ordering; never wire-encoded
    Type   string   // MIME type, leaves only
    Data   []byte   // raw bytes, leaves only
    Values []*Value // children — presence of Values makes this a bundle, not a leaf
}
```

A `Value` is either a **leaf** (`Type`+`Data`) or a **bundle** (`Values`) — `Encode` walks `Values` recursively and only writes leaves to the wire. `Name` is never wire-encoded; it only orders/re-identifies entries for `sort.txt` and `Save`/`Load`.

```go
type Circuit struct {
    Routes map[string]*Value
}

func NewCircuit() *Circuit
```

`Fx` embeds `*Circuit`, so `Routes` and every method below are promoted onto `Fx` (and, through `One`, onto `one`).

### `Encode(v) []byte` / `Decode(buf) []*Value` — wire format

```
[1B tableCount]
tableCount × [1B typeLen][type]        // string table: distinct MIME types
repeated until EOF:
  [1B tableIndex][4B dataLen BE][data]
```

`Encode` flattens `v` to its leaves in order, deduplicates MIME types into a table, and writes the layout above — no leaf count, the response's length is the terminator. `Decode` is the exact inverse. This is purely the wire format, applied fresh at serve time — changing it never invalidates anything already persisted via `Save`.

### `Load(path)` — registering a route

| `path`           | Result                                                                                                                            |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| local file       | one leaf (`read`) — `Type` from extension, content-sniffed as fallback                                                            |
| local directory  | one bundle (`walk`) — every file in the directory, in `sort.txt` order if present                                                 |
| `http(s)://` URL | fetched (`ToBytes`) and `gob`-decoded directly into a `Value` tree — expects a `Save`-produced blob, not an arbitrary remote file |

The route key is always `filepath.Base(path)`. `walk` skips `sort.txt` itself, then, if present, reorders `Values` to match it: listed stems first in listed order, unlisted files appended after.

### `Save(key) error`

Gob-encodes the full `Value` tree at `Routes[key]` (the tree, not `Encode`'s wire output) to `s3/<key>`. Because it's the tree, `Encode`/`Decode` can change freely without ever requiring a re-save. A saved route round-trips through `Load(url)`.

---

## `fx.go` — frame/panel building

```go
type Fx struct {
    *Circuit
    frames []*template.HTML // registered space frames, in registration order
    panels []*template.HTML // registered panel frames, in registration order
}

func NewFx() *Fx
```

| Member                    | Behavior                                                                                                                                                                      |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Frame(path)`             | reads a local `.html` file and passes it to `Build`. Nothing is wrapped for you — the file authors its own root container div, so only pass local, developer-controlled paths |
| `Build(elements...)`      | concatenates elements' HTML, runs `consolidateAssets`, registers the result via `Frames`                                                                                      |
| `Frames(frame...)`        | appends `frame` (if given), returns the whole frame pool pre-encoded as one bundle (`bundle`)                                                                                 |
| `Panel(path)`             | reads a local file, consolidates it via `BuildPanel` — **returns** the result, does not register it                                                                           |
| `BuildPanel(elements...)` | same consolidation as `Build` — returns the result, does not register it                                                                                                      |
| `Panels(panels...)`       | registers any given panel(s), returns the whole panel pool pre-encoded as one bundle                                                                                          |

The panel split (`Panel`/`BuildPanel` build but don't register) is why panel frames are composed then passed explicitly: `p.Panels(p.Keyboard())`. `Frame`/`Build` are how a template builder or custom `.html` file becomes one entry in the frame pool `one.Serve` sends as item 1 of the root payload.

### `consolidateAssets(s)` — build-time transform

Runs once per `Build`/`BuildPanel` call, never per request:

1. More than one `<style>` block → hoisted (concatenated, in order) into a single block at the front; originals removed.
2. Every `<script>` block's content is rewrapped in `{ }` if it isn't already (the fallback for scripts that weren't authored wrapped, per [mechanics.md](../mechanics.md#hard-rules)).
3. All script content is concatenated into one block, appended at the end.

The result is exactly one `<style>` (if any) → markup → one `<script>{ ... }</script>`, regardless of how many blocks the source had.

### `bundle(frames) *Value` — nested pre-encoding

Wraps each frame's HTML as a `text/html` leaf, then pre-encodes the list with `Encode` into a single `application/octet-stream` `Value`. This is what lets `Frames`/`Panels` hand back one `*Value` that already contains a fully wire-encoded bundle — `one.Serve` embeds it directly as one leaf of the root payload, and the client's `decode` recurses one level (see [zero](../zero/README.md#decodebytes--wire-format)) to unpack it.

---

## See also

- [mechanics.md](../mechanics.md#fx--authoring-and-registering-a-frame) — the frame-author-facing rules for `main.go` (`p.Frame`, `p.Load`, `p.Panels`, `sort.txt`) that this package implements.
- [zero/README.md](../zero/README.md) — the client that decodes what `Encode`/`bundle` produce here.
