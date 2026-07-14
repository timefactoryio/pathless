# pathless 

pathless renders **frames** — single `.html` files built for data the developer already knows: its shape, its cardinality, its natural interactions. The data can be anything expressible as bytes — JSON, markdown, images, CSV, logs, binary formats, generated structures — there is no schema requirement. There's also no separate mobile path: one `Input` model resolves mouse, touch, and pen into the same events, and one sizing unit (`cqw`/`cqh`) fits a frame to whatever container it's given. A frame is authored once and rendered wherever it's observed.

The three sections below follow the Go packages a frame touches, in the order it touches them: **zero** (what the frame calls at runtime), **fx** (how the frame's file is authored and registered), **one** (how it's encoded and served).

---

## frame

A frame is one `.html` file with three parts: `<style>`, static markup (the shell), and a `<script>`. The shell — including its root container div — is authored as divs directly in the HTML. Because the data is known ahead of time, its structure (root, sections, lists, an `<img>` per slot, …) is written statically. The script does **not** build the DOM; it queries the shell, wires interaction, and updates the dynamic slots (`textContent`, `src`, `hidden`, dataset/state attributes).

```html
<style>
  /* scoped to .<framename>; use cqw / cqh units only */
</style>

<!-- the shell: root container + static structure mirroring the known data -->
<div class="frame"></div>

<script>
{
  class Frame {
    constructor(p) {
      // 1. pin state, read persisted values
      // 2. query your own subtree under p.universe.frame
      // 3. register p.input.bind
      // 4. load async data, then populate the shell
      this.el = p.universe.frame;                       // your root container (the .frame div)
      this.img = this.el.querySelector('.viewer img');  // a slot, reached by class
      this.desc = this.el.querySelector('.desc');
    }
  }
  new Frame(pathless);
}
</script>
```

### Authoring rules

A frame is scoped by containment — **universe → space → frame**. A space holds one frame at a time through a single root container; everything **below** that container is the frame's alone, everything **at or above** it (the space, the universe) is the client shell's. Every authoring rule falls out of that one boundary:

