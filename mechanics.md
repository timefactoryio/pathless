
# Building with pathless

## What pathless is

pathless is a Go + HTML framework. You write a Go `main.go` that configures content, call `Serve()`, and the browser receives a single binary payload containing everything. No build step, no bundler, no JS framework.

**What pathless handles for you:**
- The HTML shell — served once, bootstraps itself
- The input system — swipe, long-press, tap, keyboard, mobile panel
- Navigation between frames — swipe or keyboard, automatic
- Layout switching — single, double, and triple views, with optional panel
- Wire encoding — all content, regardless of origin, encoded into one binary format and served locally
- State persistence — per-space key/value Map, survives re-render
- Built-in frame types — `Home`, `Text`, `Slides`, `CustomHTML`

**What pathless does not handle:**
- What a frame displays or how it behaves internally
- Routing, URLs, browser history
- Authentication or sessions

---

## Setup

```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless()
    p.Home("https://cdn.example.com/logo.svg", "my app")
    p.Text("https://raw.githubusercontent.com/org/repo/main/readme.md")
    p.Slides("./images")
    p.CustomHTML("./dashboard.html")
    p.Serve()
}
```

`NewPathless()` — development. Runs on `localhost:1000` (shell) and `localhost:1001` (wire gateway). CORS is open.

`NewPathless("origin.com", "api.origin.com")` — production. HTTPS assumed. CORS on the gateway restricted to origin.

`Serve()` blocks on `:1000`. The wire gateway runs on `:1001` in a goroutine.

---

## Content Sources

All content in pathless is resolved at startup on the Go side — either read from the local filesystem or fetched over HTTPS. Once resolved, it is encoded into the wire format and served through the wire gateway. The client always fetches from the same local origin, regardless of where the content originally came from.

**Local path** — read from disk at startup:
```go
p.Text("./readme.md")
p.Slides("./images")
p.Load("./audio.mp3")
```

**HTTPS URL** — fetched from the network at startup, then served locally:
```go
p.Text("https://raw.githubusercontent.com/org/repo/main/readme.md")
p.Slides("https://cdn.example.com/slides")
p.Load("https://cdn.example.com/assets")
```

Both resolve to the same thing from the client's perspective: a wire-encoded binary route on the gateway. `CustomHTML` is the only exception — it accepts local paths only.

---

## The Wire Format

The wire format is how pathless packages all known content — frames, assets, and the input system — into binary payloads served over HTTP.

At startup, `Serve()` assembles the hello bundle: the coordinates script, the input script, the panel script, then all frame HTML strings, encoded in order. This single gzip'd binary is served at `/`. The client fetches it once, decodes it, and has everything needed to bootstrap.

Asset routes (registered via `Load`) are each encoded as a named bundle — a manifest of `{ name, type, size }` entries followed by the raw blobs — gzip'd and served at `/:key`.

In a frame, `p.source(url)` fetches a route from the gateway, wire-decodes it, and caches the Promise by URL. The decoded entries have `{ name, type, size, data: Uint8Array }`:

```js
p.source('/images').then(entries => {
    for (const e of entries)
        e.url = URL.createObjectURL(new Blob([e.data], { type: e.type }));
    this.entries = entries;
});
```

Multiple calls to `p.source()` with the same URL make one fetch. Use `p.universe.cache` to persist blob URLs across re-renders:

```js
const key = name + '#blob';
if (p.universe.cache.has(key)) return p.universe.cache.get(key);
const url = URL.createObjectURL(new Blob([data], { type }));
p.universe.cache.set(key, url);
```

---

## Frames and Spaces

A **frame** is a unit of content — an HTML string held in memory. All frames are registered at startup and form a finite pool. The user navigates between them; all frames exist simultaneously.

A **space** is a display slot — one of three (`#zero`, `#fx`, `#one`). Each space shows one frame at a time and holds its own independent state. If the same frame appears in two spaces simultaneously, each space has entirely separate state.

Navigation (swipe, keyboard `q`/`e`) moves a space through the frame pool. Layout controls how many spaces are visible at once. Frame authors don't implement any of this — it's automatic. A frame's only job is to render correctly and manage its own state within the space it's given.

### Layout reference

| layout | spaces visible  | variants                  |
| ------ | --------------- | ------------------------- |
| `0`    | zero only       | 1                         |
| `1`    | zero + fx       | 2 (side-by-side, stacked) |
| `2`    | zero + fx + one | 4 (arrangement variants)  |

