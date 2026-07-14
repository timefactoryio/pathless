# zero

`zero` compiles the two browser-runtime assets every request is built from — the HTML shell and the universe payload — once, at startup, from embedded sources to be served from memory.

| File            | Role                                                                                  |
| --------------- | ------------------------------------------------------------------------------------- |
| `zero.go`       | Go package `zero` — embeds the two HTML files, templates + minifies the shell         |
| `pathless.html` | the shell: page chrome + `Pathless` client (wire fetch/decode, render, bootstrap)     |
| `universe.html` | the universe payload: `Universe`, `Input`, `Panel` — layout, state, and input runtime |

---

## `zero.go` — build-time compilation

```go
type Zero struct {
    Pathless []byte // minified, templated HTML shell
    Universe []byte // universe payload, embedded bytes untouched
}

func NewZero(circuit string) *Zero
```

`NewZero` does all the work up front:

1. Executes `pathless.html` as a Go template with `{{.CIRCUIT}}` substituted (baking `circuit` in as `window.circuit`), then minifies the result into `Pathless`.
2. Carries `universe.html`'s embedded bytes into `Universe` untouched — it needs no consolidation, being a single already-wrapped `<script>` with no `<style>`.

`circuit` is only used during templating; it is **not** stored on `Zero` — nothing reads it after compilation. Wrapping `Universe` in a wire `Value` and encoding both artifacts is [one](../one/README.md)'s job.

`minify` aggressively strips `pathless.html`'s `<style>`/`<script>`/markup down to its minimum byte size. It is hand-tuned for `pathless.html`'s exact content — not a general-purpose minifier — and runs once, inside `NewZero`.

---

## `pathless.html` — shell + client bootstrap

Static page chrome (`<style>` for the universe/panel layout grid, `<body>` with `#universe` and `#panel` containers) plus one inline script defining `Pathless`, the sole global (`window.pathless`).

### `Pathless`

| Member              | Signature                           | Behavior                                                                                                                                                                                                                                                                                                |
| ------------------- | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cache`             | `Map<string, Promise<Value>>`       | one in-flight/settled fetch per route path; a settled entry is a single `Value`, so every caller of a path shares one fetch, one decode, and one set of blob urls                                                                                                                                       |
| `source(path = '')` | `(string) => Promise<Value>`        | fetches `${window.circuit}/${path}` and reconstructs the single `Value` it frames inline; caches by `path`; failed fetches evict the cache entry. The response has no meaningful `Content-Type` — the type is in-band. A leaf route is read via `.data`/`.url`; a bundle route via `.children`          |
| `exec(el, data)`    | `(HTMLElement, Uint8Array) => void` | decodes UTF-8 `data` into a document fragment (`Range#createContextualFragment`) and replaces `el`'s children — this is how a frame's `<style>/markup/<script>` gets injected and (re-)executed                                                                                                         |
| `init()`            | `() => Promise<void>`               | fetches the root route (`''`), whose `Value` is a bundle of `[universe, frames, panels]`; keeps the universe payload, maps each pool's `.children` to raw bytes on `this.frames`/`this.panels`, execs the universe payload into `#universe` (constructing `Universe` with them), then `universe.init()` |

`source` parses a route's one-value frame inline; a private `Pathless.#decode` reads a bundle's child sequence for the `Value` constructor (`.children`), delegating the shared type table to `Pathless.#typeTable`. Frames never touch the wire — they read `.data`/`.url`/`.children`.

`window.circuit` is set from the templated `{{.CIRCUIT}}`, with `localhost` swapped for the page's actual hostname — so the same compiled shell works when accessed via a LAN IP or a tunnel in dev.

### wire format — `#decode` and `#typeTable`

A route is served as one value frame (parsed inline by `source`); a bundle's children are a nested sequence:

```
route frame:       [1B typeLen][type][payload]           // one Value (a route)

#typeTable(bytes): [1B tableCount]
                   tableCount × [1B typeLen][type]        // string table: MIME types

#decode(bytes):    <type table, via #typeTable>
                   [1B entryCount]
                   entryCount × [1B tableIndex]?[4B dataLen BE][data]   // index omitted when tableCount == 1
```