- **Query your own subtree via `p.universe.frame`, never `document`.** That's your root container; `querySelector` down from it. `p.universe.space.el` is the space *hosting* you — the shell's layer — so you don't reach up into it to find your own markup (see the [runtime table](#zero--what-a-frame-calls-at-runtime)).
- **Classes, never `id`.** `id`s name client-shell structure (`#universe`, `#zero`, `#panel`); a frame lives entirely in class-space within its subtree. It's separation of concerns first, and it keeps the same frame safe when it renders into several spaces at once.
- **Author the shell as static markup; the script only fills slots.** The root container and its structure are HTML, because the data's shape is known ahead of time. The script queries that shell and writes into slots — `textContent`, `src`, `hidden`, dataset/state attributes — and never builds structure from data. `innerHTML` with data stays out on both counts: an injection vector, and a break from the static-shell model.
- **Size against your container, never the viewport.** `cqw`/`cqh` only, never `vw`/`vh`, and no `width`/`height` on the root — the space already sizes it. This one is correctness: split and quad layouts put several frames on screen at once, so viewport units would size to the wrong box (see [Sizing](#sizing)).

Two further rules are runtime concerns, covered where they apply: input goes through [`p.input.bind`](#pinputbindbinds) (never `addEventListener` for navigation/taps/keys), and deferred state access must use a pinned `read`/`write` (see [State semantics](#state-semantics)).

**When you must wrap the script in `{ }`.** A frame's script re-executes on every render, so any top-level lexical declaration (`class`, `const`, `let`, `function`) would redeclare on the second render and throw — block scope is what prevents it. For a frame registered through `Frame`/`Panel` or a built-in template you never have to think about this: `build()` already wraps every consolidated script in an outer block, so authoring the braces is cosmetic. You only wrap by hand when a script **bypasses that build path** — the universe payload and any HTML served raw (not run through `build()`) must carry their own `{ }`, because nothing adds it for them.

---

## zero — what a frame calls at runtime

`zero` is the browser runtime a frame executes inside. `window.pathless` (passed as `p`) is the only global; `p.universe` (including `p.universe.panel`), `p.input`, and `p.keyboard` (the default keyboard panel, always registered) are the modules attached to it.

| Member                       | Signature                                | Behavior                                                                                  |
| ---------------------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------- |
| `p.source(key)`              | `(string) => Promise<[{type,data,url}]>` | fetch + decode a route; cached per key                                                    |
| `p.universe.frame`           | `HTMLElement`                            | the focused frame's root container — the base for querying your own markup                |
| `p.universe.space.el`        | `HTMLElement`                            | the space element hosting the frame (shell layer); sizing context, not your query root    |
| `p.universe.read(i?)`        | `(number?) => Map`                       | per-(frame, space) state map; survives re-render; defaults to the currently focused space |
| `p.universe.write(k, v, i?)` | `(string, any, number?) => void`         | persist `k → v` into the state map; same default caveat as `read`                         |
| `p.universe.pin(i?)`         | `(number?) => {i, read, write}`          | captures the focused index once and returns read/write bound to it                        |
| `p.universe.sync(...i)`      | `(...number) => void`                    | re-render the given spaces, or all visible spaces if none given                           |
| `p.input.bind(binds)`        | `(object) => void`                       | register gesture and key handlers for the focused space                                   |
| `p.universe.panel.toggle()`  | `() => void`                             | show/hide the panel strip (also bound to reserved key `z`)                                |

### `p.source(key)` — data access

```js
p.source(key) // → Promise<Array<{ type, data, url }>>
```

- `key` is the route key **without a leading slash**: `p.source('catalog.json')`, `p.source('pics')`. The empty key is reserved for the shell.
- Each entry:
  - `type` — MIME type string (`image/avif`, `application/json`, `text/plain`, …)
  - `data` — `Uint8Array` of the raw bytes, already in memory
  - `url` — lazy getter; creates a `blob:` object URL from `data` on first access
- There are **no names**. Entries are identified by position (see `sort.txt` under [fx](#fx--authoring-and-registering-a-frame)).
- The promise is cached per key — repeated calls cause one fetch.
- A single-file route yields a **one-element array**.

Decode `data` however the format requires — the bytes are already local, so no second fetch is ever needed:

```js
// structured text (JSON here; same idea for CSV, NDJSON, etc.)
const [entry] = await p.source('catalog.json');
const data = JSON.parse(new TextDecoder().decode(entry.data));

// media: hand the lazy blob url to the element
const pics = await p.source('pics');
img.src = pics[i].url; // index i per sort.txt order

// data + companion bundle, referenced by index
const [[file], pics] = await Promise.all([
    p.source('catalog.json'),
    p.source('pics'),
]);
const records = JSON.parse(new TextDecoder().decode(file.data));
// record { "image": 2 } → pics[2].url
```

### `p.input.bind(binds)`

A frame has exactly one way to register input, for both touch and keyboard: `p.input.bind({...})`. Every trigger — tap or key — is a named property on one plain object, passed in a single call; calling `bind` again replaces the whole set for the focused space.

This is one delegated table, not many listeners. The shell keeps a single set of DOM listeners and routes every named event into your bindings — so a frame never attaches `addEventListener` for navigation, taps, or keys. That's less a prohibition than the point of the API: one handler that already knows *what* to route and *when* beats stacking per-element listeners a frame would have to add, track, and tear down across re-renders. (For genuinely element-local events — a `<video>`'s `ended`, an `IntersectionObserver` — listen on your own elements as usual; those aren't input.)

Underneath, a tap and a keypress are the same thing: a named event the shell resolves and routes to one binding table. That's why there's no separate touch/mouse handling to author — a gesture is a literal alias of a key, so a feature bound once works from a thumb or a keyboard.

The value shape tells `bind` what kind of trigger it is:

```js
p.input.bind({
  tapLeft:  () => this.prev(),               // gesture -> plain function, fires once
  tapRight: () => this.next(),
  a:        { down: () => this.prev() },      // key -> { down, up }, down repeats while held
  d:        { down: () => this.next() },
  w:        { down: () => this.scroll(-1), up: () => this.stop() },
});
```

Gesture names map to a function because a gesture is instantaneous. Key names (any `e.key.toLowerCase()`) map to `{ down, up }` because a key has duration a frame may need to act on — `down` fires (and repeats) while held, `up` on release.

**The only gestures available to frames are `tapLeft` and `tapRight`** — a press resolves to a tap on the left or right half of the space, or to a horizontal swipe. Horizontal swipes are reserved (below); vertical movement is deliberately unclassified so native scrolling inside a frame is never hijacked.

Reserved names are resolved by the shell before a frame's bindings are ever consulted, so they can't be overridden:

| Name               | Action                                                    |
| ------------------ | --------------------------------------------------------- |
| `q` / `swipeLeft`  | previous frame in the focused space                       |
| `e` / `swipeRight` | next frame in the focused space                           |
| `1` / `2` / `3`    | space-layout controls — reserved, not available to frames |
| `tab`              | move focus to the next visible space                      |
| `z`                | toggle the panel                                          |

Pressing anywhere inside a space also moves focus to it before the gesture is classified.

### State semantics

State is a `Map` keyed to the (frame, space) pair. The same frame in two spaces has two independent maps. Persist only serializable view state (indices, scroll offsets, toggles).

`read()`/`write()` default their space index to `p.universe.state.focused` — but that value is only guaranteed correct **synchronously during construction** (the shell sets it right before executing the frame's script). Any code that runs later — a promise callback, an event listener, a `bind` handler firing on user input — must not rely on the default, because by then a *different* space may be focused. Capture the index once with `p.universe.pin()` and use the `read`/`write` it returns everywhere else:

```js
constructor(p) {
  const { read, write } = p.universe.pin();  // captures once, synchronously
  this.write = write;
  this.index = read().get('index') ?? 0;
  // ...
}
// later, from a promise/event/bind callback:
this.write('index', this.index);
```

The tradeoff is small: a frame that only touches state synchronously in its constructor can use the `read`/`write` defaults, but anything deferred — a `p.source().then(...)`, a `bind` handler, any later callback — must go through a pinned `read`/`write`, because by then `focused` may point at a different space. Since almost every frame defers (it binds input or loads data), pinning once at construction is the simple, always-correct default.

### Sizing

The container a frame renders into is always a flex child with `container-type: size` already set — that's what enables container query units. `1cqw` = 1% of that container's width, `1cqh` = 1% of its height, correct no matter how many other spaces happen to be on screen at the same time, because sizing is relative to the frame's own container, never the viewport. Use `clamp(min, Ncqw, max)` for fluid type. For a scrollable frame, set `overflow-y: auto` and `height: 100%` on the root.

---

## fx — authoring and registering a frame

```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless()

    // templates and custom frames compose freely
    p.Home("./logo.svg", "Title")

    // custom data frame: build values, register them as routes, register the frame
    cat, _ := p.ToValue("./data/catalog.json")
    p.Route("catalog.json", cat) // route "catalog.json" (base name of path)
    pics, _ := p.ToValue("./pics")
    p.Route("pics", pics)        // route "pics" (directory → nested bundle)
    p.Frame("./catalog.html")    // frame; file authors <div class="catalog">

    p.Serve()
}
```

Rules an agent must follow when emitting `main.go`:

- `p.Frame(path)` registers a space frame. The file authors its own root container div, classed by convention after the filename stem (`catalog.html` → `<div class="catalog">…</div>`). Nothing is wrapped for you. The content is trusted as-is — **local, developer-controlled files only.**
- `p.ToValue(path)` **builds** a `*Value` and nothing more — it never registers. It accepts a file, a directory (one `Value` whose `Children` hold every file, in `sort.txt` order), or an `http(s)://` URL (fetched and treated like a file — it does not decode a `Save`-produced blob back into a bundle). Any file type works — the wire carries typed bytes, and the frame decides how to decode them. MIME type is inferred from extension, with content sniffing as fallback.
- `p.Route(key, v)` **registers** a built `*Value` as a fetchable route and returns `key`. This is the only thing that makes content reachable via `p.source(key)` — splitting it from `ToValue` means building a `Value` never implies serving it. By convention `key` is `filepath.Base(path)`: `./data/catalog.json` → `catalog.json`, `./pics` → `pics`.
- `p.Save(key)` gob-encodes an already-registered route's `Value` to `s3/<key>`, ready to sync to object storage (e.g. `rclone sync s3 remote:bucket`). There is currently no counterpart that loads a saved blob back through `ToValue`.
- Every route a frame reads via `p.source(...)` **must** be registered via `p.Route(...)` (directly, or by a template that registers internally).

### Custom templates: register while building

A built-in like `Home`/`Text`/`Slides` is just an `Fx` method that builds a frame and, when that frame needs data, registers it in the same call. Adding one follows a fixed shape — **build the companion value, `Route` it, bake the returned key into the frame's markup, then register the frame**:

```go
func (f *Fx) Slides(dir string) {
    key := filepath.Base(dir)
    if v, err := f.ToValue(dir); err == nil {
        f.Route(key, v) // exposed at p.source(key)
    }
    html := execute(slidesHtml, key)     // {{.PREFIX}} = key
    f.Frames = append(f.Frames, f.build(html))
}
```

The frame's script reads that route back by the same key, handed in at build time:

```js
const prefix = '{{.PREFIX}}';           // the route key
p.source(prefix).then((entries) => { /* … */ });
```

`ToValue` → `Route` → embed key → `build`/`Frames` is the whole contract; any builder that needs fetchable data follows it.

### Panel frames

The **panel** is a strip appended below the universe, toggled with `z`, hidden by default. Its frame pool travels alongside the frame pool in the `/` payload. Build a panel frame with `p.Panel(path)` — same consolidation as `p.Frame`, but appends to the panel pool instead of the frame pool. A panel frame authors its own root div exactly like a space frame; it renders into `p.universe.panel.el`, not a space. Registering more panels appends after any already registered, so `p.universe.panel`'s index cycles through all of them.

### Ordering: `sort.txt`

Bundle entries carry **no filenames on the wire — order is the contract.** A directory route's order defaults to filesystem walk order. To pin it, place a `sort.txt` in the directory: one file stem (name without extension) per line. Listed files come first, in that order; unlisted files follow. `sort.txt` itself is never included in the bundle.

```
cover
intro
alpha
```

Frames reference bundle entries **by index**, so `sort.txt` is how data and companion files stay aligned. When the agent designs a data structure alongside a bundle (e.g. records referencing images), it should reference entries by index and emit the matching `sort.txt`.

### Wire format

One binary format, both directions:

```
[1B typeCount]
typeCount × [1B typeLen][type]        // string table: distinct MIME types
repeated until EOF:
  [1B typeID][4B dataLen (big-endian)][data]
```

Distinct types are written once, up front; each entry then costs a 1-byte type id and a 4-byte length. There is no entry count — the response's length is the terminator. Names are never encoded.

### Build-time transforms

The frame file authored under [The frame file](#the-frame-file) is not served as-written — `fx` consolidates it once at build time: styles are hoisted to the top (when a frame has more than one block), every script body is wrapped in `{ }` if it isn't already, and all scripts are moved to the end. This happens once, before `Serve()` freezes the routes — not per request.

---

## one — serving

`one` turns a fully registered `pathless` into two running listeners: it encodes every route into the wire format, gzips each once, and serves them from memory. It holds no `zero`/`fx` state of its own — it receives their outputs and assembles the response.

### Construction

```go
p := pathless.NewPathless()                                  // development
p := pathless.NewPathless("timefactory.io", "api.timefactory.io") // production
```

- **No arguments** — localhost. The HTML shell is served on `:1000`, the wire gateway on `:1001`, CORS open (`*`).
- **Two arguments** — `origin` (the domain serving the shell) and `circuit` (the wire gateway host). Both are assumed HTTPS; CORS on the gateway is restricted to the origin. Any other argument count is fatal.

### What goes between `NewPathless()` and `Serve()`

Every call in between registers something — into `Fx`'s frame/panel pools, or into `Fx`'s route map (`Routes`) — and returns immediately. Nothing here talks to the network or renders anything; `Serve()` is the one point where all of it gets frozen: wire-encoded and gzip-compressed once, then served from memory for the life of the process. So this section of `main.go` is a **declaration** of what the shell contains, not a sequence of runtime actions.

What goes here, in practice:

- **Space frames** — `p.Home(...)`, `p.Text(...)`, `p.Slides(...)`, `p.Frame(path)`. Registration order is navigation order: `q`/`e` page through frames in exactly the order they were registered, since the wire carries no names, only position.
- **Routes for a frame's own data** — `p.ToValue(path)` + `p.Route(key, v)`, whenever a hand-authored `p.Frame(...)` will call `p.source(key)` for data a built-in template doesn't already register internally (`Slides`/a non-`.svg` `Logo` do this for you). Must happen before `Serve()`, since that's the point routes get frozen:

  ```go
  cat, _ := p.ToValue("./data/catalog.json")
  p.Route("catalog.json", cat) // registers route "catalog.json"
  pics, _ := p.ToValue("./pics")
  p.Route("pics", pics)        // registers route "pics" (directory -> nested bundle)
  p.Frame("./catalog.html")    // its script reads both back via p.source(...)
  ```

- **Persisting a route for reuse** — `p.Save(key)`, once `key`'s route is registered, gob-encodes its `Value` to `s3/<key>` so it can be synced to object storage, instead of every deploy re-reading local files:

  ```go
  pics, _ := p.ToValue("./pics")
  p.Route("pics", pics) // registers route "pics" from local files
  p.Save("pics")        // writes s3/pics — sync this to a bucket separately

  // later, any deploy (this one, or a different app entirely):
  p.Slides("https://bucket.example.com/pics") // ToValue(url) fetches the bucket URL directly
  ```

- **Extra panels** — `p.Panel(path)`, called again for each additional panel; anything registered here is appended after any already registered.

Order only matters among calls that share a pool (frames, panels) — it's the sole ordering signal the client has, since nothing is named on the wire. `ToValue`/`Save` calls need no particular order relative to each other, only relative to `Serve()` (must come before it) and, for `Save`, relative to the `Route` call for the route being saved.

### Templates (the brainless path)

Before authoring a custom frame, check whether the data maps cleanly onto a built-in — if it does, only the one-line `main.go` registration is needed, no HTML authoring:

| Builder                 | Input                                                                                         | Produces                                                                                                                                 |
| ----------------------- | --------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `p.Home(logo, heading)` | `.svg` (inlined), local image (registered via `ToValue`), or `https://` image; heading string | centered logo + `<h1>`                                                                                                                   |
| `p.Text(path)`          | markdown file (local or `https://`)                                                           | rendered HTML, `w`/`s` to scroll, scroll position persisted                                                                              |
| `p.Slides(dir)`         | image directory (local), or `https://` URL                                                    | full-screen viewer; tap halves or `a`/`d` to page; index persisted. Internally `ToValue(dir)` + `Route(key, v)`, then references the key |

`Home`, `Text`, and `Slides` register **space frames**.

A keyboard panel — a live map of the shell's reserved keys, layout, and focus — is built by `p.Keyboard()`, but nothing calls it automatically: a program must call `p.Keyboard()` itself to get one registered (see [Panel frames](#panel-frames)).

### `Serve()`

```go
p.Home("./logo.svg", "Title")
cat, _ := p.ToValue("./data/catalog.json")
p.Route("catalog.json", cat)
pics, _ := p.ToValue("./pics")
p.Route("pics", pics)
p.Frame("./catalog.html")
p.Serve()
```

`p.Serve()` is the last call. At that point every registered route is wire-encoded and gzip-compressed **once**; requests are served from memory. Route map:

| Route    | Content                                                                                                                                                                           |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/`      | a 1-byte frame-count header; the universe HTML; every space frame; every panel frame — all `Encode`d together as one flat list (see [fx](fx/README.md#fxgo--framepanel-building)) |
| `/<key>` | one `Value` per registered route (`p.Route`), keyed by `filepath.Base(path)`                                                                                                      |

Route entries that are themselves directories (e.g. a `Slides` bundle) hold `Children`; `one` encodes those into a nested blob — the client decodes an entry's data again to recover the individual files.

---

## rules at a glance

Every rule above, in one place — all of them correctness requirements. (Hand-wrapping a script in `{ }` is the one thing that isn't a rule here: `build()` does it for you, except for scripts served outside the build path — see [Authoring rules](#authoring-rules).)

| Rule                                                                                          | Why                                                                               |
| --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Query your subtree via `p.universe.frame`, never `document` or `space.el`                     | containment — the frame owns below its root; the shell owns the space and above   |
| Classes only, never `id`                                                                      | `id`s are the shell's namespace; one frame may render into several spaces at once |
| Author the shell statically; fill slots with `textContent`/`src`, never `innerHTML` with data | static-shell model + injection safety                                             |
| `cqw`/`cqh` only, never `vw`/`vh`; no `width`/`height` on the root                            | split/quad layouts share the screen — viewport units size the wrong box           |
| Input via `p.input.bind`; no `addEventListener` for navigation/taps/keys                      | one delegated table routes named events; stacking listeners is redundant          |
| Read state before binding input; write before `sync()`                                        | state must be current when the frame renders                                      |
| Pin `read`/`write` (`p.universe.pin()`) for any deferred state access                         | `focused` is only correct synchronously at construction                           |
