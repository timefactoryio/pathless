# pathless

Pathless serves a pathless browser shell and a circuit of typed content trees.
Frames occupy up to three observable spaces; panels occupy a separate region.
Templates and application-specific content live outside this module.

This is the reference for the current Go and browser APIs. See
[mechanics.md](mechanics.md) for the private implementation, wire protocol,
and initialization sequence.

## Quick Start

```go
package main

import (
        "log"

        "github.com/timefactoryio/pathless"
)

func main() {
        p := pathless.NewPathless()
        if err := p.Frame(`<main>Hello</main>`); err != nil {
                log.Fatal(err)
        }
        p.Start()
}
```

Open `http://localhost:1000`. The circuit listens on `:1001`. Construction
starts the shell server; `Start` starts the circuit and blocks. Register all
content before calling `Start`.

## Go API

### Pathless and NewPathless

```go
type Pathless struct {
        fx.Fx
}

func NewPathless(args ...string) *Pathless
```

The embedded `fx.Fx` interface promotes `Input`, `Frame`, `Save`, and `Start`
onto `*Pathless`. Use the constructor: a zero-value `Pathless` has no method
implementation.

| Arguments                              | Shell origin / circuit CORS | Circuit URL                   |
| -------------------------------------- | --------------------------- | ----------------------------- |
| None                                   | Shell on `:1000`; CORS `*`  | `http://localhost:1001`       |
| `"example.com", "circuit.example.com"` | `https://example.com`       | `https://circuit.example.com` |

Any other argument count panics. Hosted arguments are domains, not full URLs.
They configure origins, not listeners or TLS: both servers still listen over
HTTP on ports `1000` and `1001`. Supply HTTPS through external infrastructure.
In local mode, the browser substitutes its current hostname for `localhost`
in the circuit URL.

Construction renders and compresses the shell and starts its server in a
goroutine. Non-root shell paths and nonempty queries receive a `303` redirect
to `/`. Server errors panic. There is no shutdown, context, listener, port,
or HTTP-client configuration API.

### Fx and One

Exported by `github.com/timefactoryio/pathless/fx`:

```go
type Fx interface {
        Input(string, ...bool) ([]*One, error)
        Frame(string, ...bool) error
        Save(string, ...bool) ([]byte, error)
        Start()
}

func NewFx(z *zero.Zero) Fx

type One struct {
        Name string
        Type string
        Data []byte
        Ones []*One
}
```

`NewFx` wraps a constructed `Zero`, initializes collections/routing, and uses
an HTTP client with a ten-second timeout. Normally use `NewPathless` instead.

`One` represents a content node: sibling name, MIME type, bytes, and child
nodes. A file is normally a leaf; a directory has children and empty type/data.
The representation allows both data and children. Payloads are slices of nodes.

Each variadic boolean below defaults to false. Only its first value is used;
additional values are ignored.

### Input

```go
Input(path string, public ...bool) ([]*fx.One, error)
```

Reads a local file, a directory recursively, or a URL starting with `http://`
or `https://`.

| Input     | Returned nodes                             | Public route key                                       |
| --------- | ------------------------------------------ | ------------------------------------------------------ |
| File      | One leaf                                   | Filename without its final extension                   |
| Directory | Its children, including directory subtrees | Directory basename                                     |
| URL       | Decoded payload, or one raw resource       | Original URL path basename without its final extension |

`p.Input("./images", true)` retains a route named `images`, available as
`pathless.source('images')`. Without true it only reads. Reusing a route key
replaces its retained slice. Returned nodes and retained nodes are shared,
not deep copies. Registration/mutation is not synchronized for concurrent use.

Local MIME types use the file extension with content sniffing as fallback.
Remote responses must be `2xx`; their bytes are first tried as a gzip Pathless
payload. Otherwise they become one node, using the final response URL for its
name and extension/sniffed MIME type, not its `Content-Type` header. Route
naming uses the original URL. An empty URL route name is an error even for
nonpublic input. See the transport caveat in [mechanics.md](mechanics.md).

Within each directory, names lose their final extension only when the shortened
name is unique among siblings. This also applies to directory node names.
Dot-prefixed names retain their extensions; dotfiles are not filtered.
Entries initially follow `os.ReadDir` filename order.

A normalized entry named `sequence` supplies newline-separated normalized
sibling names. Matching entries come first in listed order, duplicate/unknown
names are ignored, and unlisted entries retain their relative order afterward.
The sequence entry is consumed. Each directory is processed independently.

Failures generally return errors with source context. Current exception: if
a single-file read fails after a successful `os.Stat`, `Input` can panic by
accessing the failed read's nil result before checking the error.

### Frame

