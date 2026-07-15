# fx

`fx` sources and processes content into `Value`s and builds the frame/panel pools and route map. It knows nothing of the wire format, HTTP, or where content is served from — [one](../one/README.md) encodes and serves what `fx` produces.

| File           | Role                                                                                                                     |
| -------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `value.go`     | `Value` — the content tree; `ToValue`/`walk` build it from files, dirs, and URLs                                         |
| `fx.go`        | `Fx` — frame/panel pools, route map (`Routes`), `<style>`/`<script>` consolidation                                       |
| `templates.go` | `Home`/`Logo`/`Text`/`Slides`/`Keyboard` — built-in frame builders (see [below](#templatesgo--built-in-frame-templates)) |

---

## `value.go` — content values

### `Value`

```go
type Output struct {
    Name     string   // server-internal only — sort.txt ordering & Save; never on the wire
    Type     string   // MIME type
    Data     []byte   // leaf bytes
    Inputs   []*Output // directory bundle — its files, in sort.txt order
}

func (v *Value) Save() ([]byte, error)
```

`Value` is a **tree**: a leaf carries `Data`; a directory carries `Inputs` and no `Data` of its own. `fx` never encodes — turning this tree into wire bytes (a directory's payload becomes `Encode(Inputs)`) is [one](../one/README.md#wirego--wire-format)'s job. `Name` is never sent on the wire; it only orders/re-identifies entries for `sort.txt` and `Save`.

`Save` gob-encodes the full tree (`Name`, `Type`, `Data`, `Inputs`) into bytes and hands them back — no filesystem, no bucket, no assumed destination. It's deliberately separate from the wire format, so changes to `Encode` never invalidate anything already saved. `Fx.Save(key)` (see [fx.go](#fxgo--framepanel-building)) looks up `Routes[key]` and calls this.

`Routes map[string]*Value` is a plain field on `Fx` (see [fx.go](#fxgo--framepanel-building)).

### `ToValue(input) (*Value, error)` — building a value

| `input`          | Result                                                                                        |
| ---------------- | --------------------------------------------------------------------------------------------- |
| local file       | one leaf `Value` — `Type` from extension, content-sniffed as fallback                         |
| local directory  | one directory `Value` (`walk`) — every file as a child leaf, in `sort.txt` order if present   |
| `http(s)://` URL | fetched directly and treated like a file — `Type` from extension, content-sniffed as fallback |

`ToValue` only builds the `*Value`; it doesn't register anything. To make it fetchable, callers hand it to `Route(key, v)` (see [fx.go](#fxgo--framepanel-building)), keyed by `filepath.Base(input)` (as `Logo`/`Slides` do in [templates.go](#templatesgo--built-in-frame-templates)). `walk` skips `sort.txt` itself, then, if present, reorders entries to match it: listed stems first in listed order, unlisted files appended after.

---

## `fx.go` — frame/panel building

```go
type Fx struct {
    Frames *Value          // registered space frames, in registration order
    Panels *Value          // registered panel frames, in registration order
    Routes map[string]*Value // custom data routes, keyed by filepath.Base
}

func NewFx() *Fx
```

`Fx` holds only content — no config, no wire state. Assembling the `/` payload from these pools is done in [one](../one/README.md#onego--server), not here.

| Member          | Behavior                                                                                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Frame(path)`   | reads a local/remote `.html` file (`ToValue`) and appends its consolidated (`build`) form to `Frames`. A failed read is fatal — everything served must be available at startup |
| `Panel(path)`   | same as `Frame`, but appends to `Panels` instead                                                                                                                               |
| `Route(key, v)` | registers a built `*Value` as a served route under `key` (into `Routes`) and returns `key` — the one operation that makes content fetchable via `p.source(key)`                |
| `Save(key)`     | looks up `Routes[key]` and returns its gob-encoded bytes (via `Value.Save`), ready for the caller to persist however it likes                                                  |
| `build(s)`      | consolidates a fragment's `<style>`/`<script>` blocks into one `text/html` `Value` (see below)                                                                                 |

### `build(s)` — build-time transform

Runs once per `Frame`/`Panel`/template-builder call, never per request:

1. More than one `<style>` block → hoisted (concatenated, in order) into a single block at the front; originals removed.
2. Every `<script>` block's content is rewrapped in `{ }` if it isn't already (the fallback for scripts that weren't authored wrapped, per [mechanics.md](../mechanics.md#authoring-rules)).
3. All script content is concatenated into one block, appended at the end.

The result is exactly one `<style>` (if any) → markup → one `<script>{ ... }</script>`, regardless of how many blocks the source had.

---

## `templates.go` — built-in frame templates

Four embedded template sources (`frames/home.html`, `frames/slides.html`, `frames/text.html`, `panels/keyboard.html`), parsed and executed on demand by these builders. Each appends directly to `Frames`/`Panels` via `build`, so registration follows the same rules as any hand-authored frame.

| Builder               | Behavior                                                                                                                                                                                                                                                                                       |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Home(logo, heading)` | renders `frames/home.html` with `{{.LOGO}}` (via `Logo`) and `{{.HEADING}}`, registers it as a space frame                                                                                                                                                                                     |
| `Logo(path)`          | `.svg` → inlined verbatim (`<svg>` markup, via `ToValue`). Anything else → registered as a route (`ToValue` + `Route`) and referenced by `data-src="{routekey}"` — the client's `p.source` prepends `window.circuit` when it lazily fetches. A remote URL is used as `src` directly, unfetched |
| `Text(path)`          | fetches `path` (`ToValue`), renders it as Markdown (`markdown.New("")`), injects the HTML into `frames/text.html`, registers it as a space frame                                                                                                                                               |
| `Slides(dir)`         | registers `dir` as a route (`ToValue` + `Route`; its images, in `sort.txt` order if present), renders `frames/slides.html` with `{{.PREFIX}}` set to the route's key, registers it as a space frame — the frame's own script then fetches that route client-side                               |
| `Keyboard()`          | builds `panels/keyboard.html` via `build` and appends it directly to `Panels`. Nothing calls it automatically — a program must call `p.Keyboard()` itself to get the default panel                                                                                                             |

`Logo`'s two paths mirror the two things a frame can point at: embed small/local assets directly, or register anything larger as its own route and let the client fetch it lazily against `window.circuit`.

---
