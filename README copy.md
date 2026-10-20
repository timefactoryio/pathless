# pathless

Pathless serves HTML frames through a browser shell and a content circuit.
The shell stays at `/`; navigation changes frames, not URLs.

Frames occupy up to three independent spaces, with a separate region for
panels. The circuit distributes frames and typed resource trees. Templates
and application-specific content live outside this module.

## Quick Start

From a Go module, add the dependency with `go get github.com/timefactoryio/pathless`:

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

Run `go run .` and open `http://localhost:1000`. The shell starts during
construction; `Start` serves the circuit on `:1001` and blocks. Register all
content before calling it. Content is served as a startup snapshot, not watched
for changes.

## Go API

Construct with `pathless.NewPathless()`. Its embedded `fx.Fx` provides:

| Method | Purpose |
| --- | --- |
| `Input(path, public ...bool) ([]*fx.One, error)` | Read a file, recursive directory, or HTTP(S) URL. Pass `true` to publish it. |
| `Frame(source, panel ...bool) error` | Register HTML content as a frame, or as a panel with `true`. |
| `Save(key, binary ...bool) ([]byte, error)` | Encode a published route as gzip bytes; `true` also writes a file named `key`. |
| `Start()` | Serve the circuit. Call once, after registration. |

The optional booleans default to `false`. Frames and panels retain registration
order. A frame source is literal HTML starting with `<`, or a file/URL/input
that resolves to exactly one node.

```go
if err := p.Frame("./home.html"); err != nil {
	log.Fatal(err)
}
if err := p.Frame("./controls.html", true); err != nil {
	log.Fatal(err)
}
```

### Publish Resources

```go
if _, err := p.Input("./photos", true); err != nil {
	log.Fatal(err)
}
```

The route is now available to frames as `pathless.source('photos')`.
Directory routes use the directory basename; file and URL routes use the
basename without its final extension. Publishing the same key replaces it.

Content uses `fx.One` nodes with `Name`, MIME `Type`, byte `Data`, and child
`Ones`. Directories return their children. Within each directory, extensions
are dropped only when names remain unique; dot-prefixed names are preserved.
MIME types come from extensions, with content sniffing as fallback.

A `sequence.txt` file can list normalized sibling names, one per line, to set
their order. It is consumed as metadata; unlisted entries follow in filename
order. Sequencing applies independently to each directory.

`Save` exports one published route, not the whole application. Remote `Input`
can import saved gzip payloads; local file input reads raw bytes. See
[mechanics.md](mechanics.md#transport-boundaries) for HTTP transport limits.

### Deploy

```go
p := pathless.NewPathless("example.com", "circuit.example.com")
```

Pass either no arguments or two bare domains. Hosted mode advertises HTTPS
origins and restricts circuit CORS to the shell origin. The listeners remain
HTTP on `:1000` and `:1001`; provide TLS and routing through a reverse proxy.
Local mode allows any origin and uses the browser's hostname for the circuit.

## Browser API

Frame scripts use the global `pathless` object. The runtime handles rendering,
focus, and pointer input; frames supply content and directional callbacks.

| API | Purpose |
| --- | --- |
| `frame` | Current frame's `{ el, state }`, where `state` is a persistent Map. |
| `source(path, blobs = false)` | Load a public route as content nodes or blob URLs. |
| `x(left, right)` | Bind left/right taps in the current space. |
| `y(up, down)` | Bind up/down taps in the current space. |
| `navigate(direction)` | Move through frames; `-1` is previous, `1` is next. Wraps. |
| `focus(direction = 1)` | Cycle focus through visible main spaces. |
| `layout(layout, variant)` | Select a layout or cycle its variants. |
| `togglePanel()` | Show or hide the panel; no-op when none are registered. |

### Write a Frame

Give each frame one root element. Capture `pathless.frame` and register
bindings during synchronous initialization:

```html
<main><output></output></main>
<script>
        const { el, state } = pathless.frame;
        const output = el.querySelector('output');
        const show = () => {
                output.textContent = state.get('count') ?? 0;
        };
        const change = (amount) => {
                state.set('count', (state.get('count') ?? 0) + amount);
                show();
        };
        pathless.x(() => change(-1), () => change(1));
        show();
</script>
```

State persists per space and frame index. Navigation replaces the frame's DOM
and bindings, but not its Map. Capture `{ el, state }` before any `await` or
later callback: a fresh `pathless.frame` lookup then follows the focused space.

### Load Resources

```js
const { el } = pathless.frame;
const [imageURL] = await pathless.source('photos', true);
el.querySelector('img').src = imageURL;
```

Use the route key registered by `Input(..., true)`, without a leading slash.
The default result is an array of `{ name, type, data, ones }` nodes: HTML
data is a string; other data is a `Uint8Array`. `ones` contains child nodes.
Passing `true` returns blob URLs for top-level resources, not nested children.
Requests and blob URLs are cached for the page lifetime; failed requests can
be retried.

### Layouts and Input

| Layout | Arrangement | Variants |
| --- | --- | --- |
| `0` | Focused space only | One |
| `1` | Two spaces, side by side or stacked | Two |
| `2` | Three spaces, with one spanning a row or column | Four |

`layout(1)` and `layout(2)` enter a split layout; repeating the call cycles its
variants. Pass an explicit zero-based variant to select one. `layout(0)`
toggles between the focused space and the last split layout. `layout()` cycles
the current split's variants or restores the saved split.

Each space navigates independently. Pressing a main space focuses it. Horizontal
swipes navigate frames; vertical swipes are ignored. A tap near the full
viewport's center toggles the panel. Other taps invoke `x` or `y` callbacks
according to their direction from the pressed space's center.

Callbacks receive `(start, end, space)`. Coordinates share the full viewport's
normalized plane: left/right are `-1`/`1`, top/bottom are `1`/`-1`. Bindings are
replaced on every render, and callback promises are not awaited.

## Boundaries

Load trusted content only. Frames execute in the shell's document and origin,
without sanitization or sandboxing. Rendering does not clean up custom timers,
listeners, or observers. CORS is not authentication.

For component ownership, startup, and wire format, see [mechanics.md](mechanics.md).

## Boundaries

- There is no `Text`, `Slides`, `Route`, or `pathless.input` API in this version.
    Compose templates externally and register their HTML with `Frame`.
- Frame scoping is not security isolation. HTML/scripts must be trusted.
- Servers, cached object URLs, state maps, and observers have no coordinated
    disposal API. Construction/startup are one-shot operations.
- Go reads/decompression have no explicit size limits; the browser decoder
    assumes valid payloads and performs fewer checks than the Go decoder.
