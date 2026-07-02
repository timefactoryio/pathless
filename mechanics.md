# pathless frame specification

pathless renders **frames** — single `.html` files that visualize and interact with data the developer knows ahead of time. The data can be anything expressible as bytes: JSON, markdown, images, CSV, logs, binary formats, generated structures. There is no schema requirement.

This document is the contract for an AI agent working with a developer. Given the developer's data (content, structure, and intent), the agent must:

1. **Prefer a template.** If the data maps cleanly onto a built-in (`Home`, `Text`, `Slides`), emit only the one-line `main.go` registration — no HTML authoring.
2. **Otherwise author a custom frame** via `CustomHTML`: design an HTML shell for visualizing, engaging with, and interacting with that specific data, following the rules below.

Because the data is known at build time, the frame should be designed *for* it — its shape, its cardinality, its natural interactions — not as a generic viewer.

---

## 1. Path one: templates (the brainless path)

| Builder                 | Input                                                              | Produces                                                                                                            |
| ----------------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------- |
| `p.Home(logo, heading)` | `.svg` (inlined), local image, or `https://` image; heading string | centered logo + `<h1>`                                                                                              |
| `p.Text(path)`          | markdown file (local or `https://`)                                | rendered HTML, `w`/`s` to scroll, scroll persisted                                                                  |
| `p.Slides(dir)`         | image directory (local), or `https://` URL of a `Save`'d bundle    | full-screen viewer; tap halves or `a`/`d` to page; index persisted. Internally calls `Load(dir)` and `source(base)` |

If one of these fits, stop here — only the `main.go` line is needed.

---

## 2. Path two: custom frames — registration (Go `main.go`)

```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless()

    // templates and custom frames compose freely
    p.Home("./logo.svg", "Title")

    // custom data frame: expose the data, register the frame
    p.Load("./data/catalog.json")    // route "catalog.json" (base name of path)
    p.Load("./pics")                 // route "pics"         (directory → bundle)
    p.CustomHTML("./catalog.html")   // frame; file authors <div class="catalog">

    p.Serve()
}
```

Rules an agent must follow when emitting `main.go`:

- `p.CustomHTML(path)` registers a frame. The file authors its own root container div, classed by convention after the filename stem (`catalog.html` → `<div class="catalog">…</div>`). Nothing is wrapped for you. **Local files only.**
- `p.Load(path)` exposes a file or directory as a fetchable route. **The route key is `filepath.Base(path)`** — `./data/catalog.json` → `catalog.json`, `./pics` → `pics`. A directory becomes a bundle of all its files. Any file type works — the wire carries typed bytes, and the frame decides how to decode them.
- `p.Load(url)` with an `http(s)://` URL expects a **pre-encoded wire blob** (produced by `Save`) and decodes it back into a bundle. It is not for arbitrary remote files.
- `p.Save(key)` writes the wire encoding of a loaded route to `s3/<key>`, ready to sync to object storage (e.g. `rclone sync s3 remote:bucket`). A saved route round-trips: `Save("slides")` → upload → `p.Slides("https://bucket.example.com/slides")`.
- Every route a frame reads via `p.source(...)` **must** be registered with `p.Load(...)` (or by a template that loads internally).

### Ordering: `sort.txt`

Bundle entries carry **no filenames on the wire — order is the contract.** A directory route's order defaults to filesystem walk order. To pin it, place a `sort.txt` in the directory: one file stem (name without extension) per line. Listed files come first, in that order; unlisted files follow.

```
cover
intro
alpha
```

Frames reference bundle entries **by index**, so `sort.txt` is how data and companion files stay aligned. When the agent designs a data structure alongside a bundle (e.g. records referencing images), it should reference entries by index and emit the matching `sort.txt`.

---

## 3. Data access (client `p.source`)

```js
p.source(key) // → Promise<Array<{ type, data, url }>>
```