`source` reads a route's frame inline — a type, then the payload (the rest of the buffer) — and builds one `Pathless.#Value`. `#decode` reads a bundle's `payload`: it delegates the shared type table to `#typeTable` (which returns the parsed types and the offset where entries begin), then reads one length-prefixed entry per child, each becoming a `#Value`. An entry typed `application/x-bundle` carries a further sequence as its `data`; the `Value` constructor decodes it into `children` (see below). `entryCount` lets the result array be allocated at its exact size up front, matching the wire format described in [mechanics.md](../mechanics.md#wire-format).

### `Value` (private, `Pathless.#Value`)

```js
{ type: string, data?: Uint8Array, children?: Value[], get url(): string }
```

A faithful mirror of `fx.Value`: a **leaf** holds its raw bytes in `data`; a **bundle** (`application/x-bundle`) holds its decoded child `Value`s in `children`. The constructor decides by `type` — a bundle's payload is one wire level, decoded into `children` right there via `#decode`; a leaf just keeps `data`. The type travels in-band, so a `Value` is self-describing and the transport `Content-Type` is irrelevant.

- `url` — a `blob:` object URL (`URL.createObjectURL`) for a leaf you render (an image, etc.), memoized on first access; never computed for entries only read as bytes (e.g. JSON).
- `children` — the per-file `Value`s of a bundle; server-side this is `fx.Value.Children`. Because a route's `Value` is cached per path, a bundle reused across spaces yields the same child `Value`s (and thus the same `url`s).

The node's `type` tells you which side applies, exactly as it does server-side (`Data` xor `Children`).

### Boot sequence

```
DOMContentLoaded
  → new Pathless()               // constructs cache, calls init()
    → source('')                 // fetch the root route → one bundle Value
      → root.children            // [universe, frames, panels]
    → exec(#universe, universe.data)    // injects & runs universe.html's script
      → new Universe(pathless, pathless.frames, pathless.panels)
    → universe.init()                   // renders layout 0
```

`exec` running `universe.html`'s script is what defines `Universe`/`Input`/`Panel` and attaches `pathless.universe`, `pathless.input`, `pathless.universe.panel` — this only happens after the root fetch resolves, so nothing here exists until the wire round-trip completes once.

---

## `universe.html` — layout, state, input

Markup: three space containers (`#zero`, `#fx`, `#one`) — `fx` and `one` start `hidden`. Script wraps everything in a block (`{ ... }`) so re-injection via `exec` never redeclares.

### `Universe`

Owns the three spaces, the frame pool, per-(frame, space) state, and layout/navigation.

| Static     | Value                    | Meaning                                                          |
| ---------- | ------------------------ | ---------------------------------------------------------------- |
| `ids`      | `['zero', 'fx', 'one']`  | space element ids, index-aligned with `spaces`                   |
| `layouts`  | `[null, [0,1], [0,1,2]]` | which space indices are visible at layout level `l`              |
| `variants` | `[1, 2, 4]`              | how many rotations (`v`) exist at each layout level, for `cycle` |

| Member                         | Signature                                | Behavior                                                                                                                                                                                                                             |
| ------------------------------ | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `space`                        | getter → `{ el, frame }`                 | the currently focused space entry                                                                                                                                                                                                    |
| `frame`                        | getter → `HTMLElement`                   | the focused space's content root — `space.el.querySelector('div')`; always the first (and only) `<div>`, since every frame's markup is `<style>` → one root `<div>` → `<script>`                                                     |
| `visible(l = state.l)`         | `(number?) => number[]`                  | space indices visible at layout `l` (layout 0 → just the focused index)                                                                                                                                                              |
| `init()`                       | `() => void`                             | resets every space to frame 0 and renders layout 0 (the frame/panel pools were passed to the constructor)                                                                                                                            |
| `read(i = focused)`            | `(number?) => Map`                       | per-(frame, space) state map, created on first access; keyed by `` `${frame}:${spaceId}` ``                                                                                                                                          |
| `write(key, val, i = focused)` | `(string, any, number?) => void`         | `read(i).set(key, val)`                                                                                                                                                                                                              |
| `pin(i = focused)`             | `(number?) => {i, read, write}`          | snapshots the focused index once; see [State semantics](../mechanics.md#state-semantics)                                                                                                                                             |
| `sync(...indices)`             | `(...number) => void`                    | re-executes the current frame into each given space (or all visible spaces); temporarily walks `state.focused` through each index so `p.universe.space`/`frame` resolve correctly mid-render; refreshes the keyboard panel afterward |
| `layout(l, v = 0)`             | `(number, number?) => void`              | sets `data-layout`/`data-variant` on `#universe` (drives the CSS grid), shows/hides spaces per `visible(l)`, then `sync()`s all of them. Non-zero layouts are remembered in `state.prev` for `cycle`'s single-layout toggle          |
| `cycle(l)`                     | `(number?) => void`                      | reserved-key handler (`1`/`2`/`3`): from single-space, restores the last multi-space layout if any; from a multi-space layout, re-pressing the same key rotates `variant`, pressing a different one switches layout at variant 0     |
| `nav(dir, i = focused)`        | `(number, number?) => void`              | advances that space's `frame` index by `dir` (wrapping), re-syncs just that space                                                                                                                                                    |
| `focus()`                      | `() => void`                             | moves `state.focused` to the next visible space index (`tab`)                                                                                                                                                                        |
| `at(x, y, rect)`               | `(number, number, DOMRect) => [nx, ny]`  | normalizes a point to `[-1, 1]` within `rect`                                                                                                                                                                                        |
| `press(x, y)`                  | `(number, number) => {i, close} \| null` | finds which visible space contains `(x, y)`, focuses it, and returns a `close(x, y)` closure that classifies the full gesture on release                                                                                             |
| `classify(a, b)`               | `([nx,ny], [nx,ny]) => string \| null`   | `tap` vs `swipe` vs `null`, based on `tap`/`swipe` distance thresholds and dominant axis (see below)                                                                                                                                 |

**Gesture classification** (`tap = 0.1`, `swipe = 0.25`, both in normalized `[-1,1]` units):

- total displacement `< tap` → `'tapLeft'` / `'tapRight'` (by end x sign)
- vertical dominant, or displacement `< swipe` → `null` (never hijacks vertical scroll)
- otherwise → `'swipeLeft'` / `'swipeRight'` (by dx sign)

### `Input`

Resolves pointer and keyboard events into one event model, shared by touch and physical keys.

| Static         | Meaning                                                                                           |
| -------------- | ------------------------------------------------------------------------------------------------- |
| `shellNames`   | reserved key → `{ fn(universe), desc?, touch? }` (layout switches, `tab`, `q`/`e` nav, `z` panel) |
| `gestureNames` | derived from `shellNames`: `touch` alias (e.g. `swipeLeft`) → same handler                        |

| Member                                    | Behavior                                                                                                                                                                                                                                       |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| constructor                               | binds `pointerdown`/`pointerup`/`pointercancel` on `universe.el` (passive), `keydown`/`keyup` on `window`; also installs a document-level double-tap-to-zoom blocker                                                                           |
| `handleEvent(e)`                          | single DOM listener entry point — dispatches to `this[e.type]`, then `dispatch()`s the result                                                                                                                                                  |
| `pointerdown`/`pointerup`/`pointercancel` | track one in-flight gesture per `pointerId` via `universe.press`/`close`                                                                                                                                                                       |
| `keydown`/`keyup`                         | normalize to `{ i, name: key.toLowerCase(), phase, event }`                                                                                                                                                                                    |
| `dispatch({name, phase, i, event, ...g})` | reserved names (`shellNames`/`gestureNames`) run immediately and flash the keyboard panel; otherwise looks up a per-space binding registered via `bind()` — function bindings fire once on non-`up`, object bindings dispatch to `.down`/`.up` |
| `bind(binds)`                             | replaces the entire binding table for the **currently focused space**; see [`p.input.bind`](../mechanics.md#pinputbindbinds)                                                                                                                   |

### `Panel`

A single hidden strip (`#panel`) below the universe, independent of the three spaces.

| Member         | Behavior                                                                                      |
| -------------- | --------------------------------------------------------------------------------------------- |
| `init(frames)` | stores the panel frame pool, shows the frame matching the focused space's current frame index |
| `toggle()`     | flips `el.hidden` (bound to reserved key `z`)                                                 |
| `show(i)`      | `exec`s `frames[i % frames.length]` into `el`                                                 |

### Wiring

At the end of the script:

```js
pathless.universe = new Universe(pathless, pathless.frames, pathless.panels);
pathless.input = new Input(pathless);
pathless.universe.panel = new Panel(pathless);
```

`Panel` is attached as `universe.panel` (not `pathless.panel`) — every reference elsewhere (`Input.dispatch`, `Universe.init`) goes through `pathless.universe.panel`.

---
