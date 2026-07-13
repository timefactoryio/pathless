# one

`one` is the HTTP server. It wires `zero`'s compiled shell and `fx`'s registered frames/routes into two listeners, gzip-compressing everything once at startup and serving it from memory thereafter. `templates.go` builds the handful of ready-made frame templates (`Home`, `Text`, `Slides`, `Keyboard`) shipped with the package.

| File               | Role                                                                                           |
| ------------------ | ---------------------------------------------------------------------------------------------- |
| `one.go`           | `One` (embeds `*zero.Zero`, `*fx.Fx`) — two `http.ServeMux`s, gzip, `Serve()`                  |
| `templates.go`     | `Home`/`Logo`/`Text`/`Slides`/`Keyboard` — template builders that call `Frame`/`Panel`/`build` |
| `templates/*.html` | the four embedded template sources these builders parse                                        |

---

## `one.go` — server

```go
type One struct {
    *zero.Zero
    *fx.Fx
    pathless *http.ServeMux // ":1000" — the compiled shell
    circuit  *http.ServeMux // ":1001" — wire routes
}

func NewOne(z *zero.Zero, f *fx.Fx) *One
```

`One` embeds both `*zero.Zero` and `*fx.Fx`, so their fields/methods (`Pathless`, `Origin`, `ToValue`, `Build`, `Frame`, `Panel`, `Routes`, `Hello`, ...) are all directly callable on `One` — `templates.go`'s builders rely on this promotion.

`NewOne` gzip-compresses the shell (`o.Pathless = zip(o.Pathless)`, promoted from `zero.Zero`), registers `Keyboard()` as the first panel (guaranteeing the panel pool is never empty), and registers the shell's catch-all handler. There is no separately-tracked universe field — the universe payload is `Hello[0]`, assembled by `Fx.Build()` inside `Serve()`.

| Member                 | Behavior                                                                                                                                                    |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `handlePathless(w, r)` | serves `Pathless` at `/` (gzip). Any other path or a non-empty query redirects to `/` — the shell is a single page, routing lives entirely on the wire side |
| `wire(path, v)`        | gzip-compresses `Encode(v)` once and registers a handler on `circuit` that serves those fixed bytes as `application/octet-stream`                           |
| `cors(next)`           | sets `Access-Control-Allow-Origin: o.Origin` (from `zero.Zero`) on every response, short-circuits `OPTIONS` with `204` — wraps `circuit` only               |
| `Serve()`              | assembles and registers every route, then blocks (see below)                                                                                                |

### `Serve()`

```go
func (o *One) Serve() {
    o.Fx.Build()
    o.wire("/", o.Hello)

    for key, v := range o.Fx.Routes {
        o.wire("/"+key, []*fx.Value{v})
    }
    ...
}
```

`Build()` assembles `Hello` as `[header, universe, ...Frames, ...Panels]` — a 1-byte frame-count header lets the client slice `Frames` from `Panels` without either needing its own wire entry. `Hello` as a whole is `Encode`d once, at `/` on `:1001`. `NewOne` always registers `Keyboard()` as the first panel, so the panel pool is never empty.

Every other `fx.Routes` entry (populated by a direct `Routes[key] = v` assignment after `ToValue`, e.g. in `Slides` or a non-SVG `Logo`) is wired individually at `/`+key.

`Serve()` then starts both listeners — `:1001` (wire, CORS-wrapped) in a goroutine, `:1000` (shell) blocking on the main goroutine. Both mux's routes are fixed by this point; nothing is registered after `Serve()` is called.

### `zip(data) []byte`

Gzips `data` at `gzip.BestCompression`. Used for the shell (once, in `NewOne`) and for every wire route (once, in `wire`) — never per request. Every response this package writes sets `Content-Encoding: gzip` to match.

---

## `templates.go` — built-in frame templates

Five embedded `templates/*.html` files, parsed and executed on demand by these builders. Each one appends directly to `Frames`/`Panels` (promoted from `fx.Fx`) via `build`, so registration follows the same rules as any hand-authored frame (see [fx](../fx/README.md#fxgo--framepanel-building)).

| Builder               | Behavior                                                                                                                                                                                                                                                                      |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Home(logo, heading)` | renders `templates/home.html` with `{{.LOGO}}` (via `Logo`) and `{{.HEADING}}`, registers it as a space frame                                                                                                                                                                 |
| `Logo(path)`          | `.svg` → inlined verbatim (`<svg>` markup, via `ToValue`). Anything else → registered as a route (`ToValue` + `Routes[key] = v`) and referenced by `data-src="{Origin}/{basename}"`; a remote URL is used as `src` directly, unfetched                                        |
| `Text(path)`          | fetches `path` (`ToValue`), renders it as Markdown (`markdown.New("")`), injects the HTML into `templates/text.html`, registers it as a space frame                                                                                                                           |
| `Slides(dir)`         | registers `dir` as a route (`ToValue` + `Routes[key] = v`; its images, in `sort.txt` order if present), renders `templates/slides.html` with `{{.PREFIX}}` set to the route's key, registers it as a space frame — the frame's own script then fetches that route client-side |
| `Keyboard()`          | builds `templates/keyboard.html` via `build` and appends it directly to `Panels`. Nothing currently calls it automatically — a program must call `o.Keyboard()` itself to get the default panel                                                                               |

`Logo`'s two paths mirror the two things a frame can point at: embed small/local assets directly, or register anything larger as its own route and let the client fetch it lazily.

Nothing currently registers the keyboard panel automatically — a program must call `o.Keyboard()` itself (e.g. right after `NewPathless()`) to get a working panel strip.

---