`prev` is a `[l, v]` tuple saved on `universe` whenever `layout(l > 0)` is called. `cycle()` with no argument reads the current layout. If the current layout is `0` and `prev` exists, it restores the previous layout instead of advancing — this is the fullscreen→restore cycle.

### Frame lifecycle

1. `sync(i)` is called for space `i`
2. `space.el.innerHTML = frame.el` — raw HTML string injected
3. `exec(space.el)` — all `<script>` and `<style>` tags cloned fresh and executed
4. Frame constructor runs synchronously
5. On next `sync()`, the DOM is fully replaced — no cleanup callbacks

**Frames have no persistent DOM.** State that must survive re-renders goes in `p.write()`, not in instance variables or the DOM.

---

## Built-in Frame Types

Register frames in order. Navigation wraps through them in registration order.

### `p.Home(logo, heading string)`

Centered logo and title. Shows a toggle button when the panel is hidden and layout is fullscreen.

- `logo` — local `.svg` → inlined; local file → blob via `data-src`; `https://` URL → `<img src>`
- `heading` — rendered as `<h1>`

### `p.Text(path string)`

Markdown rendered to HTML. Full typographic CSS included (`h1`–`h4`, `p`, `pre`, `code`, `table`, `blockquote`, `hr`, images). Keyboard scroll (`w`/`s`). Scroll position persisted per space.

- `path` — local file path or `https://` URL, fetched once at startup

### `p.Slides(dir string)`

Full-screen image viewer. Tap left/right half to navigate, or keyboard `a`/`d`. Images fetched, blob URL'd, and cached. Slide index persisted per space.

- `dir` — local directory path or `https://` URL to an asset bundle, fetched once at startup

### `p.CustomHTML(path string)`

Loads an arbitrary local HTML file as a frame. The filename without extension becomes the wrapper div's class — `dashboard.html` → `<div class="dashboard">`.

- `path` — local file path only
- The file provides only `<style>`, markup, and `<script>` — no outer wrapper div needed

---

## Writing a Custom Frame

A frame file is a fragment: style, markup, script. The wrapper div is injected automatically by `CustomHTML`. Interaction is handled entirely through `p.input.tap` and `p.kb.keyNav` — the space is the interaction surface, not individual elements within it.

```html
<style>
    .dashboard {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 3cqh;
        font-size: clamp(1rem, 3cqw, 2rem);
    }
    .dashboard h1 { font-size: clamp(1.5rem, 5cqw, 3rem); }
    .dashboard .hint { font-size: clamp(0.7rem, 1.8cqw, 1rem); opacity: 0.4; }
</style>

<h1>Dashboard</h1>
<p class="value">0</p>
<p class="hint">tap · a / d</p>

<script>
    {
        class Dashboard {
            constructor(p) {
                this.count = p.read().get('count') ?? 0;
                this.display = p.space.el.querySelector('.dashboard .value');
                this.display.textContent = this.count;

                p.input.tap(p.focused, ([x]) => this.#adjust(x < 0 ? -1 : 1, p));

                p.kb.keyNav(p.focused, {
                    a: { down: () => this.#adjust(-1, p) },
                    d: { down: () => this.#adjust(1, p) },
                });
            }
            #adjust(delta, p) {
                this.count += delta;
                p.write('count', this.count);
                this.display.textContent = this.count;
            }
        }
        new Dashboard(pathless);
    }
</script>
```

**Rules:**
- Always wrap in a block scope `{ }` — all frame scripts share global scope
- Never use `id` attributes — IDs must be unique; frames re-render
- Query DOM from `p.space.el`, not `document` — the same frame can appear in multiple spaces simultaneously
- Use `p.input.tap` for touch and `p.kb.keyNav` for keyboard — input.html owns the event listeners; frames register callbacks into the existing pipeline, never create their own
- Read state before registering input; write state before any `sync()`
- Load async data in `.then()` after all synchronous setup

---

## The Frame Environment

Every frame constructor receives `pathless` as `window.pathless`. It runs synchronously immediately after the frame's HTML is injected into the space.

### Internal shapes

```js
universe = {
    el,      // #universe element
    cache,   // Map — shared blob/fetch cache across all frames
    prev,    // [l, v] | null — last non-zero layout
    space,   // [{ el, frame, co }, ...]
    panel,   // #panel element
    frames,  // [{ el: htmlString, state: Map<spaceId, Map> }]
}

frame = {
    el,     // raw HTML string
    state,  // Map<spaceId, Map<k, v>> — independent state per space
}
```

