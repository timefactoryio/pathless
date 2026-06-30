# pathless frame specification

This document specifies how to build a **frame** for pathless. A frame is a single `.html` file containing `<style>`, optional markup, and a `<script>`. It renders any bytes (JSON, images, text) into an interactive view. An AI agent reading this file should be able to produce a correct frame for an arbitrary data structure without further context.

---

## 1. Registration (Go `main.go`)

The developer writes a `main.go` that registers frames and data routes, then calls `Serve()`.

```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless()

    // built-in template frames
    p.Home("./logo.svg", "Title")    // logo + heading
    p.Text("./readme.md")            // markdown → html
    p.Slides("./pics")               // image viewer (also registers "pics" route)

    // custom data frame
    p.Load("./data/catalog.json")    // route "catalog.json" (base name of path)
    p.Load("./pics")                 // route "pics"         (directory → bundle)
    p.CustomHTML("./catalog.html")   // frame, wrapper <div class="catalog">

    p.Serve()
}
```

Rules an agent must follow when emitting main.go:

- `p.CustomHTML(path)` registers a frame. The wrapper `<div>` class is the filename without extension (`catalog.html` → `class="catalog"`). **Local files only.**
- `p.Load(path)` exposes a file or directory as a fetchable route. **The route key is `filepath.Base(path)`** — `./data/catalog.json` → `catalog.json`, pics → pics. A directory becomes a bundle of all its files.
- Data a frame reads via `p.source(...)` **must** be registered with `p.Load(...)` (or by a template builder that loads internally, e.g. `Slides`).
- Built-in builders: `Home(logo, heading)`, `Text(path)`, `Slides(dir)`. Each accepts a local path or an `https://` URL.

---

## 2. Data access (client `p.source`)

```js
p.source(key) // → Promise<Array<{ name: string, url: string }>>
```

- `key` is the route key **without a leading slash**: `p.source('catalog.json')`, `p.source('pics')`.
- Each entry: `name` is the filename (with extension), `url` is an object URL (`blob:`) to that file's bytes.
- The promise is cached per key — calling it repeatedly causes one fetch.
- A single-file route (e.g. a JSON file) yields a **one-element array**.

### Reading JSON

```js
const [entry] = await p.source('catalog.json');
const data = await fetch(entry.url).then((r) => r.json());
```

### Reading an image set and mapping by name

Image references in data often omit the extension (e.g. `"image": "alpha"`), while bundle entries include it (`alpha.avif`). Build a lookup keyed by the stem:

```js
const pics = await p.source('pics');
const img = Object.fromEntries(
    pics.map(({ name, url }) => [name.replace(/\.[^.]+$/, ''), url]),
);
// img['alpha'] === 'blob:...'
```

### Loading JSON and images together

```js
const [[file], pics] = await Promise.all([
    p.source('catalog.json'),
    p.source('pics'),
]);
const data = await fetch(file.url).then((r) => r.json());
const img = Object.fromEntries(
    pics.map(({ name, url }) => [name.replace(/\.[^.]+$/, ''), url]),
);
```

---

## 3. Frame anatomy

A frame file has three parts. The Go `Build` step **automatically**: hoists every `<style>` to the top, wraps every `<script>` body in a `{ }` block, and moves all scripts to the end. Authoring contract:

```html
<style>
  /* scoped to .<framename>; use cqw / cqh units only */
</style>

<!-- optional static markup; becomes children of <div class="<framename>"> -->

<script>
{
  class Frame {
    constructor(p) {
      // 1. read persisted state
      // 2. build / query DOM under p.space.el
      // 3. register p.input.tap / p.kb.keyNav
      // 4. load async data, then render
    }
  }
  new Frame(pathless);
}
</script>
```

### Hard rules

