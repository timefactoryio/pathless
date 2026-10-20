# Mechanics

A maintenance map for the runtime. Start with [README.md](README.md) for usage.

## Components

| Source                           | Responsibility                                                     |
| -------------------------------- | ------------------------------------------------------------------ |
| [pathless.go](pathless.go)       | Compose the shell and content runtime behind `Pathless`.           |
| [zero/zero.go](zero/zero.go)     | Embed the browser runtime, configure origins, serve the shell.     |
| [fx/fx.go](fx/fx.go)             | Register frames and routes, prepare HTML, serve the circuit.       |
| [fx/input.go](fx/input.go)       | Read resources, build content trees, normalize names and ordering. |
| [fx/output.go](fx/output.go)     | Define `One` and encode/decode content trees.                      |
| [zero/zero.html](zero/zero.html) | Shell CSS, resource loading, browser decoding, HTML execution.     |
| [zero/one.html](zero/one.html)   | Spaces, rendering, layouts, focus, state, pointer input.           |

Templates and applications compose these APIs from separate modules.

## Startup

1. `NewPathless` constructs `Zero`, renders the shell, and starts its server
  on `:1000`.
2. The application registers frames, panels, and public resources. `Start`
  encodes them and serves the circuit on `:1001`.
3. On `DOMContentLoaded`, the shell creates `window.pathless`. Its `init()`
  fetches the bootstrap, stores frames/panels, and executes the universe HTML.
4. `new Universe()` creates the spaces and Pointer, measures and observes the
  root, exposes bound controls and the `frame` getter on `pathless`, then calls
  `sync()` to render visible content. Pointer receives the instance directly.

Both HTML files are embedded in Go; changes require rebuilding the application.
The bootstrap is positional: universe HTML first, then frame and panel
containers whose `Ones` fields hold the registered content.

## Rendering

Universe owns three main spaces and a panel. Each record tracks its element,
frame index, state Maps, bindings, and measured center. Layout and focus live
in the root's `data-layout`, `data-variant`, and `data-focused`; panel visibility
uses its `panel` class. CSS maps this state to a viewport-filling grid.

`sync()` fills visible spaces with no first element child; explicit records
force a rerender. Navigation wraps the target space's frame index and rerenders
it. Layout, focus, and panel changes measure geometry and sync missing content.
Hidden spaces keep their DOM until rerendered.

During execution, `pathless.frame` resolves against the rendering space;
afterward it follows the focused main space. Rendering replaces DOM and
bindings while preserving the state Map for each space/frame-index pair.

The frame builder merges exact, attribute-free `<style>` blocks and moves
exact `<script>` blocks into scoped blocks at the end. It uses regular
expressions, not an HTML parser. The browser inserts a contextual fragment
with `replaceChildren`, executing inline scripts in the shared document.

## Pointer Input

Pointer captures one primary, button-zero gesture at a time. Pressing a main
space focuses it; pressing the panel leaves main focus unchanged. Cancellation,
lost capture, a prevented release, or a hidden target space cancels the gesture.

Coordinates use the full root's normalized plane, clamped to `[-1, 1]`, with
positive Y upward. Geometry assumes the root begins at viewport `(0, 0)`;
each axis scales independently. Dispatch uses this precedence:

1. Movement of at least `0.1` on either axis is a swipe. Horizontal dominance
  navigates; vertical dominance does nothing.
2. A tap with both endpoints inside the central `[-0.05, 0.05]` square toggles
  the panel.
3. Other taps select an axis and direction relative to the pressed space's
  center, choosing horizontal on ties, and invoke its registered binding.

## Wire Format

Circuit responses and saved payloads are gzip-compressed node lists. After
decompression:

```text
unsigned-varint node count
for each node:
   unsigned-varint name byte length + UTF-8 name
   unsigned-varint type byte length + UTF-8 MIME type
   unsigned-varint data byte length + raw data
   child node list (same format)
```

Lengths count bytes. Leaves end with a zero child count. There is no version
or magic prefix. Go uses `One` values; the browser produces
`{ name, type, data, ones }`, decoding HTML data to strings and retaining other
data as byte arrays. Browser HTTP handling decompresses circuit responses
before the JavaScript decoder runs.

## Transport Boundaries

- The shell redirects non-root paths and queries to `/`. The circuit's root
  handler is a catch-all, so unknown resource paths can return bootstrap data.
- Circuit responses use `application/octet-stream` and `Content-Encoding: gzip`.
  CORS advertises `GET, OPTIONS` and answers preflight requests; it does not
  enforce authentication or restrict other HTTP methods.
- Go URL input tries gzip decoding before falling back to a raw resource.
  Go's HTTP transport may already decompress a circuit response, making direct
  circuit imports fall back to raw bytes. A saved gzip artifact served without
  `Content-Encoding` avoids this mismatch.
- The Go decoder checks framing, nesting, and trailing bytes. The browser
  decoder assumes valid input. Neither is a resource-budget boundary for
  untrusted payloads.

One recursive type covers a file, a directory, a frame, and a payload, so no conversion layer exists between what `Input` returns and what the circuit sends. Frames receive complete node metadata when identity and MIME type matter — associating slide images with descriptions, for example — without a second representation.

The result is a small origin from which applications can be assembled without inheriting a device vendor’s application model.