```go
Frame(source string, panel ...bool) error
```

Registers a frame, or a panel when true, in registration order. Trimmed source
starting with `<` is literal HTML; otherwise `Input(source)` must return
exactly one node. Zero/multiple nodes return an error. Data is processed as
HTML and the output type becomes `text/html`, without MIME validation.

```go
if err := p.Frame("./home.html"); err != nil {
        log.Fatal(err)
}
if err := p.Frame(`<nav>Controls</nav>`, true); err != nil {
        log.Fatal(err)
}
```

The builder merges multiple exact, attribute-free `<style>` blocks and merges
exact `<script>` blocks into scoped blocks at the end. It is not a general
HTML parser, sanitizer, or module loader. Frame scripts share the shell's
document and origin; only load trusted code.

### Save

```go
Save(key string, binary ...bool) ([]byte, error)
```

Returns gzip wire bytes for a route previously registered by `Input(..., true)`.
Use the route key without a leading slash. Unknown keys return an error.
With true it also writes a file named exactly `key`, creating it with mode
`0644` or overwriting it. Parent directories are not created; write errors
are returned. It does not export the whole application or its frame collection.

Saved payloads can be hosted remotely for another instance's `Input`. Local
file input does not attempt payload decoding.

### Start

```go
Start()
```

Encodes the bootstrap and all public routes, registers handlers, and serves
the circuit on `:1001`. It blocks and panics on server failure. Call once,
after registration. Served bytes are startup snapshots, not live resources.

Responses use `application/octet-stream` and `Content-Encoding: gzip`.
CORS advertises `GET, OPTIONS`, allows the configured shell origin (or `*`
locally), and answers `OPTIONS` with `204`. Other methods are not rejected.
CORS is not authentication. `http.ServeMux` root catch-all semantics apply:
unknown paths can return the bootstrap instead of `404`.

### Zero

Exported by `github.com/timefactoryio/pathless/zero`:

```go
type Zero struct {
        UniverseHTML []byte
        PathlessURL  string
        // Unexported server configuration.
}

func NewZero(args ...string) *Zero
```

`NewZero` configures origins using the same constructor arguments, embeds the
shell/universe, and starts the shell server. `UniverseHTML` supplies the
bootstrap runtime. `PathlessURL` supplies circuit CORS; changing it does not
rerender the already-built shell. Pass the instance to `fx.NewFx` for low-level
composition. These are construction facilities, not extra registration APIs.

## Browser API

### Lifecycle and Nodes

The shell in [zero/zero.html](zero/zero.html) creates `window.pathless` and calls
its asynchronous `init()`. Initialization loads the bootstrap, stores frames
and panels, executes [zero/one.html](zero/one.html), and syncs visible content.
The payload constructs a Universe without storing it on `pathless`; its constructor exposes `frame`,
`x`, `y`, `navigate`, `layout`, `focus`, and `togglePanel` directly on `pathless`
before calling `sync()` to execute visible frame scripts.
Together with Pathless's `source` and `get` methods, these form the frame-facing
core API. Control methods are bound to Universe; `frame` is a live read-only
getter. Pathless extends native `EventTarget`, so
all components share its standard event methods. There is no separate ready
event or public ready promise.

Decoded nodes are `{ name, type, data, ones }`. Names/types are strings and
`ones` is an array. Data is text when `type.startsWith('text/html')`, otherwise
a `Uint8Array`. Other text formats require `TextDecoder`.

### Pathless Members

| Member                                                     | Contract                                                        |
| ---------------------------------------------------------- | --------------------------------------------------------------- |
| `el`                                                       | Root `#universe` element.                                       |
| `frames`, `panels`                                         | Decoded node arrays in registration order, assigned by `init`.  |
| `universe`                                                 | Running `Universe` instance, retained as a compatibility alias. |
| `frame`                                                    | Live getter returning `{ el, state }` for the current frame.    |
| `status`                                                   | System-owned Map from field names to getter functions.          |
| `get(field)`                                               | Reads one registered field; unknown fields return `undefined`.  |
| `get()`                                                    | Frozen snapshot of all registered fields.                       |
| `addEventListener`, `removeEventListener`, `dispatchEvent` | Native EventTarget methods on Pathless.                         |
| `x(left, right)`                                           | Registers left/right tap callbacks in the current space.        |
| `y(up, down)`                                              | Registers up/down tap callbacks in the current space.           |
| `navigate`, `layout`, `focus`, `togglePanel`               | Frame controls bound to Universe; contracts below.              |

`pathless` owns frames/panels; `Universe` reads them directly. They are mutable
implementation state, not managed collections. Replacing/shrinking them does
not reset existing space indexes.