- `key` is the route key **without a leading slash**: `p.source('catalog.json')`, `p.source('pics')`. The empty key is reserved for the shell.
- Each entry:
  - `type` — MIME type string (`image/avif`, `application/json`, `text/plain`, …)
  - `data` — `Uint8Array` of the raw bytes, already in memory
  - `url` — lazy getter; creates a `blob:` object URL from `data` on first access
- There are **no names**. Entries are identified by position (see `sort.txt`).
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

---

## 4. Frame anatomy

A frame file has three parts: `<style>`, **static markup (the shell)**, and a `<script>`. The Go `Build` step **automatically**: hoists every `<style>` to the top, wraps every `<script>` body in a `{ }` block, and moves all scripts to the end.

The shell — including its root container div — is authored as divs directly in the HTML. Because the data is known ahead of time, its structure (root, sections, lists, an `<img>` per slot, …) is written statically. The script does **not** build the DOM; it queries the shell, wires interaction, and updates the dynamic slots (`textContent`, `src`, `hidden`, dataset/state attributes).

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
      // 1. read persisted state
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

| Rule                                                                                                                                                           | Reason                                                                                                                                                                                                                              |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Wrap script body in `{ }`                                                                                                                                      | scripts re-execute on every render; block scope prevents redeclaration errors                                                                                                                                                       |
| No `id` attributes                                                                                                                                             | the same frame may render into multiple spaces simultaneously; ids would collide                                                                                                                                                    |
| Query DOM via `p.universe.space.el.querySelector(...)`, never `document`                                                                                       | isolates per-space DOM                                                                                                                                                                                                              |
| Never call `addEventListener` for navigation/tap/keys                                                                                                          | use `p.input.bind`; the shell owns input                                                                                                                                                                                            |
| Read state before registering input; write state before any `sync()`                                                                                           | state must be current at render time                                                                                                                                                                                                |
| Capture `const i = p.universe.state.focused` at construction; pass `i` explicitly to every `read`/`write` in async callbacks (promises, event listeners, bind) | the default index reflects whichever space is *currently* focused, evaluated at call time — correct only synchronously during construction. An async callback firing later reads the wrong space's default and spills state into it |
| Author the shell as static markup, including the root container div                                                                                            | data is known at build time — structure belongs in HTML, behavior in script                                                                                                                                                         |
| Update dynamic slots via `textContent` / `src`, never `innerHTML` with data                                                                                    | prevents injection                                                                                                                                                                                                                  |
| All sizing in `cqw` / `cqh`                                                                                                                                    | spaces are containers; `vw`/`vh` break in multi-space layouts                                                                                                                                                                       |
| Do not set `width`/`height` on the frame root                                                                                                                  | the space sizes it (see §6)                                                                                                                                                                                                         |

---

## 5. APIs available inside a frame

`window.pathless` (passed as `p`) is the only global. `p` is the thin wire client; `p.universe`, `p.input`, and `p.keyboard` are the modules attached to it.

| Member                       | Signature                                | Behavior                                                                                  |
| ---------------------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------- |
| `p.source(key)`              | `(string) => Promise<[{type,data,url}]>` | fetch + decode a route; cached per key                                                    |
| `p.universe.space.el`        | `HTMLElement`                            | the current space's element; root for all queries and appends                             |
| `p.universe.read(i?)`        | `(number?) => Map`                       | per-(frame, space) state map; survives re-render; defaults to the currently focused space |
| `p.universe.write(k, v, i?)` | `(string, any, number?) => void`         | persist `k → v` into the state map; same default caveat as `read`                         |
| `p.universe.sync()`          | `() => void`                             | re-render visible spaces                                                                  |
| `p.input.bind(binds)`        | `(object) => void`                       | register gesture and key handlers for the focused space                                   |

### `p.input.bind(binds)`