| Rule                                                                        | Reason                                                                           |
| --------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Wrap script body in `{ }`                                                   | scripts re-execute on every render; block scope prevents redeclaration errors    |
| No `id` attributes                                                          | the same frame may render into multiple spaces simultaneously; ids would collide |
| Query DOM via `p.space.el.querySelector(...)`, never `document`             | isolates per-space DOM                                                           |
| Never call `addEventListener` for navigation/tap/keys                       | use `p.input.tap` / `p.kb.keyNav`; the shell owns input                          |
| Read state before registering input; write state before any `sync()`        | state must be current at render time                                             |
| Build DOM with `createElement` + `textContent`, never `innerHTML` with data | prevents injection                                                               |
| All sizing in `cqw` / `cqh`                                                 | spaces are containers; `vw`/`vh` break in multi-space layouts                    |
| Do not set `width`/`height` on the frame root                               | the space sizes it (see §5)                                                      |

---

## 4. APIs available inside a frame

`window.pathless` (passed as `p`) is the only global.

| Member               | Signature                           | Behavior                                                      |
| -------------------- | ----------------------------------- | ------------------------------------------------------------- |
| `p.space.el`         | `HTMLElement`                       | the current space's element; root for all queries and appends |
| `p.read()`           | `() => Map`                         | per-frame, per-space state map; survives re-render            |
| `p.write(k, v)`      | `(string, any) => void`             | persist `k → v` into the state map                            |
| `p.source(key)`      | `(string) => Promise<[{name,url}]>` | fetch + decode a route; cached                                |
| `p.sync()`           | `() => void`                        | re-render visible spaces                                      |
| `p.input.tap(fn)`    | `(g => void) => void`               | register tap handler for the focused space                    |
| `p.kb.keyNav(binds)` | `(object) => void`                  | register key handlers for the focused space                   |

### `p.input.tap(fn)`

Fires when a touch ends with negligible movement (not a swipe). `fn` receives:

```js
g = {
  end:       [x, y, t], // x,y normalized to [-1,1] within the space; t = ms timestamp
  direction: 'left' | 'right' | 'up' | 'down' | null,
  quadrant:  1 | 2 | 3 | 4, // 1=TR, 2=TL, 3=BL, 4=BR
  distance:  number,
  axis:      'x' | 'y',
}
```

`g.end[0] < 0` → left half, `> 0` → right half. Re-registering replaces the handler.

### `p.kb.keyNav(binds)`

Active only while its space is focused. `down` repeats while held; `up` fires on release; both optional.

```js
p.kb.keyNav({
  a: { down: () => this.prev() },
  d: { down: () => this.next() },
  w: { down: () => this.scroll(-1), up: () => this.stop() },
});
```

Reserved keys handled by the shell (do not bind): `q` `e` (nav), `1` `2` `3` (layout), `tab` (focus), `z` (panel).

### State semantics

State is a `Map` keyed to the (space, frame) pair. The same frame in two spaces has two independent maps. Persist only serializable view state (indices, scroll offsets, toggles).

```js
this.index = p.read().get('index') ?? 0;
// ...
p.write('index', this.index);
```

---

## 5. Layout contract (CSS the frame inherits)

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
- 
Pattern summary an agent should generalize:
- One small class per record type; recurse for nested arrays (here `options`).
- Resolve image stems to blob URLs once, before constructing DOM.
- Build all DOM with `createElement` + `textContent` (never `innerHTML` with data).
- Append the built tree under `p.space.el.querySelector('.<framename>')`.
- Persist only view state through `p.read()` / `p.write()`.

---

## 6. Template builders (when not writing a custom frame)

If the data maps cleanly onto a built-in, register a builder instead of authoring HTML:

| Builder                 | Input                                                                     | Produces                                                                                                            |
| ----------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `p.Home(logo, heading)` | `.svg` (inlined), local image (blob), or `https://` image; heading string | centered logo + `<h1>`                                                                                              |
| `p.Text(path)`          | markdown file (local or `https://`)                                       | rendered HTML, `w`/`s` to scroll, scroll persisted                                                                  |
| `p.Slides(dir)`         | image directory (local or `https://`)                                     | full-screen viewer; tap halves or `a`/`d` to page; index persisted. Internally calls `Load(dir)` and `source(base)` |

These require no `.html` authoring — only the main.go registration line.
