# pathless frame specification

pathless renders **frames** — single `.html` files built for data the developer already knows: its shape, its cardinality, its natural interactions. The data can be anything expressible as bytes — JSON, markdown, images, CSV, logs, binary formats, generated structures — there is no schema requirement. There's also no separate mobile path: one `Input` model resolves mouse, touch, and pen into the same events, and one sizing unit (`cqw`/`cqh`) fits a frame to whatever container it's given. A frame is authored once and rendered wherever it's observed.

This document is the contract for building **one frame**: its file, what it can call at runtime, how it's registered, and how it's served. Frames can be observed side by side in multiple spaces at once — that's a property of the shell's layout system, not something a frame needs to author for, so it isn't covered here.

Because the data is known at build time, the frame should be designed *for* it — not as a generic viewer. Match interactions to the data's shape: paging for sequences, taps on halves for prev/next, held keys for scrubbing/scrolling, per-record expansion for hierarchies.

The three sections below follow the Go packages a frame touches, in the order it touches them: **zero** (what the frame calls at runtime), **fx** (how the frame's file is authored and registered), **one** (how it's templated and served).

---

## The frame file

A frame is one `.html` file with three parts: `<style>`, static markup (the shell), and a `<script>`. The shell — including its root container div — is authored as divs directly in the HTML. Because the data is known ahead of time, its structure (root, sections, lists, an `<img>` per slot, …) is written statically. The script does **not** build the DOM; it queries the shell, wires interaction, and updates the dynamic slots (`textContent`, `src`, `hidden`, dataset/state attributes).

```html
<style>
  /* scoped to .<framename>; use cqw / cqh units only */
</style>

<!-- the shell: root container + static structure mirroring the known data -->
<div class="catalog">
  <div class="viewer"><img></div>
  <div class="detail"><p class="desc"></p><span class="count"></span></div>
</div>

<script>
{
  class Frame {
    constructor(p) {
      // 1. pin state, read persisted values
      // 2. query the shell under p.universe.space.el
      // 3. register p.input.bind
      // 4. load async data, then populate the shell
      this.el = p.universe.space.el.querySelector('.catalog');
      this.img = this.el.querySelector('.viewer img');
      this.desc = this.el.querySelector('.desc');
    }
  }
  new Frame(pathless);
}
</script>
```

### Hard rules

| Rule                                                                                                                         | Reason                                                                                                                                                 |
| ---------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Wrap script body in `{ }`                                                                                                    | scripts re-execute on every render; block scope prevents redeclaration errors (the build wraps unwrapped scripts, but author them wrapped)             |
| No `id` attributes                                                                                                           | the same frame may render into multiple spaces simultaneously; ids would collide                                                                       |
| Query DOM via `p.universe.space.el.querySelector(...)`, never `document`                                                     | isolates per-space DOM                                                                                                                                 |
| Never call `addEventListener` for navigation/tap/keys                                                                        | use `p.input.bind`; the shell owns input                                                                                                               |
| Read state before registering input; write state before any `sync()`                                                         | state must be current at render time                                                                                                                   |
| Capture `p.universe.pin()` at construction; use its `read`/`write` in every async callback (promises, event listeners, bind) | the default focused index is only valid synchronously during construction — see **State semantics** under [zero](#zero--what-a-frame-calls-at-runtime) |
| Author the shell as static markup, including the root container div                                                          | data is known at build time — structure belongs in HTML, behavior in script                                                                            |
| Update dynamic slots via `textContent` / `src`, never `innerHTML` with data                                                  | prevents injection                                                                                                                                     |
| All sizing in `cqw` / `cqh`, never `vw`/`vh`                                                                                 | a frame is sized by its own container, not the viewport — see **Sizing** under zero                                                                    |
| Do not set `width`/`height` on the frame root                                                                                | the container that hosts the frame already sizes it                                                                                                    |

---

## zero — what a frame calls at runtime

`zero` is the browser runtime a frame executes inside. `window.pathless` (passed as `p`) is the only global; `p.universe` (including `p.universe.panel`), `p.input`, and `p.keyboard` (the default keyboard panel, always registered) are the modules attached to it.

| Member                       | Signature                                | Behavior                                                                                  |
| ---------------------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------- |
| `p.source(key)`              | `(string) => Promise<[{type,data,url}]>` | fetch + decode a route; cached per key                                                    |
| `p.universe.space.el`        | `HTMLElement`                            | the currently focused space's element; root for all queries                               |
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

    // custom data frame: expose the data, register the frame
    p.Load("./data/catalog.json") // route "catalog.json" (base name of path)
    p.Load("./pics")              // route "pics"         (directory → bundle)
    p.Frame("./catalog.html")     // frame; file authors <div class="catalog">

    p.Serve()
}
```

Rules an agent must follow when emitting `main.go`:

- `p.Frame(path)` registers a space frame. The file authors its own root container div, classed by convention after the filename stem (`catalog.html` → `<div class="catalog">…</div>`). Nothing is wrapped for you. The content is trusted as-is — **local, developer-controlled files only.**
- `p.Load(path)` exposes a file or directory as a fetchable route. **The route key is `filepath.Base(path)`** — `./data/catalog.json` → `catalog.json`, `./pics` → `pics`. A directory becomes a bundle of all its files. Any file type works — the wire carries typed bytes, and the frame decides how to decode them. MIME type is inferred from extension, with content sniffing as fallback.
- `p.Load(url)` with an `http(s)://` URL expects a **pre-encoded wire blob** (produced by `Save`) and decodes it back into a bundle. It is not for arbitrary remote files.
- `p.Save(key)` writes the wire encoding of a loaded route to `s3/<key>`, ready to sync to object storage (e.g. `rclone sync s3 remote:bucket`). A saved route round-trips: `Save("slides")` → upload → `p.Slides("https://bucket.example.com/slides")`.
- Every route a frame reads via `p.source(...)` **must** be registered with `p.Load(...)` (or by a template that loads internally).

