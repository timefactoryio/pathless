# one

`one` is the HTTP server. It wires `zero`'s compiled shell and `fx`'s registered frames/routes into two listeners, gzip-compressing everything once at startup and serving it from memory thereafter. `templates.go` builds the handful of ready-made frame templates (`Home`, `Text`, `Slides`, `Keyboard`) shipped with the package.

| File               | Role                                                                                        |
| ------------------ | ------------------------------------------------------------------------------------------- |
| `one.go`           | `One` (embeds `*zero.Zero`, `*fx.Fx`) — two `http.ServeMux`s, gzip, `Serve()`               |
| `templates.go`     | `Home`/`Logo`/`Text`/`Slides`/`Keyboard` — template builders that call `Build`/`BuildPanel` |
| `templates/*.html` | the four embedded template sources these builders parse                                     |

---

## `one.go` — server

```go
type One struct {
    *zero.Zero
    *fx.Fx
    pathless *http.ServeMux // ":1000" — the compiled shell
    circuit  *http.ServeMux // ":1001" — wire routes
    Pathless []byte         // gzip-compressed shell HTML
    Universe []byte         // universe.html, uncompressed (bundled into the root wire payload)
}

func NewOne(z *zero.Zero, f *fx.Fx) *One
```

`One` embeds both `*zero.Zero` and `*fx.Fx`, so their fields/methods (`Origin`, `Load`, `Build`, `Panels`, `Routes`, ...) are all directly callable on `One` — `templates.go`'s builders rely on this promotion.

`NewOne` calls `z.Compile()` once, gzip-compresses the shell into `Pathless`, keeps `Universe` raw (it's bundled into the wire payload, not served directly), registers `Keyboard()` as the first panel (guaranteeing the panel pool is never empty), and registers the shell's catch-all handler.

| Member                 | Behavior                                                                                                                                                    |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `handlePathless(w, r)` | serves `Pathless` at `/` (gzip). Any other path or a non-empty query redirects to `/` — the shell is a single page, routing lives entirely on the wire side |
| `wire(path, v)`        | gzip-compresses `Encode(v)` once and registers a handler on `circuit` that serves those fixed bytes as `application/octet-stream`                           |
| `cors(next)`           | sets `Access-Control-Allow-Origin: o.Origin` (from `zero.Zero`) on every response, short-circuits `OPTIONS` with `204` — wraps `circuit` only               |
| `Serve()`              | assembles and registers every route, then blocks (see below)                                                                                                |

### `Serve()`

```go
o.wire("/", &fx.Value{Values: []*fx.Value{
    {Type: "text/html", Data: o.Universe},
    o.Frames(),
    o.Panels(),
}})
```

The root wire route (`/` on `:1001`) is always exactly three items, in fixed order: the universe payload, the registered frame pool, and the panel pool. `o.Panels()` is never empty — `NewOne` always registers `Keyboard()` as the first panel — so, unlike `Frames`/`Panels` calls elsewhere, no nil-check is needed here. This order is exactly what the client destructures: `const [universe, frames, panels] = await this.source()` (see [zero](../zero/README.md#pathlesshtml--shell--client-bootstrap)).

Every other `fx.Routes` entry (populated by `Load`, e.g. a `Slides` directory or a non-SVG `Logo`) is wired at `/`+key.

`Serve()` then starts both listeners — `:1001` (wire, CORS-wrapped) in a goroutine, `:1000` (shell) blocking on the main goroutine. Both mux's routes are fixed by this point; nothing is registered after `Serve()` is called.

### `zip(data) []byte`

Gzips `data` at `gzip.BestCompression`. Used for the shell (once, in `NewOne`) and for every wire route (once, in `wire`) — never per request. Every response this package writes sets `Content-Encoding: gzip` to match.

---

## `templates.go` — built-in frame templates

Five embedded `templates/*.html` files, parsed and executed on demand by these builders. Each one ends by calling `o.Build`/`o.BuildPanel` (promoted from `fx.Fx`), so registration follows the same rules as any hand-authored frame (see [fx](../fx/README.md#fxgo--framepanel-building)).

| Builder               | Behavior                                                                                                                                                                                                                                     |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Home(logo, heading)` | renders `templates/home.html` with `{{.LOGO}}` (via `Logo`) and `{{.HEADING}}`, registers it as a space frame                                                                                                                                |
| `Logo(path)`          | `.svg` → inlined verbatim (`<svg>` markup, via `ToBytes`). Anything else → `Load`ed as a route and referenced by `data-src="{Origin}/{basename}"`; a remote URL is used as `src` directly, unfetched                                         |
| `Text(path)`          | fetches `path` (`ToBytes`), renders it as Markdown (`markdown.New("")`), injects the HTML into `templates/text.html`, registers it as a space frame                                                                                          |
| `Slides(dir)`         | `Load`s `dir` as a route (its images, in `sort.txt` order if present), renders `templates/slides.html` with `{{.PREFIX}}` set to the route's key, registers it as a space frame — the frame's own script then fetches that route client-side |
| `Keyboard()`          | builds `templates/keyboard.html` via `BuildPanel` and returns it. `NewOne` already registers it as the first panel (`o.Panels(o.Keyboard())`) — call it again only to reference the same markup elsewhere, not to re-register it             |

`Logo`'s two paths mirror the two things a frame can point at: embed small/local assets directly, or register anything larger as its own route and let the client fetch it lazily.

Because the keyboard panel is guaranteed, a consumer only needs to call `Panels(...)` at all if it wants to add *more* panels after it — it's never required just to get a working panel strip.

---

