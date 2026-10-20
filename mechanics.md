# Mechanics

This guide describes the components behind the public API in
[README.md](README.md). Private names are implementation details, not additional
application contracts.

## Ownership

| Component          | Owns                                                                     |
| ------------------ | ------------------------------------------------------------------------ |
| Go `Pathless`      | Embedded `fx.Fx` interface                                               |
| `zero.Zero`        | Universe bytes, shell origin, private circuit URL and shell mux          |
| Private `fx`       | Zero reference, frames/panels, routes, circuit mux, HTTP client          |
| Browser `Pathless` | Root, resource/blob cache, decoded frames/panels, universe               |
| Browser `Universe` | Space records, geometry, bindings, layout/focus, public API registration |
| Browser `Pointer`  | Universe reference, active gesture, pointer-event dispatch               |

There is no separate Go `one` serving package: `fx.One` is the tree data type,
and `fx` encodes and serves circuit payloads.

## Construction

1. `NewPathless(args...)` calls `zero.NewZero(args...)`.
2. `NewZero` chooses local/hosted origins, exposes embedded universe bytes,
   and calls `Zero.pathless` to render/compress/register the shell.
3. `NewZero` launches the shell server on `:1000` in a goroutine.
4. `fx.NewFx` allocates routes, the circuit mux, and the HTTP client.
5. Application code registers content with `Input` and `Frame`.
6. `Start` encodes bootstrap/public routes, registers handlers, and blocks
   serving the CORS-wrapped circuit mux on `:1001`.

These are separate HTTP servers with no exposed shutdown handle. `Start`
does not watch files or dynamically register later content.

### Zero Internals

`pathlessHTML` embeds [zero/zero.html](zero/zero.html) as a string;
`universeHTML` embeds [zero/one.html](zero/one.html) as bytes. Editing either
file requires rebuilding the Go application.

`Zero.pathless` renders the shell with `html/template`, passing the private
`circuitURL` as `CIRCUIT`. `Zero.zip` compresses it using `gzip.BestCompression`.
The shell handler closes over these bytes: no per-request rendering/compression.
It redirects non-root paths/nonempty queries and otherwise serves gzip HTML.

Private `mux` owns shell routing. Exported `PathlessURL` supplies circuit CORS,
not the server's bind address. Hosted mode changes advertised HTTPS origins,
not the plain HTTP listeners.

## Content Pipeline

### fx State

```go
type fx struct {
    z      *zero.Zero
    frames []*One
    panels []*One
    routes map[string][]*One
    mux    *http.ServeMux
    client *http.Client
}
```

Collections preserve registration order; route keys derive from source paths.
Public input retains slices of pointers. Frame building shallow-copies the
node, replacing type/data but sharing children. Mutations are not synchronized.

### Input Helpers

| Helper                        | Role                                                                                                            |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `(*fx).url(source)`           | GET, reject non-2xx, read bytes, try `decode`, otherwise return a raw leaf.                                     |
| `detectType(path, data)`      | Extension-based MIME resolution, then content sniffing.                                                         |
| `readDir(path)`               | Read listing, call `processFiles`, apply `sequence`.                                                            |
| `processFiles(path, entries)` | Recurse into directories, read files, call `stripExtensions`.                                                   |
| `readFile(path, name)`        | Read bytes and construct a typed leaf with the supplied name.                                                   |
| `stripExtensions(entries)`    | Count shortened candidates, mutate only unique candidates.                                                      |
| `sequence(entries)`           | Consume the normalized `sequence` node and order siblings.                                                      |
| `baseName(source)`            | Extract URL path if applicable, take basename, preserve dot-prefixed names, otherwise drop the final extension. |

`baseName` returns empty for `.` or `/`. Sibling normalization includes
directories, is local to each directory, and starts from `os.ReadDir` order.
A direct file input instead gets its name from `baseName` immediately.

`sequence` trims overall data and individual lines, ignores unknown names,
and tracks node identity to suppress duplicate listed entries. Unlisted nodes
follow in their existing order. It triggers on normalized name `sequence`,
not on a special filesystem extension. A collision that preserves the name
`sequence.txt` prevents that entry from being consumed under that name.

### Remote Transport Caveat

`url` tries gzip-only `decode` on bytes read from the HTTP response body.
The standard Go HTTP transport can automatically decompress responses carrying
`Content-Encoding: gzip`, as Pathless circuit responses do. In that case it
passes bare wire bytes to `decode`, which rejects them; `url` falls back to
a raw resource. Direct circuit-to-circuit consumption is therefore not always
equivalent to fetching a saved gzip artifact served without content decoding.

