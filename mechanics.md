# Mechanics

Pathless is a library. It is constructed in two phases:

1. Establish the point of observation.
2. Register and serve the experience observed from it.

Templates and applications live outside this module. `pathless` provides the runtime, `frames` provides ready-made frames, and a `main` package composes them.

## Composition

`Pathless` embeds the `fx.Fx` interface:

```go
type Pathless struct {
	fx.Fx
}

func NewPathless(args ...string) *Pathless {
	z := zero.NewZero(args...)
	return &Pathless{Fx: fx.NewFx(z)}
}
```

`NewPathless` constructs `Zero` first, then hands it to `Fx`. Because `Fx` is embedded, its methods are promoted onto `Pathless`:

```go
type Fx interface {
	Input(string, ...bool) ([]*One, error)
	Frame(string, ...bool) error
	Save(string, ...bool) ([]byte, error)
	Start()
}
```

Four methods are the entire public surface. Everything else — templates, markdown rendering, slide decks — is built on top of them by other modules.

## Zero

`zero.NewZero` establishes the point of origin.

It embeds two resources into the Go binary:

- `pathless.html`: the browser shell and payload decoder
- `universe.html`: the observable runtime delivered through the circuit

During construction, Zero:

1. Selects local or hosted shell and circuit URLs.
2. Renders `pathless.html` with the circuit URL.
3. Compresses the rendered shell once.
4. Registers the shell handler.
5. Starts the shell server on port `1000` in a goroutine.

The shell payload is closed over by its HTTP handler. Rendering and compression do not occur per request. Any request that is not exactly `/` is redirected to `/` — the shell has no paths.

`NewZero` accepts zero or two arguments. Zero arguments is local mode:

```text
Shell:   http://localhost:1000
Circuit: http://localhost:1001
```

Two arguments are the shell and circuit domains, served as `https://`. Any other argument count panics.

`Zero` exposes `UniverseHTML` and `PathlessURL`. `PathlessURL` is the origin `Fx` allows through CORS — `*` in local mode.

## Fx

`fx` owns the experience assembled above Zero:

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

Its responsibilities are:

- Read local and remote resources into `One` values.
- Build executable HTML frames and panels.
- Retain named routes for on-demand payloads.
- Encode the bootstrap and route payloads.
- Serve the circuit on port `1001`.

The concrete type is unexported. Consumers hold the `Fx` interface.

## One

There is one data type. `One` is a node in a tree:

```go
type One struct {
	Name string
	Type string
	Data []byte
	Ones []*One
}
```

A file is a leaf. A directory is a node whose `Ones` are its children. A frame is a leaf whose `Type` is `text/html`. A payload is just `[]*One`.

Source location and cardinality do not create separate APIs, and nesting does not create a second representation. Consumers decide whether they need one node, a list, or a subtree.

## Input

```go
Input(path string, public ...bool) ([]*One, error)
```

`Input` accepts:

- An HTTP or HTTPS URL
- A local file
- A local directory

A URL or file returns one entry. A directory returns its children, recursively.

When `public` is true, the result is also retained in `routes` under a derived name — the URL or file base name, or the directory's own name. Retained routes are encoded and served at `/<name>` by `Start`.

A URL response is first attempted as an encoded payload via `decode`. If it decodes, those entries are returned directly, so one Pathless instance can consume another's payload. Otherwise the raw bytes become a single entry.

### Directory processing

`readDir` reads a directory, then:

1. Reads each file into a `One` via `readFile`.
2. Recurses into subdirectories, attaching children as `Ones`.
3. Detects each file's MIME type by extension, falling back to content sniffing.
4. Strips extensions from names that stay unique once shortened.
5. Applies an optional `sequence` file.

A `sequence.txt` file defines presentation order by listing normalized entry names line by line. The sequence entry is consumed and removed from the result. Unlisted entries keep their relative order afterward.

Names beginning with a dot keep their extension (`.gitignore`, `.env.local`).

## Frames and panels

```go
Frame(source string, panel ...bool) error
```

`source` is either literal HTML — detected by a leading `<` — or a path or URL resolved through `Input`, which must produce exactly one entry. When `panel` is true the result is appended to `panels`, otherwise to `frames`. Registration order is presentation order.

Every frame passes through `build`, which:

1. Merges multiple style blocks into one.
2. Merges script blocks into one block wrapped in an isolated scope.
3. Returns a `text/html` entry.

Script bodies that are not already a block are wrapped in braces before merging, so frames do not leak identifiers into each other.

Frames are rendered into observable spaces. Panels are rendered into the panel region.

A frame may call `pathless.source(route)` to obtain a public route. This keeps the bootstrap small while allowing frames to load typed resources as needed.

## Start

```go
func (f *fx) Start() {
	f.handle("/", encode(f.universe()))
	for key, payload := range f.routes {
		f.handle("/"+key, encode(payload))
	}
	http.ListenAndServe(":1001", f.cors(f.mux))
}
```

The bootstrap payload is positional:

```text
0: universe HTML
1: frames   (as Ones)
2: panels   (as Ones)
```

Every public route is encoded at startup as well. The compressed bytes are retained in memory and written directly for each request, so the served experience is a startup snapshot. Register everything before `Start`.

`Start` blocks, and panics if the circuit port cannot be bound.

## Save

```go
Save(key string, binary ...bool) ([]byte, error)
```

`Save` encodes an already-registered route and returns the wire bytes. When `binary` is true it also writes them to a file named for the route. This makes a payload publishable as a static artifact — object storage, a CDN — that another instance can consume through `Input`.

## Payload protocol

A payload is `[]*One`, encoded recursively:

```text
one-count
name  type  data  child-count  <children...>
name  type  data  child-count  <children...>
...
```

Counts and field lengths are unsigned varints, so small values need only one byte of framing. There are no type tags; every node has the same shape, and a zero child count terminates a leaf.

Payloads are gzip-compressed at best compression when encoded, which happens once per route at registration or in `Save`.

`decode` is the inverse and is defensive: it bounds nesting depth, rejects entry counts larger than the remaining bytes could hold, rejects field lengths that exceed the remainder, and rejects trailing bytes.

Responses are served as `application/octet-stream` with `Content-Encoding: gzip`. CORS allows `GET` and `OPTIONS` from `Zero.PathlessURL`.

## Client decoding

The shell fetches payloads from the circuit and decodes them into JavaScript values.

The decoder maintains one cursor over the response bytes:

- `uv()` reads an unsigned varint.
- `field()` reads one length-prefixed byte field.
- Each node reads name, type, data, then recurses over its child count.
- A node becomes `{ name, type, data, ones }`, with `data` as a `Uint8Array`.

The corrected field reader decodes the length before advancing the cursor:

```js
const field = () => {
	const length = uv();
	return bytes.slice(position, (position += length));
};
```

`source(path)` caches the decoded promise by route. Failed requests are removed from the cache so a later request can retry.

## Bootstrap

Client initialization loads one payload:

```js
const [universe, frames, panels] =
	await pathless.source('');
```

The universe HTML is executed inside `#universe`.

Frame and panel data are stored directly on `pathless`:

```js
pathless.frames
pathless.panels
```

The executed universe creates:

```js
pathless.universe
pathless.input
```

`Universe` references frames and panels through `pathless`, preserving one canonical owner.

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
