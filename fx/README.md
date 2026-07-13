# fx

`fx` encodes content into the wire format and registers routes and frames on top of it. `circuit.go` defines `Value`, its wire encoding, and route persistence; `fx.go` defines `Fx` — the frame/panel pools, the route map, and asset consolidation.

| File         | Role                                                                         |
| ------------ | ---------------------------------------------------------------------------- |
| `circuit.go` | `Value` — wire `Encode`/`Decode`, `ToValue`/`walk`, `Save`                   |
| `fx.go`      | `Fx` — frame/panel pools, route map (`Routes`), asset consolidation, `Build` |

---

## `circuit.go` — wire format & routes

### `Value`

```go
type Value struct {
    Name string // server-internal only — sort.txt ordering; never wire-encoded
    Type string // MIME type
    Data []byte // raw bytes — a directory's Data is itself a pre-encoded Encode() blob
}
```

A directory has no separate tree shape: `ToValue` walks it (`walk`) and pre-encodes every file into one `Value` whose `Data` is already a complete `Encode()` blob (`Type: "application/octet-stream"`). `Name` is never wire-encoded; it only orders/re-identifies entries for `sort.txt` and `Save`.

`Routes map[string]*Value` is a plain field on `Fx` (see [fx.go](#fxgo--framepanel-building)) — there is no separate `Circuit` wrapper type.

### `Encode(v) []byte` / `Decode(buf) []*Value` — wire format

```
[1B tableCount]
tableCount × [1B typeLen][type]        // string table: distinct MIME types
repeated until EOF:
  [1B tableIndex][4B dataLen BE][data]
```

`Encode` takes the `*Value`s directly — there's no tree to flatten — deduplicates MIME types into a table, and writes the layout above — no entry count, the response's length is the terminator. `Decode` is the exact inverse. This is purely the wire format, applied fresh at serve time — changing it never invalidates anything already persisted via `Save`, since `Save` gob-encodes the `Value` itself, not this wire encoding.

### `ToValue(input) (*Value, error)` — building a route value

| `input`          | Result                                                                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------ |
| local file       | one `Value` — `Type` from extension, content-sniffed as fallback                                                         |
| local directory  | one `Value` (`walk`) — every file in the directory, pre-encoded via `Encode` into `Data`, in `sort.txt` order if present |
| `http(s)://` URL | fetched directly and treated like a file — `Type` from extension, content-sniffed as fallback                            |

`ToValue` only builds the `*Value`; it doesn't register anything. Callers assign the result into `Routes` themselves, keyed by `filepath.Base(input)` (e.g. `f.Routes[filepath.Base(path)] = v`, as `Logo`/`Slides` do in [templates.go](../one/README.md#templatesgo--built-in-frame-templates)). `walk` skips `sort.txt` itself, then, if present, reorders entries to match it: listed stems first in listed order, unlisted files appended after.

### `Save(key) error`

Gob-encodes the `Value` at `Routes[key]` (`Name`, `Type`, `Data` — not `Encode`'s wire output) to `s3/<key>`. Because it's gob, `Encode`/`Decode` can change freely without ever requiring a re-save. Note `ToValue` doesn't currently gob-decode a `Save`d blob back — an `http(s)://` input is just fetched and treated as a plain file, so a full round trip through a remote URL isn't wired up yet.

---

## `fx.go` — frame/panel building

```go
type Fx struct {
    Frames []*Value          // registered space frames, in registration order
    Panels []*Value          // registered panel frames, in registration order
    Routes map[string]*Value // custom data routes, keyed by filepath.Base
    Origin string            // CORS-allowed root domain
    Hello  []*Value          // the "/" payload — assembled by Build
}

func NewFx(origin string) *Fx
```

`NewFx` builds the universe shell (`build(universeHtml)`) as `Hello[0]`; `Build()` turns that into the final ordered payload.

| Member        | Behavior                                                                                                                                                                       |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Frame(path)` | reads a local/remote `.html` file (`ToValue`) and appends its consolidated (`build`) form to `Frames`. A failed read is fatal — everything served must be available at startup |
| `Panel(path)` | same as `Frame`, but appends to `Panels` instead                                                                                                                               |
| `build(s)`    | consolidates a fragment's `<style>`/`<script>` blocks into one `text/html` `Value` (see below)                                                                                 |
| `Build()`     | prepends a 1-byte frame-count header to `Frames`, then assembles `Hello = [header, universe, ...Frames, ...Panels]`                                                            |

`Build`'s header lets the client slice `Frames` from `Panels` without either needing its own wire entry — `Hello` as a whole is `Encode`d once, by `one.wire("/", o.Hello)`.

### `build(s)` — build-time transform

Runs once per `Frame`/`Panel`/template-builder call, never per request:

1. More than one `<style>` block → hoisted (concatenated, in order) into a single block at the front; originals removed.
2. Every `<script>` block's content is rewrapped in `{ }` if it isn't already (the fallback for scripts that weren't authored wrapped, per [mechanics.md](../mechanics.md#hard-rules)).
3. All script content is concatenated into one block, appended at the end.

The result is exactly one `<style>` (if any) → markup → one `<script>{ ... }</script>`, regardless of how many blocks the source had.

---