### Frame Builder

`(*fx).build` uses case-sensitive regular expressions for exact tags:

```text
(?s)<style>(.*?)</style>
(?s)<script>(.*?)</script>
```

Multiple matching styles are concatenated and prepended; one style stays in
place. Matching scripts are removed and merged at the end. Bodies whose
trimmed source does not begin with `{` get block-wrapped, and the merged
script receives an outer block. Output is a shallow node copy with rewritten
data and `Type: "text/html"`.

Attributes, uppercase tags, module scripts, and external scripts are not
handled by the merger. Unmatched tags keep their original HTML behavior.
Block scoping does not isolate globals, DOM access, asynchronous work, or
privileges. This is not sanitization or general HTML parsing.

### Bootstrap and Serving

`(*fx).universe()` creates three positional nodes:

```text
0: { Name: "universe", Type: "text/html", Data: Zero.UniverseHTML }
1: { Ones: frames }
2: { Ones: panels }
```

The containers have otherwise zero-valued fields. Browser initialization
depends on position, not names.

`(*fx).handle(path, bytes)` closes over encoded bytes in a `ServeMux` handler,
writing octet-stream/gzip headers on every response. The root route is a
catch-all; route keys and patterns follow Go mux semantics. `Start` encodes
once per route at startup, while `Save` encodes on each invocation.

`(*fx).cors(next)` sets allowed-origin/method headers, returns `204` for
`OPTIONS`, and passes other requests through. It advertises rather than
enforces `GET, OPTIONS`. Compression is unconditional, without per-request
`Accept-Encoding` negotiation.

## Wire Protocol

After gzip decompression, a list is encoded as:

```text
unsigned-varint node count
repeat for each node:
    unsigned-varint name byte length; UTF-8 name bytes
    unsigned-varint type byte length; UTF-8 MIME type bytes
    unsigned-varint data byte length; raw data bytes
    child list (same format, starting with its count)
```

There is no version, magic prefix, or directory tag. A leaf ends in a zero
child count. Field lengths count bytes, not characters.

### Encoder

Private `encoder` embeds `io.Writer`. `writeOnes` writes counts, fields, and
children; `writeField` writes a byte length and bytes; `writeUvarint` uses
`binary.PutUvarint` with a fixed scratch array.

`encode` writes through best-compression gzip into an in-memory buffer. Internal
writer errors are not propagated. Nodes must be non-nil and the tree acyclic;
the encoder does not validate arbitrary user-created graphs.

### Go Decoder

`decode` opens gzip, reads its full decompressed contents, constructs a private
`decoder` embedding `*bytes.Reader`, calls `readOnes(0)`, and rejects trailing
bytes.

`readOnes` rejects depths above 1000 and node counts larger than remaining
minimum framing permits. It reads fields and recursively reconstructs nodes.
`readField` bounds lengths against remaining bytes and uses `io.ReadFull`.
These are framing checks, not resource limits: decompressed bytes are fully
read before validation, without a total-size cap.

## Browser Bootstrap

The shell defines the circuit URL and `Pathless`, then runs on `DOMContentLoaded`:

```js
window.pathless = new Pathless();
pathless.init();
```

Construction captures `#universe`. The asynchronous `init` loads `source('')`,
stores the two content collections, and calls
`this.exec(this.el, zero.data).sync()`. Insertion adds four spaces,
defines `Universe` and `Pointer`, and ends with:

```js
pathless.universe = new Universe();
```

Universe's constructor installs the bound methods and live `pathless.frame`
getter before returning. Inline insertion executes the assignment before
`exec` returns the Pathless instance, so chained `sync()` can render frames
with the API already installed. Sync skips empty collections without an outer
frame-count guard. `Pathless` has no Universe-specific getter or binding list;
there is no `pathless.input` object or stored ready promise.

### Pathless Private Components

`#cache` holds two kinds of entries: route strings map to fetch/decode promises;
decoded node identities map to blob URL strings. `source` deletes rejected
requests for retry and optionally maps top-level nodes to blob URLs. Each URL
lookup explicitly creates and stores a Blob URL only on a cache miss. There
is no invalidation, bounded retention, recursive conversion, or URL revocation.