A frame has exactly one way to register input, for both touch and keyboard: `p.input.bind({...})`. Every trigger — tap, swipe, or key — is a named property on one plain object, passed in a single call; calling `bind` again replaces the whole set for the focused space. This exists because a gesture and a key press are, underneath, the same thing: a named event the shell resolves and routes. One object means a frame thinks about *what* should happen, not *where* the input came from, and can freely mix touch and keyboard triggers for the same action.

The value shape tells `bind` what kind of trigger it is:

```js
p.input.bind({
  tapLeft:  () => this.prev(),               // gesture -> plain function, fires once
  tapRight: () => this.next(),
  swipeUp:  () => this.expand(),
  a:        { down: () => this.prev() },      // key -> { down, up }, down repeats while held
  d:        { down: () => this.next() },
  w:        { down: () => this.scroll(-1), up: () => this.stop() },
});
```

Gesture names (`tapLeft`/`tapRight`/`tapTop`/`tapBottom`/`swipeUp`/`swipeDown`) map to a function because a gesture is instantaneous — it either happened or it didn't. Key names (any `e.key.toLowerCase()`) map to `{ down, up }` because a key has duration a frame may need to act on — `down` fires (and repeats) while held, `up` on release.

Reserved names are resolved by the shell before a frame's bindings are ever consulted, so they can't be overridden: `q` `e` `swipeLeft` `swipeRight` (nav — the swipes are gesture aliases of the `q`/`e` keys), `1` `2` `3` (layout), `tab` (focus), `z` (panel).

### State semantics

State is a `Map` keyed to the (frame, space) pair. The same frame in two spaces has two independent maps. Persist only serializable view state (indices, scroll offsets, toggles).

`read()`/`write()` default their space index to `p.universe.state.focused` — but that value is only guaranteed correct **synchronously during construction** (the shell sets it right before executing the frame's script). Any code that runs later — a promise callback, an event listener, a `bind` handler firing on user input — must not rely on the default, because by then a *different* space may be focused. Capture the index once, up front, and pass it explicitly everywhere else:

```js
constructor(p) {
  this.i = p.universe.state.focused;      // capture once, synchronously
  this.index = p.universe.read(this.i).get('index') ?? 0;
  // ...
}
// later, from a promise/event/bind callback:
p.universe.write('index', this.index, this.i);
```

---

## 6. Layout contract (CSS the frame inherits)

The shell provides, for every space `#zero` / `#fx` / `#one`:

```css
:is(#zero, #fx, #one) {
  flex: 1;
  overflow: hidden;
  container-type: size;     /* enables cqw / cqh */
  display: flex;
  flex-direction: column;
}
:is(#zero, #fx, #one) > *            { width: 100%; }
:is(#zero, #fx, #one) > :first-child { flex: 1; min-height: 0; }
```

Consequences for the frame root (`.<framename>`):

- It is the space's first child → already full-width and flex-grown to fill height. **Do not** set its `width`/`height`.
- It is a block in a flex column; give it its own `display: flex` + `flex-direction` for internal layout.
- `1cqw` = 1% of the space's width, `1cqh` = 1% of its height — correct under any layout. Use `clamp(min, Ncqw, max)` for fluid type.
- For a scrollable frame, set `overflow-y: auto` and `height: 100%` on the root.

---

## 7. Designing a frame for known data

Because the data is fixed at build time, the agent should tailor the frame to it rather than generalize:

- Choose interactions that fit the data's shape: paging for sequences, taps on halves for prev/next, `bind` for scrubbing/scrolling, per-record expansion for hierarchies.
- Author the shell to mirror the data: root container div plus one static structure per record type, repeated/nested in the markup as the known data dictates.
- The script queries the shell (`p.universe.space.el.querySelector('.<framename>')`), registers input, and populates dynamic slots via `textContent` / `src`.
- Decode each route's bytes according to its known format (`entry.data` for text/structured data, `entry.url` for media) — never a second network fetch.
- Reference bundle entries by index (`sort.txt` order); emit the `sort.txt` alongside any index-referencing data the agent authors.
- Persist only view state through `p.universe.read()` / `p.universe.write()`.