`el`, `frames`, `panels`, `universe`, `status`, `exec`, and `init` remain accessible for
system bootstrap and internal collaboration, but are not part of the
frame-facing core API. Universe's other state and `sync` are not exposed as
flat Pathless members.

#### source

```js
await pathless.source(path, blobs = false)
```

Fetches `${circuit}/${path}` and resolves to decoded top-level nodes. Use `''`
for bootstrap or a route key without a leading slash. Requests, including
in-flight promises, are cached by path. Rejections remove that cache entry
for retry. HTTP error statuses are not explicitly checked. No refresh or
eviction API is provided.

With true it returns object URL strings, cached by node identity, for each
top-level node's data/type. It does not traverse children or revoke URLs.
Use binary leaves rather than directories or already text-decoded HTML.

```js
const { el } = pathless.frame;
const [imageURL] = await pathless.source('photo', true);
el.querySelector('img').src = imageURL;
```

#### System Notifications

Pathless owns `status`, a Map keyed by field name. Universe registers getter
functions for numeric `layout`, `variant`, and `focused`, boolean `panelOpen`,
and a frozen `visible` index array. `pathless.get(field)` evaluates only that
field's getter; unknown fields return `undefined`. Getters read current system
state rather than maintaining a second copy. The registry is internal system
state; frames read fields through `get`, not by mutating the Map.

```js
const layout = pathless.get('layout');
const visible = pathless.get('visible');
```

Indexes `0`, `1`, and `2` identify main spaces; `3` identifies the panel.
Universe owns visibility selection. `pathless.get()` builds a frozen snapshot
from all registered fields without a fixed status-object schema.

Universe dispatches native `CustomEvent('change', { detail: status })` events
on Pathless after layout, focus, and panel transitions finish measuring and
syncing. Unchanged commands do not notify. Navigation indexes and frame-owned
Maps are not part of this status. Internal `sync()` does not publish events.

The notification mechanism belongs to the system. Frames do not register their
own event listeners and need no listener cleanup. System-managed frame binding
is outside the scope of this API change. System listeners receive future events
only; use `get(field)` or `get()` explicitly for initial state.
Notifications are synchronous and do not wait for asynchronous frame resources.

#### exec, init, and decode

```js
pathless.exec(el, html)
await pathless.init()
Pathless.decode(bytes)
```

`exec` replaces the target's children with a contextual HTML fragment. Inline
scripts execute on insertion. It returns the Pathless instance. It does not sanitize and
does not await external scripts or asynchronous work. It does not dispose
replaced frames' timers, listeners, or observers.

`init` loads bootstrap, stores frames/panels, and executes universe HTML.
Universe installs the browser API and then calls `sync()` at the end of its
constructor to fill visible spaces, skipping empty collections.
`init` is called once by the shell, not an
idempotent reset.

Static `Pathless.decode(bytes)` synchronously decodes a decompressed wire
`Uint8Array` into nodes. It does not accept gzip directly and assumes trusted,
valid input. Normal HTTP gzip decoding is performed by the browser before
`source` reads `response.bytes()`.

### Universe State

The members below describe internal Universe state; the instance is not stored
on `pathless`, but `frame` is exposed directly. Status fields live in Pathless's Map, not
in a Universe status getter. Do not construct another `Universe`,
which would install extra observers/listeners. The zero-argument constructor
depends on `pathless` and the inserted space elements.

| Member            | Meaning                                                  |
| ----------------- | -------------------------------------------------------- |
| `el`              | Root element shared with `pathless.el`.                  |
| `spaces`          | Main records in `zero`, `one`, `two` order.              |
| `panel`           | The `three` space record.                                |
| `width`, `height` | Last measured root border-box dimensions in CSS pixels.  |
| `space`           | Currently rendering space, otherwise focused main space. |
| `frame`           | `{ el, state }`: rendered element and persistent Map.    |
| `visible`         | Visible space records derived from keyed system fields.  |

A space has `el`, zero-based `frame`, `states`, `bindings`, and measured
`origin: { x, y }`, the space's center in the shared normalized plane.
State maps are lazily allocated by the `frame` getter, persist per
space/frame-index pair, and survive navigation/rerenders. DOM and bindings
are replaced on rendering. Spaces and the panel navigate independently.

During synchronous frame execution, `space` and `frame` resolve against that
rendering space. After an `await` or in a later callback, they resolve against
the focused main space instead. Capture `const { el, state } = pathless.frame`
during synchronous initialization and retain those references for later callbacks.

Universe passes itself to a new Pointer controller but does not retain it as a public
property. Universe owns binding registration through its `x` and `y` methods;
Pointer reads those bindings when dispatching gestures.

### Rendering and Navigation