Static `#decoder` is a shared `TextDecoder`. Static `#range` is a shared DOM
`Range`; `exec` creates a contextual fragment and inserts it with
`replaceChildren`, enabling inline script execution. It returns `this` for
chaining; it does not automatically sync or await asynchronous frame work.

`Pathless.decode` closes over a byte offset. Its local helpers are:

- `uvarint`: reads continuation-bit integers, coercing to unsigned 32-bit with
  `>>> 0`.
- `field`: reads length, advances offset, returns a `subarray` view.
- `ones`: reads the node count and uses `Array.from` to decode nodes in order,
  recursively decoding each child list before the next sibling. HTML data
  becomes strings; other data remains byte views.

Unlike the Go decoder, it does not explicitly reject truncation, trailing
bytes, excessive depth, or oversized varints. It assumes trusted valid wire
bytes and relies on browser HTTP gzip decompression.

## Universe Private Components

Construction creates three main space records and one panel record, each at
frame zero with empty states/bindings. It creates `Pointer(this)`, measures,
and observes the root border box with `ResizeObserver`.

The controller is retained by its bound event listeners, not a `universe.pointer`
property. The observer is constructed and observed directly without a stored
handle. There is no teardown API for either.

The constructor registers an explicit array of references to `x`, `y`,
`navigate`, `layout`, `focus`, `togglePanel`, and `sync`. Each becomes
`pathless[method.name] = method.bind(this)`, preserving the Universe receiver
even when called through Pathless or detached. This relies on method names
remaining intact if a future build adds minification.

`Object.defineProperty` registers `pathless.frame` with `get: () => this.frame`.
The arrow retains Universe as its receiver and delegates on every access,
rather than capturing a frame snapshot. The property has no setter and is
configurable so it can be registered again. This does not make initialization
an idempotent reset: constructing another Universe still adds observers/listeners.

| Private member                     | Purpose                                                      |
| ---------------------------------- | ------------------------------------------------------------ |
| `#previous`                        | Last split layout/variant, used for single-space toggling.   |
| `#rendering`                       | Temporary space context during synchronous script execution. |
| `#cycle(index, direction, length)` | Positive modulo wrapping.                                    |
| `#measure()`                       | Store root dimensions and normalize visible space origins.   |

Each `space.origin` is its bounding rectangle's center in the shared plane,
normalized using the root border-box dimensions, with positive Y upward.
The root fills the viewport at `(0, 0)`; measurement and pointer conversion
assume that placement rather than subtracting root offsets.
Hidden spaces are measured only when visible.
Resizes, focus/layout changes, and panel toggles trigger measurement. The
observer has no disposal method.

`space` chooses rendering context or focused main space. `frame` lazily creates
its state map at `states[frame]`. `visible` derives main visibility from root
dataset state and appends an open panel.

`sync` fills visible empty spaces unless explicit records are supplied. It
chooses `pathless.frames`/`pathless.panels`, skips empty collections, sets
`#rendering`, resets bindings, and executes data. `finally` clears rendering
context even on an exception. It does not clamp stale indexes or roll back
DOM. Async work runs after the rendering context has ended.

`navigate` wraps and rerenders. The `focused` setter, `layout`, and `togglePanel`
update root state, measure, and sync missing content. `focus` computes a visible
main target and delegates to the setter. Persistent state is per space/index;
bindings and DOM are rebuilt, not stored in that state automatically.

### CSS Components

The shell owns CSS; the universe payload supplies the space elements. A fixed
full-viewport grid uses `data-layout`/`data-variant` for grid areas,
`data-focused` for selected main-space visibility, and class `panel` for a
`15dvh` panel row. Main tracks use `minmax(0, 1fr)` to resist content overflow.

`.space` supplies flex sizing, padding, minimum-size constraints, clipping,
and a size container. Its first child flexes to fill it. The root is also a
size container and has a pointer-transparent border overlay. Global CSS
disables selection, native dragging, touch callouts, tap highlighting, root
touch actions, and overscroll.

## Pointer Private Machinery

`Pointer` receives its Universe instance in the constructor and holds it with
nullable `state`. Universe's regular `x(left, right)` and `y(up, down)` methods
assign callback pairs to the current space; Pointer only reads the bindings.
Construction binds direct handler references to the controller and listens
for root `pointerdown`, `pointerup`,
`pointercancel`, and `lostpointercapture`. The last two delegate to `cancel`.
There is no move listener or teardown API.