### Panel frames

The **panel** is a strip appended below the universe, toggled with `z`, hidden by default. Its frame pool travels as item 2 of the `/` payload — always present, since a keyboard panel is registered automatically (see [Templates](#templates-the-brainless-path)) — and is rendered by `p.universe.panel` (`toggle()`). Build a panel frame with `p.Panel(path)` or `p.BuildPanel(elements...)` — same consolidation as space frames — then register it by passing the result to `p.Panels(...)`, e.g. `p.Panels(myPanel)`. A panel frame authors its own root div exactly like a space frame; it renders into `p.universe.panel.el`, not a space. Registering more panels appends after the keyboard, so `p.universe.panel`'s index cycles through all of them.

### Ordering: `sort.txt`

Bundle entries carry **no filenames on the wire — order is the contract.** A directory route's order defaults to filesystem walk order. To pin it, place a `sort.txt` in the directory: one file stem (name without extension) per line. Listed files come first, in that order; unlisted files follow. `sort.txt` itself is never included in the bundle.

```
cover
intro
alpha
```

Frames reference bundle entries **by index**, so `sort.txt` is how data and companion files stay aligned. When the agent designs a data structure alongside a bundle (e.g. records referencing images), it should reference entries by index and emit the matching `sort.txt`.

### Wire format

One binary format, both directions. A bundle flattens to its leaves in order; per leaf, back-to-back:

```
[1B typeLen][type] [4B dataLen (big-endian)][data]
```

There is no leaf count — the response's length is the terminator. Names are never encoded.

### Build-time transforms

The frame file authored under [The frame file](#the-frame-file) is not served as-written — `fx` consolidates it once at build time: styles are hoisted to the top (when a frame has more than one block), every script body is wrapped in `{ }` if it isn't already, and all scripts are moved to the end. This happens once, before `Serve()` freezes the routes — not per request.

---

## one — templates and serving

`one` decides whether a frame needs to be authored at all, and turns a fully registered `pathless` into two running listeners.

### Construction

```go
p := pathless.NewPathless()                                  // development
p := pathless.NewPathless("timefactory.io", "api.timefactory.io") // production
```

- **No arguments** — localhost. The HTML shell is served on `:1000`, the wire gateway on `:1001`, CORS open (`*`).
- **Two arguments** — `origin` (the domain serving the shell) and `circuit` (the wire gateway host). Both are assumed HTTPS; CORS on the gateway is restricted to the origin. Any other argument count is fatal.

### Templates (the brainless path)

Before authoring a custom frame, check whether the data maps cleanly onto a built-in — if it does, only the one-line `main.go` registration is needed, no HTML authoring:

| Builder                 | Input                                                                              | Produces                                                                                         |
| ----------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `p.Home(logo, heading)` | `.svg` (inlined), local image (auto-`Load`ed), or `https://` image; heading string | centered logo + `<h1>`                                                                           |
| `p.Text(path)`          | markdown file (local or `https://`)                                                | rendered HTML, `w`/`s` to scroll, scroll position persisted                                      |
| `p.Slides(dir)`         | image directory (local), or `https://` URL of a `Save`'d bundle                    | full-screen viewer; tap halves or `a`/`d` to page; index persisted. Internally calls `Load(dir)` |

`Home`, `Text`, and `Slides` register **space frames**.

A default keyboard panel — a live map of the shell's reserved keys, layout, and focus — is registered automatically by `NewPathless`/`NewOne`; there's no builder call needed to get it (see [Panel frames](#panel-frames)). `p.Keyboard()` still exists to build the same panel frame on demand, if something other than automatic registration ever needs it.

### `Serve()`

```go
p.Home("./logo.svg", "Title")
p.Load("./data/catalog.json")
p.Load("./pics")
p.Frame("./catalog.html")
p.Serve()
```

`p.Serve()` is the last call. At that point every registered route is wire-encoded and gzip-compressed **once**; requests are served from memory. Route map:

| Route    | Content                                                                                                                                                                                                |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `/`      | item 0: the universe HTML; item 1: every space frame, pre-encoded as one nested bundle; item 2: every panel frame (keyboard plus any registered via `p.Panels(...)`), pre-encoded as one nested bundle |
| `/<key>` | one bundle per `Load`ed path, keyed by `filepath.Base(path)`                                                                                                                                           |

Item 0 is used as-is; items 1 and 2 are themselves wire-encoded — the client decodes them a second time to recover the individual frames.