### Method reference

| Method               | Description                                                                                  |
| -------------------- | -------------------------------------------------------------------------------------------- |
| `p.layout(l, v = 0)` | Go to layout `l`, variant `v`. Saves `prev` if `l > 0`.                                      |
| `p.cycle(l?)`        | Advance variant for layout `l`. Restores `prev` if `l === 0` and `prev` exists.              |
| `p.nav(dir, i?)`     | Move space `i` forward/back through frames. Wraps.                                           |
| `p.read(i?)`         | Returns the state `Map` for space `i`'s current frame.                                       |
| `p.write(k, v, i?)`  | Sets `k → v` in space `i`'s frame state.                                                     |
| `p.source(url)`      | Fetch + wire-decode a binary route. Cached. Returns a Promise.                               |
| `p.sync(...i?)`      | Re-render spaces. No args = all visible spaces.                                              |
| `p.bindPanel(node)`  | Register a DOM node as panel content for the current frame. Restored to default on nav away. |

### `pathless.space`

The current space — shorthand for `universe.space[focused]`:

```js
p.space.el    // the space's DOM element — query all frame DOM from here
p.space.co    // Coordinates instance for this space
p.space.frame // the current frame object
```

### State API

State is a `Map` owned by the space's current frame assignment. It persists across re-renders and layout changes. It is independent per space — the same frame in two spaces has two separate state Maps.

```js
const saved = p.read().get('key') ?? defaultValue;

p.write('key', value);

// Both default to focused space; pass index to target another
p.read(1).get('key');
p.write('key', value, 1);
```

### Re-render

```js
p.sync()      // re-render all visible spaces
p.sync(i)     // re-render space i only
```

---

## Input

input.html installs one set of event listeners on the universe at startup. It handles all gesture recognition — swipe detection, long-press timing, movement thresholds, touch-action conflicts — before any frame loads. Frames never create their own event listeners; they register callbacks into the pipeline that's already running.

### What's automatic

| Gesture / Key                       | Action                                 |
| ----------------------------------- | -------------------------------------- |
| Horizontal swipe (≥25% space width) | `nav(±1)` on the swiped space          |
| Long press (300ms, no movement)     | `toggle()` — show/hide panel           |
| Keyboard `q` / `e`                  | `nav(-1)` / `nav(+1)` on focused space |
| Keyboard `1` / `2` / `3`            | Cycle layouts                          |
| Keyboard `tab`                      | Cycle focused space                    |
| Keyboard `z`                        | `toggle()`                             |
| Mobile panel ← / →                  | `nav(-1)` / `nav(+1)`                  |

### `p.input.tap(i, fn)` — touch interaction

Register a tap handler for space `i`. Fires when a touch ends with minimal movement — mutually exclusive from swipe. Coordinates are normalized `[x, y, z]` in `[-1, 1]` space:

```js
p.input.tap(p.focused, ([x]) => {
    this.show(x < 0 ? this.index - 1 : this.index + 1);
});
```

`x < 0` is the left half, `x > 0` is the right half. One handler per space index.

### `p.kb.keyNav(i, binds)` — keyboard interaction

Register key bindings for space `i`. Only active when that space is focused:

```js
p.kb.keyNav(p.focused, {
    w: { down: () => this.#scroll(-1), up: () => this.#stop() },
    s: { down: () => this.#scroll(1),  up: () => this.#stop() },
    a: { down: () => this.prev() },
    d: { down: () => this.next() },
});
```

`down` fires on keydown and repeats while held. `up` fires on keyup. Both are optional.

---

## CSS Rules

The space handles all sizing for the frame root. You get these for free:

```css
:is(#zero, #fx, #one) > * { width: 100%; }
:is(#zero, #fx, #one) > :first-child { flex: 1; min-height: 0; }
* { box-sizing: border-box; margin: 0; padding: 0; user-select: none; }
```

The frame root needs only internal layout:

```css
.my-frame {
    display: flex;
    flex-direction: column;
    padding: 2cqw;
}
```

**Always use `cqw`/`cqh` for responsive sizing.** Each space has `container-type: size` and a container name (zero, fx, one). Container units resolve to the space's own dimensions and stay correct when layouts resize spaces. `vw`/`vh` will give wrong values in double or triple layouts.