1. `down` filters nonprimary/nonzero-button/overlapping presses, finds the
   containing visible space, focuses a main space if needed, captures the
   pointer, and records `{ id, start, space }`.
2. `up` checks the active ID, cancels prevented releases or hidden spaces,
   otherwise computes the endpoint and calls `output`.
3. `cancel` clears only a matching pointer ID.
4. `coordinate` maps client coordinates using measured root dimensions:
  `x = 2 * clientX / width - 1`, `y = 1 - 2 * clientY / height`. Both are
  clamped to `[-1, 1]`, so captured events outside the element map to its edges.
5. `output` retains the pressed space/bindings/origin, clears gesture state,
  then uses early returns to select swipe, central-origin panel toggle, or
  directional tap.

The swipe threshold is `0.1` normalized units, with horizontal dominance
required for navigation; vertical-dominant swipes return without dispatching
a tap. Panel toggling requires the maximum absolute value of all four endpoint
coordinates to be at most `0.05`. Both press and release must therefore lie
in the central `[-0.05, 0.05]` square, not along either full axis. The region
spans 5% of root width and height and covers 0.25% of the plane. There is no
debug overlay in the payload.
Other taps compare offsets from `space.origin`, choosing horizontal on ties,
and call `(start, end, space)`. Returns are ignored and promises are not awaited.

Thresholds use the full root's normalized plane, not pixels or fractions of
an individual space. The axes scale independently. The panel does not become
the focused main space when pressed; the focus setter ignores nonmembers of
the main spaces.

## Observation

Universe provides the device-independent observation model:

- Three spaces
- Shared layouts and variants
- Focus
- Frame navigation
- Panel display
- Per-frame state
- Normalized pointer coordinates
- Tap and swipe classification
- Keyboard and gesture dispatch

Physical devices provide different screens and input mechanisms, but frames observe one Pathless universe.

A pointer, touch, or key event is normalized before a frame handles it. Screen dimensions affect the viewport without creating a separate application model.

Pathless therefore does not reproduce an application across platforms. It establishes one point of origin that any HTTP-capable screen can observe.

## Frames library

`github.com/timefactoryio/frames` is a separate module built entirely on the four `Fx` methods:

```go
type Frames struct {
	*pathless.Pathless
	*fx.Fx
}

func NewFrames(p *pathless.Pathless) *Frames
```

Its `Fx` holds the `*pathless.Pathless` it was given plus the embedded frame and panel templates, and exposes template constructors:

- `Home(logo, heading)` renders the home template. An SVG logo is inlined from its bytes, a local raster logo is registered as a public route and referenced by `data-src`, and a remote logo is referenced by `src`.
- `Text(path)` reads one entry, converts markdown to HTML, and renders it.
- `Slides(dir)` registers the directory as a public route and renders a template that fetches it by name.
- `Keyboard()` registers the embedded keyboard template as a panel.

Each constructor ends in `Frame(html)`. They are ordinary callers, not privileged ones.

## Application

An application is a `main` package that composes the two:

```go
func main() {
	p := pathless.NewPathless()
	f := frames.NewFrames(p)
	f.Home("https://zero.s3.timefactory.io/timefactory.svg", "the point of origin")
	f.Text("./theplan.md")
	f.Slides("https://zero.s3.timefactory.io/slides")
	f.Keyboard()
	f.Start()
}
```

`frames/one` is that demo. It demonstrates the libraries; it is not part of them.

## Deployment

Pathless requires one executable runtime.

It may run as:

- A compiled Go binary
- A container image
- A hosted service
- A native host embedding the runtime and a web view

Remote devices need no Pathless installation. They only require network access and a browser runtime.

The deployment contract is:

> Run one binary. Observe from any screen.

## Design boundaries

The architecture keeps several concerns deliberately separate:

- `Zero` establishes observation.
- `Input` converts resources into `One` values.
- `Fx` composes the experience and serves the circuit.
- `One` is both the domain type and the wire type.
- Frames interpret resources.
- Universe normalizes observation and interaction.

The library defines no templates and no content. Templates are a separate module, and applications are a third. Each layer depends only on the interface below it.

One recursive type covers a file, a directory, a frame, and a payload, so no conversion layer exists between what `Input` returns and what the circuit sends. Frames receive complete node metadata when identity and MIME type matter — associating slide images with descriptions, for example — without a second representation.

The result is a small origin from which applications can be assembled without inheriting a device vendor’s application model.
