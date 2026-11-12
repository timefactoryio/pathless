# pathless

Pathless serves a pathless browser shell and a circuit of typed resources.
Frames occupy up to three observable spaces; panels occupy a separate region.
Templates and application-specific content live outside this module.

This is the single reference for the backend Go API, frontend browser API,
and their shared transport. Templates and applications are separate modules.

- [Quick Start](#quick-start)
- [Go API](#go-api)
- [Browser API](#browser-api)
- [Transport](#transport)
- [Boundaries](#boundaries)

## Model

A universe contains three main spaces and one panel space. Each space owns its
navigation index and persistent frame state. Frames supply executable HTML;
layouts choose visible spaces and their geometry. Navigation replaces content,
not the page URL. Focus selects the default target for navigation and keyboard
commands; the panel is never included in the focus cycle.

## Quick Start

Use the Go version declared in [go.mod](go.mod), currently `1.27.3`, and add
the module with `go get github.com/timefactoryio/pathless`.

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
renders the shell; `Start` starts both servers and blocks. Register all
content before calling `Start`, including at least one main frame. The browser
initializes by rendering the first main frame and does not guard an empty list.

## Go API

### Pathless and NewPathless

```go
type Pathless struct {
        *one.One
}

func NewPathless(args ...string) *Pathless
```

The embedded `*one.One` promotes `Frame`, `Panel`, and `Frames`, along with
`Input`, `InputDir`, `Load`, `Encode`, and `Decode` through its embedded `fx.Fx`.
`Pathless` defines `Start` itself and starts that same `One` instance.
Use the constructor: a zero-value `Pathless` has no method
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

Construction renders the uncompressed shell without starting a server.
`Start` delegates both servers to `one`. Non-root shell paths and nonempty queries receive a `303` redirect
to `/`. Server errors panic. There is no shutdown, context, listener, port,
or HTTP-client configuration API.

### Fx and One

Exported by `github.com/timefactoryio/pathless/fx`:

```go
type Fx interface {
        Input(string) (N, error)
        InputDir(string) (N, error)
        Load(string) (N, error)
        Encode(...N) []byte
        Decode([]byte) (N, error)
}

func NewFx() Fx

type One struct {
        Name string
        Type string
        Data []byte
}

type N []*One

```

`NewFx` initializes an HTTP client with a
ten-second timeout. It creates no mux, registers no HTTP handlers, and starts
no servers. Normally use `NewPathless` instead.

`One` represents one resource: name, MIME type, and bytes. `N` is a flat
collection of resource pointers. Directories are not emitted as records;
nested resources use relative names such as `slides/first`.

`one.One.Frames()` returns `Encode(frames, panels)`: two consecutive collections, frames
first and panels second. Each has its own entry count, including zero for an
empty collection, preserving the boundary without modifying either collection.

Each variadic boolean below defaults to false. Only its first value is used;
additional values are ignored.

### Input

```go
Input(path string) (fx.N, error)
```

Reads a local file or a URL starting with `http://`
or `https://`.

| Input     | Returned resources                           |
| --------- | -------------------------------------------- |
| File      | One resource                                 |
| URL       | One raw resource                            |

`p.Input("./logo.png")` only reads and returns content. It does not retain resources
or register routes. Publishing imported assets through `one` is a separate
operation; local logo and slide helpers no longer publish their inputs implicitly.

Local MIME types use the file extension with content sniffing as fallback.
Remote responses must be `2xx`; their bytes become one raw resource, using the final response URL for its
name and extension/sniffed MIME type, not its `Content-Type` header. URL inputs
do not require a route name. `Input` never decodes payloads; use `Load` for encoded resources.

### InputDir

```go
InputDir(path string) (N, error)
```

Reads a local directory recursively into a flat collection with relative names.
Directory traversal is explicit: `Input` rejects directories, and `InputDir`
rejects files. It does not retain resources or register routes.

Within each directory, names lose their final extension only when the shortened
name is unique among siblings. This also applies to directory path segments.
Dot-prefixed names retain their extensions; dotfiles are not filtered.
Entries initially follow `os.ReadDir` filename order.

A normalized entry named `sequence` supplies newline-separated normalized
sibling names. Matching entries come first in listed order, duplicate/unknown
names are ignored, and unlisted entries retain their relative order afterward.
The sequence entry is consumed. Each directory is processed independently
before its contents are flattened. Sequencing a directory moves its files as
a group; empty directories produce no resources.

Read failures return errors with source context.

### Load

```go
Load(url string) (N, error)
```

Fetches an encoded payload from an HTTP(S) URL and returns its decoded resources
in order. Works with public or presigned S3 URLs and other HTTP object stores;
no AWS-specific configuration is required. HTTP, read, and decoding failures
return errors with URL context, without falling back to a raw resource.
It does not register or retain the loaded resources.

### Frame

```go
Frame(source string) error
Panel(source string) error
```

Registers a local file or raw URL resource returned by `Input(source)` as a frame
or panel, in registration order. Directory paths are not accepted.
These methods belong to `one.One`, which owns the frame and panel collections;
they are also promoted onto `Pathless`. The `fx` package only sources and encodes resources.
Every entry's MIME type must be
`text/html` (parameters such as `charset=utf-8` are allowed); other or malformed
types return an error without registering any entries from that call.

```go
if err := p.Frame("./home.html"); err != nil {
        log.Fatal(err)
}
if err := p.Panel("./controls.html"); err != nil {
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

Returns gzip wire bytes for a key in the retained route store. `Input` no longer
populates that store; explicit resource publication still needs separate wiring.
Use the route key without a leading slash. Unknown keys return an error.
With true it also writes a file named exactly `key`, creating it with mode
`0644` or overwriting it. Parent directories are not created; write errors
are returned. It does not export the whole application or its frame collection.

Saved payloads can be hosted remotely for another instance's `Load`. Local
file input does not attempt payload decoding.

### Circuit

```go
Circuit() map[string][]byte
```

Returns a snapshot of gzip-encoded payloads. The empty key contains JSON
`[frames, panels]`; other keys contain public resource payloads in the binary
wire format. Keys are content names, not URL paths. `one` adds leading slashes
when registering HTTP handlers.

### Start

```go
Start()
```

`Pathless.Start` passes the rendered shell and `Circuit()` snapshot to `one`,
which registers handlers and serves the shell on `:1000` and circuit on `:1001`.
It blocks and panics on server failure. Call once, after content registration.
Served bytes are startup snapshots, not live resources.

The circuit root uses `application/json`; public resources use
`application/octet-stream`. Both use `Content-Encoding: gzip`.
CORS advertises `GET, OPTIONS`, allows the configured shell origin (or `*`
locally), and answers `OPTIONS` with `204`. Other methods are not rejected.
CORS is not authentication. `http.ServeMux` root catch-all semantics apply:
unknown paths can return the bootstrap instead of `404`.

### Zero

Exported by `github.com/timefactoryio/pathless/zero`:

```go
type Zero struct {
        Origin []byte
}

func NewZero(circuitURL string, compressed ...bool) *Zero
```

`NewZero` uses a shared, pre-parsed template to
render the supplied circuit URL, shell, Pathless script, and Input script once
into the exported `Origin` bytes. Passing `true` gzip-compresses those bytes at
`gzip.BestCompression`;
omitting the flag or passing `false` keeps raw HTML. Only the first flag is used.
When serving compressed bytes, the HTTP layer must set `Content-Encoding: gzip`.
It provides no HTTP handling and starts no
server or listener. Rebuild the Go
application after editing embedded assets.

### One

Exported by `github.com/timefactoryio/pathless/one`:

```go
type One struct {
        fx.Fx
        Pathless *http.ServeMux
        Circuit  *http.ServeMux
}

func NewOne() *One

func (*One) Frame(string) error
func (*One) Panel(string) error
func (*One) Frames() []byte
```

`NewOne` initializes `fx.Fx`, empty frame/panel collections, and two muxes.
Construction starts no listeners and registers no handlers. Use the muxes with
your own HTTP servers, or call `Start` to listen on `:1000` and `:1001`.

```go
server := one.NewOne()
if err := server.Frame("./home.html"); err != nil {
        log.Fatal(err)
}
payload := server.Frames()
// Publish payload separately; frame registration does not register routes.
```

## Browser API

Frame scripts use `window.pathless`. All input registration and control methods
return that same object for chaining. `frame`, `source`, `exec`, and `init` have
the distinct return values documented below.

### Lifecycle and Members

The initial page includes [zero/zero.html](zero/zero.html),
[zero/universe.html](zero/universe.html), and [zero/input.html](zero/input.html).
On `DOMContentLoaded`, it creates `window.pathless` and calls `init()`. That
fetches frames and panels, constructs Input, then constructs Universe. Input
registration and controls are available before the first frame script runs.
Applications should initialize inside frame scripts; there is no ready event
or stored ready promise.

| Member             | Value                                                                           |
| ------------------ | ------------------------------------------------------------------------------- |
| `el`               | Root `#universe` element, with measured `width` and `height` in CSS pixels.     |
| `spaces`           | Four DOM elements in `zero`, `one`, `two`, `three` order; `three` is the panel. |
| `frames`, `panels` | Decoded node arrays in registration order, assigned by `init()`.                |

Spaces expose their DOM `hidden` flag and a measured normalized `origin: { x, y }`
when visible. Use controls rather than mutating visibility or root datasets.
Collections are runtime state, not supported dynamic registration APIs; replacing
them does not reset space indexes or stored state.

### Frame Context

```js
const { space, el, state } = pathless.frame();
const context = pathless.frame(pathless.spaces[0]);
```

`frame(space)` returns `{ space, el, state }`. The optional space must be an
element from `pathless.spaces`; it defaults to the synchronously rendering
space, otherwise the focused main space. `el` is its first element child, or
`null` before content exists. `state` is a lazily created `Map` retained per
space and frame index, including panel frames.

Navigation replaces DOM, not the state Map. Hidden spaces retain their DOM;
showing them again does not rerender populated content. Give each frame one
root element and register bindings in its synchronous script. Capture context
before an `await` or later callback: a fresh `frame()` call then follows focus,
not the frame that started the asynchronous work.

Save this example as a frame and register it with `Frame("./counter.html")`:

```html
<main>
        <button type="button">Increment</button>
        <output></output>
</main>
<script>
        const { el, state } = pathless.frame();
        const output = el.querySelector('output');
        const show = () => {
                output.textContent = state.get('count') ?? 0;
        };
        const change = (amount) => {
                state.set('count', (state.get('count') ?? 0) + amount);
                show();
        };
        pathless
                .click(el.querySelector('button'), () => change(1))
                .x(() => change(-1), () => change(1));
        show();
</script>
```

### Resources

```js
const nodes = await pathless.source('photos');
const { el } = pathless.frame();
const [imageURL] = await pathless.source('photos', true);
el.querySelector('img').src = imageURL;
```

`source(path, blobs = false)` fetches `${circuit}/${path}`. Use a public route
key without a leading slash. It resolves to top-level nodes with this shape:

```js
({ name: 'photo', type: 'image/png', data: new Uint8Array(), ones: [] })
```

Names and MIME types are strings. `ones` contains child nodes. Data is decoded
to a string only when `type.startsWith('text/html')`; other data remains a
`Uint8Array`. Use `TextDecoder` for other text formats.

With `blobs = true`, it instead resolves to object URL strings for top-level
node data and MIME types. It does not recurse into directories. Use binary
leaves for assets. Requests, including in-flight promises, are cached by path;
rejections remove that cache entry for retry. HTTP status codes are not checked.
Object URLs are cached by node identity, with no eviction or revocation API.

### Input Registration

```js
pathless.click(element, callback);
pathless.x(onLeft, onRight);
pathless.y(onUp, onDown);
```

`click` registers a delegated click callback on an element. It receives the
native event, with `this` set to the registered element for ordinary functions.
Clicks on descendants select the nearest registered ancestor below the root.
A later registration on the same element replaces the callback. Registered
click targets do not start directional gestures; native click behavior is not
automatically prevented. Default-prevented clicks are ignored.

`x` binds left/right taps and the `a`/`d` keys; `y` binds up/down taps and the
`w`/`s` keys. Each pair is registered against the current frame's root element,
as resolved by `frame()`. Register synchronously after that root exists.
Later registration replaces that axis's bindings. Fresh DOM needs fresh
registration; bindings on hidden, retained DOM remain available when shown.

Directional callbacks receive `(start, end, space)`. For a pointer gesture,
`start` and `end` are `[x, y]` arrays; for a key they are `undefined`. `space` is
the target DOM space element. Return values are ignored and promises are not
awaited. Bindings do not automatically dispose custom listeners or timers.

### Navigation and Focus

```js
pathless.navigate(direction = 1, space);
pathless.focus(direction = 1, target);
```

`navigate` adds an integer direction to the target space's index, wraps within
its collection, and rerenders. The default space is the focused main space,
even during another space's rendering. The panel uses `panels`; main spaces
use `frames`. Empty collections do nothing. Use `1` for next, `-1` for previous,
or `0` to rerender the current index.

`focus` cycles through visible main spaces, excluding the panel. The target
defaults to the focused space. `focus(0, space)` focuses a specific visible
main space; a hidden target or the panel is a no-op. Focus does not rerender.

Capture a frame's space when later controls must address it rather than focus:

```js
const { space } = pathless.frame();
const next = () => pathless.navigate(1, space);
```

### Layout

```js
pathless.layout(layout, space);
```

The required layout number is `0`, `1`, or `2`. The optional second argument is
a DOM space, not a variant number; it selects the main space in layout zero.
Main spaces below are named by their element IDs.

| Layout | Variant | Arrangement                                                     |
| ------ | ------- | --------------------------------------------------------------- |
| `0`    | `0`     | Selected space, otherwise focused space, fills the main region. |
| `1`    | `0`     | `zero` above `one`.                                             |
| `1`    | `1`     | `zero` left of `one`.                                           |
| `2`    | `0`     | `zero` above; `two` left of `one` below.                        |
| `2`    | `1`     | `zero` left; `one` above `two` on the right.                    |
| `2`    | `2`     | `two` above `one` on the left; `zero` right.                    |
| `2`    | `3`     | `one` left of `two` above; `zero` below.                        |

Entering a different split layout selects variant zero. Repeating the current
split layout cycles its variants modulo `2 ** layout`. There is no direct
variant parameter and no zero-argument layout command.

`layout(0)` from a split remembers that layout and variant and shows the
focused space. Calling `layout(0)` again without a space restores the remembered
split; without a remembered split it does nothing. Supplying an explicit main
space selects it without triggering restoration. If a change hides the focused
space, focus moves to the first visible main space. Layout changes preserve
populated DOM and initialize newly visible empty spaces.

The root's `data-layout` and `data-variant` contain string-valued layout state.
Use methods to keep visibility and geometry synchronized.

### Panel

```js
pathless.panel();
pathless.panel(true);
pathless.panel(false);
```

Omitting the argument toggles visibility; a boolean explicitly shows or hides
the fourth space. It occupies a bottom row sized to `15cqh` of the root.
Showing it initializes its first panel if needed; hiding/reopening it preserves
populated DOM and state. No registered panels still permits an empty panel
region. To navigate panel content, pass `pathless.spaces[3]` to `navigate`.

### Default Controls

Keyboard matching is case-insensitive. Handled keys prevent their default
action. Composition events, Ctrl/Meta/Alt combinations, default-prevented
events, and targets inside editable content, inputs, textareas, or selects
are ignored. Shift does not reverse Tab. Arrow keys are not aliases.

| Key or gesture              | Action                                                                  |
| --------------------------- | ----------------------------------------------------------------------- |
| `1`, `2`, `3`               | Call `layout(0)`, `layout(1)`, `layout(2)`.                             |
| Tab                         | Focus the next visible main space.                                      |
| `q` / `e`                   | Navigate previous / next.                                               |
| `z`                         | Toggle the panel.                                                       |
| `a` / `d`, left / right tap | Invoke the frame's `x` bindings.                                        |
| `w` / `s`, up / down tap    | Invoke the frame's `y` bindings.                                        |
| Left / right swipe          | Navigate previous / next in the pressed space.                          |
| Bottom-left tap region      | Advance layout number modulo three.                                     |
| Bottom-center tap region    | Call `layout(0)` to enter single space or restore.                      |
| Bottom-right tap region     | Repeat the current layout command, cycling split variants or restoring. |

Keys target the focused main space. Gestures target the pressed visible space,
including the panel. A non-bottom press focuses a visible main space before
dispatch. A gesture is cancelled on pointer cancellation, lost capture,
prevented/blocked release, hidden space, or replacement of its frame root.

Coordinates use the whole root's normalized plane: left/right are `-1`/`1`,
top/bottom are `1`/`-1`. Each axis is scaled independently and clamped. Only
one primary, button-zero pointer is tracked. After excluding registered click
targets and editable controls, gesture classification is:

1. Vertical displacement at least `0.1` and greater than horizontal displacement
   is ignored. Otherwise horizontal displacement at least `0.1` is a swipe.
2. With smaller movement, matching start/end bottom regions invoke layout
   commands. Regions require `y <= -0.8` and `x` within `0.2` of `-1`, `0`, or `1`.
3. Other taps select direction from the endpoint's dominant offset relative
   to the pressed space's center; ties select horizontal.

### Runtime Helpers

These are accessible for runtime integration, not additional frame controls:

| API                          | Contract                                                                                                              |
| ---------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `pathless.exec(data, space)` | Replace a space's children with an HTML contextual fragment; return `pathless`. Inline scripts run on insertion.      |
| `pathless.exec(data)`        | Return a `DocumentFragment` without inserting it.                                                                     |
| `await pathless.init()`      | Fetch bootstrap, assign collections, initialize Input and Universe; resolve to `undefined`. Called once by the shell. |
| `Pathless.decode(bytes)`     | Synchronously decode a decompressed wire `Uint8Array` into nodes; no gzip handling.                                   |

`exec` is HTML-first, not element-first. It does not sanitize, await frame
scripts, or clean up replaced content's side effects. Calling `init()` again
adds controllers/observers; it is not an idempotent reset.

## Transport

### Startup and Bootstrap

Go renders the three embedded templates into one HTML response on `:1000`.
There are no separate runtime-script requests. On initialization the browser
fetches the circuit root on `:1001`, whose payload is gzip-compressed JSON:

```text
[frames, panels]
```

Each collection contains HTML strings. Named resource routes use the binary
format below instead of the bootstrap JSON.
Circuit HTTP responses and saved payloads are gzip-compressed; the shell is
plain HTML. Browser HTTP handling
decompresses responses before `response.bytes()` and `Pathless.decode` run.
Compression is unconditional, not negotiated per request.

### Wire Format

A flat resource collection is encoded as:

```text
unsigned-varint resource count
for each resource:
        unsigned-varint name byte length + UTF-8 name
        unsigned-varint type byte length + UTF-8 MIME type
        unsigned-varint data byte length + raw data
```

Encoding and decoding operate on raw bytes; gzip is a separate transport concern.
Lengths count bytes. There are no child counts or nested records. There is no version or
magic prefix. Use `fx.NewFx().Encode(entries)` to encode a collection and
`fx.NewFx().Decode(data)` to decode wire bytes into one ordered resource collection.
`Encode` accepts multiple collections and concatenates their individual
encodings, each beginning with its own count. With no arguments it returns empty
bytes; an empty collection argument emits a zero count. `Decode` returns one `N`,
reading consecutive counted sequences until all bytes are consumed and
aggregating their resources in order. Sequence boundaries are not retained in
the result; empty bytes and empty sequences contribute no resources. Malformed input,
including an incomplete later collection, returns an error without partial results.
`Load` fetches a URL and returns the decoded `N` directly; `Input` sources raw content.
Decoding returns an error for malformed input and copies fields independently of
the input buffer. Collections passed to `Encode` must contain non-nil resource pointers.
This replaces the previous recursive format: saved payloads must be regenerated
and external decoders updated before exchanging the new records.

### Remote Import Caveat

`Load` decodes the fetched response-body bytes. Go's default HTTP
transport automatically decompresses responses marked `Content-Encoding: gzip`.
Serve raw wire bytes, or mark gzip-compressed responses with that header;
unmarked gzip bytes fail decoding. `Input` never decodes
saved payloads automatically.

## Boundaries

- Load trusted content only. Frame HTML/scripts share the shell document and
  origin; block-scoped scripts are not sandboxing or sanitization.
- There is no `Text`, `Slides`, `Route`, `get`, `status`, `togglePanel`,
  `pathless.universe`, or `pathless.input` API. Pathless is not an EventTarget
  and does not publish change notifications.
- Supported space arguments are actual elements from `pathless.spaces`;
  directions are integers and layouts are `0`, `1`, `2`. Arguments are not
  generally validated. There is no direct layout-variant setter.
- Servers, custom frame side effects, cached object URLs, state maps, and
  observers have no coordinated disposal API. Construction/startup are one-shot.
- Go reads/decompression have no explicit size limits. Its decoder checks
        framing and trailing bytes; the browser decoder assumes valid input
  and is not suitable for untrusted payloads.
- Use a modern browser supporting `Response.bytes()`, `ResizeObserver`,
  contextual HTML fragments, CSS nesting, and container query units.

## Sources

| Source                                   | Responsibility                                        |
| ---------------------------------------- | ----------------------------------------------------- |
| [pathless.go](pathless.go)               | Public Go constructor and embedded `*one.One`.        |
| [fx/fx.go](fx/fx.go)                     | Resource API and HTTP client construction.            |
| [fx/input.go](fx/input.go)               | `One`, `N`, flat imports, naming, and sequencing.     |
| [fx/output.go](fx/output.go)             | Public flat wire encoder/decoder.                     |
| [zero/zero.go](zero/zero.go)             | Embedded assets and HTML rendering.                    |
| [one/one.go](one/one.go)                 | Resource API, frame collections, muxes, and startup.   |
| [one/frame.go](one/frame.go)             | HTML registration and frame/panel encoding.            |
| [one/zero.go](one/zero.go)               | Shell handler and redirects.                          |
| [one/fx.go](one/fx.go)                   | Circuit HTTP registration, headers, and CORS.         |
| [zero/zero.html](zero/zero.html)         | Shell CSS, resource loading, and HTML execution.      |
| [zero/universe.html](zero/universe.html) | Space state, layout, focus, and navigation.           |
| [zero/input.html](zero/input.html)       | Click, keyboard, and pointer dispatch.                |