```js
pathless.navigate(direction)
pathless.focus(direction = 1)
pathless.togglePanel()
```

- Internal `universe.sync()` fills visible spaces lacking a first element child. Explicit records
    force rerendering. It skips empty collections and resets bindings before
    execution. Frames should have an element root for populated-space detection.
- `navigate` adds an integer direction, wraps within the space's frame/panel
    collection, and rerenders. Use `1`/`-1` for next/previous; empty collections
    do nothing.
- `focus` cycles visible main spaces, excluding the panel. If the starting
    space is absent, it chooses the first/last according to direction.
- The internal write-only `universe.focused` setter takes a member of `spaces`, not an index.
    It ignores nonmembers, updates focus, measures, and syncs.
- `togglePanel` changes visibility, measures, and syncs. No panels means no-op.
    Reopening a rendered panel preserves its DOM.

The public controls are bound to the Universe instance and return no result.
They resolve the current space internally; system callers can supply a space
record explicitly to `navigate` and `focus`. Arguments are not generally validated.

### Layout

```js
pathless.layout(layout, variant)
```

| Layout | Variant | Main arrangement                        |
| ------ | ------- | --------------------------------------- |
| `0`    | `0`     | Focused space only                      |
| `1`    | `0`     | `zero` left, secondary right            |
| `1`    | `1`     | `zero` above, secondary below           |
| `2`    | `0`     | `zero` left; `one` above `two` right    |
| `2`    | `1`     | `zero` above; `two` left of `one` below |
| `2`    | `2`     | `two` above `one` left; `zero` right    |
| `2`    | `3`     | `one` left of `two` above; `zero` below |

For layout one, secondary is `two` when focused, otherwise `one`. Nonzero
variants wrap modulo `2 ** layout`. Omitting a variant advances it when
remaining in a layout, or selects zero when changing layouts.

`layout(0)` from a split layout saves the pair and enters single-space mode;
calling again restores it. Without a saved split, layout zero from zero does
nothing. `layout()` cycles a split layout's variants or attempts restore from
single-space mode. Supported layout numbers are `0`, `1`, `2`.

Root state uses string-valued `data-layout`, `data-variant`, `data-focused`,
and class `panel`. An open panel occupies `15dvh` below main spaces. Use the
methods rather than direct dataset mutations to keep geometry/rendering aligned.

### Pointer Bindings and Gestures

```js
pathless.x(onLeft, onRight);
pathless.y(onUp, onDown);
```

Each call replaces that axis's callbacks in the current space. Bindings reset
on render, so register in the synchronous frame script. Missing callbacks
are no-ops. Callbacks receive `(start, end, space)`; returns are ignored and
promises are not awaited. The full Universe element defines the shared plane:
left/right are `-1`/`1`, top/bottom are `1`/`-1`, and its center is `(0, 0)`.
Coordinates are clamped to `[-1, 1]` on each axis, including captured events
outside the element. Each axis is normalized independently, so equal coordinate
distances need not represent equal pixel distances. The conversion assumes
the root starts at viewport position `(0, 0)`, as the shell's fixed grid does.

Only a primary pointer with button zero is accepted, one gesture at a time.
Pressing a visible main space focuses it; the root captures the pointer.
Default-prevented releases or releases after the pressed space becomes hidden
cancel the gesture. Precedence is:

1. Movement at least `0.1` on either normalized axis is a swipe. Horizontal
     dominance navigates: left is previous, right is next. Vertical dominance
     does nothing.
2. Otherwise, toggle the panel only when both endpoints have both coordinates
        within `[-0.05, 0.05]`. This central square in normalized coordinates spans
        5% of the root width and 5% of its height, or 0.25% of the plane's area.
3. Other taps choose a callback by the dominant offset from the pressed space's
     center. Ties select horizontal.

The controller has implementation members `state`, `universe`,
`down(event)`, `up(event)`, `cancel(event)`, `coordinate(point)`, and
`output(start, end)`. `coordinate` converts and clamps client coordinates with
measured root dimensions. `cancel` clears a matching pointer ID; the other handlers
implement the gesture state machine. `output` requires an active gesture.
Applications use `pathless.x` and `pathless.y`, not these handlers.

## Boundaries

- There is no `Text`, `Slides`, `Route`, or `pathless.input` API in this version.
    Compose templates externally and register their HTML with `Frame`.
- Frame scoping is not security isolation. HTML/scripts must be trusted.
- Servers, cached object URLs, state maps, and observers have no coordinated
    disposal API. Construction/startup are one-shot operations.
- Go reads/decompression have no explicit size limits; the browser decoder
    assumes valid payloads and performs fewer checks than the Go decoder.
