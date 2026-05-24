# pathless — AI Builder Reference

## What is pathless?

pathless is a Go + HTML framework for building keyboard-driven, fullscreen applications delivered as a single compressed binary payload over HTTP. There is no JavaScript build step, no bundler, no framework runtime. The server sends gzipped binary; the browser decodes and renders it.

The visual surface is a `#universe` div containing three **spaces**: `#zero`, `#fx`, `#one`. Each space holds one **frame** at a time. Frames are plain HTML fragments — `<style>`, `<script>`, markup — injected via `innerHTML` and re-executed on every render.

---

## Server Architecture

pathless runs two HTTP servers:

- `:1000` — serves `pathless.html`, the application shell. One response, never changes.
- `:1001` — serves the hello bundle (`/`), all frame HTML, and all binary asset routes.

`pathless.html` is templated with `window.apiUrl = '{{.APIURL}}'` which resolves to the API server address (default `http://localhost:1001`). Every client fetch — including the initial hello bundle — goes to `:1001`. The shell and the API are deliberately separate servers.

---

## The Universe Shell

`pathless.html` is served once at page load and never changes. It provides:

- The global `state` object
- The global `cache` Map
- `window.pathless` — the `Universe` instance (available after `DOMContentLoaded`)

### `state`

```js
state = {
    layout: 0,       // 0=single, 1=split, 2=quad
    variant: 0,      // layout variant index
    focused: 0,      // index of the focused space (0=zero, 1=fx, 2=one)
    prev: null,      // [layout, variant] before last change
    frames: [],      // [{ html, binds }] — all loaded frames
    spaces: [        // one entry per space
        { frameIndex: 0, frameState: [] },
        { frameIndex: 0, frameState: [] },
        { frameIndex: 0, frameState: [] },
    ],
}
```

### `cache`

A `Map` shared across all frames. Use it to cache fetched assets, blob URLs, or any computed value that should survive a frame re-render.

```js
cache.set('myKey', value);
cache.get('myKey');
cache.has('myKey');
```

### `window.pathless` API

| Method / Property                        | Description                                                                                         |
| ---------------------------------------- | --------------------------------------------------------------------------------------------------- |
| `pathless.read()`                        | Returns a shallow copy of the current frame's persisted state                                       |
| `pathless.write(k, v)`                   | Persists a key/value into the current frame's state (survives re-render)                            |
| `pathless.bind(key, down, { up, desc })` | Registers a keyboard binding for the current frame                                                  |
| `pathless.source(url)`                   | Fetches a binary asset route, decodes it via `wire()`, returns a Promise of entries. Cached by URL. |
| `pathless.toggle()`                      | Shows/hides the keyboard panel                                                                      |

---

## Frame Lifecycle

1. `render(i)` is called for space `i`
2. `space.innerHTML = frame.html` — the raw HTML string is injected
3. `exec(el)` clones all `<script>` and `<style>` tags so they execute fresh
4. The frame's constructor code runs synchronously
5. `frame.binds` is reset to `[]` on every render — keys must be re-bound each time
6. On the next `render()`, the DOM is replaced — no cleanup callbacks exist

**Frames have no persistent DOM.** State must be saved via `pathless.write()` before render, and restored via `pathless.read()` at construction.

---

## Canonical Frame Pattern

A frame is an HTML fragment with three parts: `<style>`, `<script>` with a block-scoped class, and markup.

```html
<style>
    .frame-name {
        display: flex;
        flex-direction: column;
        /* Only declare internal layout. The space handles all sizing. */
    }
</style>
<script>
    {
        class FrameName {
            constructor() {
                // 1. Query DOM
                this.root = document.querySelector('.frame-name');

                // 2. Restore persisted state
                const s = pathless.read();
                this.index = s.index ?? 0;

                // 3. Bind keys synchronously
                pathless.bind('j', () => this.next());
                pathless.bind('k', () => this.prev());

                // 4. Load data (async, non-blocking)
                pathless.source('/my-data').then(entries => {
                    this.data = entries;
                    this.render();
                });
            }

            next() {
                this.index = (this.index + 1) % this.data.length;
                pathless.write('index', this.index);
                this.render();
            }

            render() { /* update DOM */ }
        }

        new FrameName();
    }
</script>
<div class="frame-name">
    <!-- markup -->
</div>
```

### Rules

- Always use a **block scope** `{ }` — frame scripts share the global scope
- Use a **class** — it makes the constructor phases explicit and readable
- Never use `id` on frame elements — IDs must be unique in the document and frames re-render
- Use **classes** for all selectors
- Restore state before binding keys so key handlers have correct initial values
- Load async data in `.then()` after synchronous setup is complete
- If multiple instances of the same frame type exist, use a unique key prefix with `pathless.write` to avoid state collisions

---

## CSS — The Space Handles Sizing

The space (`#zero`, `#fx`, `#one`) already sets everything the frame root needs:

```css
:is(#zero, #fx, #one) {
    flex: 1; overflow: hidden; container-type: size;
    position: relative; isolation: isolate;
    display: flex; flex-direction: column;
}
:is(#zero, #fx, #one) > * { width: 100%; }
:is(#zero, #fx, #one) > :first-child { flex: 1; min-height: 0; }
```

The frame root is the first child of the space. It inherits:

- `width: 100%` — from `> *`
- Full height via `flex: 1; min-height: 0` — from `> :first-child`
- `box-sizing: border-box` — global rule
- `overflow: hidden`, `position: relative`, `isolation: isolate` — from the space

**The frame root needs no required CSS.** Only declare properties that define the frame's own internal layout:

```css
.frame-name {
    display: flex;
    flex-direction: column;
    padding: 2cqw;
    /* nothing else needed */
}
```

Use `cqw`/`cqh` for all responsive sizing. Each space has a `container-name` (zero, fx, one) and `container-type: size`. Since the frame root fills the space exactly, `cqw`/`cqh` inside the frame resolve to the frame's own dimensions. Prefer `cqw`/`cqh` over `vw`/`vh` — container units stay correct when the universe layout changes and spaces are resized.

---

## Wire Protocol

All asset routes are served as `Content-Type: application/octet-stream` + `Content-Encoding: gzip`. The client calls `pathless.source(url)` which fetches, decodes, and caches the result.

### Two formats, one `wire()` method

`bytes[0]` is a flag byte:

**`0x00` — meta format** (named asset bundles, served at `/assetname`)
```
[0x00][4B: manifest-len][JSON: [{name, type, size}, ...]][blob]...[blob]
```
Returns entries with `{ name, type, size, data }`.

**`0x01` — positional format** (hello bundle, served at `/`)
```
[0x01][4B: size][blob][4B: size][blob]...
```
Returns entries with `{ data }` only — positions are meaningful, names are not.

The client `wire()` handles both transparently:

```js
wire(buf) {
    const bytes = new Uint8Array(buf);
    const view = new DataView(buf);
    let pos = 1;
    if (bytes[0] === 0x00) {
        pos += 4 + view.getUint32(1);
        return JSON.parse(this.#td.decode(bytes.subarray(5, pos)))
            .map(e => ({ ...e, data: bytes.subarray(pos, (pos += e.size)) }));
    }
    const result = [];
    while (pos < bytes.length) {
        const size = view.getUint32(pos);
        result.push({ data: bytes.subarray((pos += 4), (pos += size)) });
    }
    return result;
}
```

### Hello bundle structure (`0x01`)

The bundle at `"/"` is positional. Entry 0 is always the keyboard panel HTML. Entries 1..n are frame HTML strings in order. The universe `init()` decodes them by position — no manifest needed.

---

## Frame-Specific Backends

A frame is not coupled to pathless's own data layer. It only needs a URL. The frame is built specifically around the known structure of what that URL returns — the schema is implicit, co-designed between the frame and its backend.

Regardless of where data comes from, the frame is a full citizen of the universe. It always has access to `pathless.read()`, `pathless.write()`, `pathless.bind()`, `state`, `cache`, and the rest of the Universe API. The backend is just a data source — state management, key handling, and lifecycle are all provided by the shell.

**Served through pathless** (registered on the same server, returns the wire format):

```js
// pathless.source() fetches, decodes wire format, caches by URL
pathless.source('/my-data').then(entries => {
    this.items = JSON.parse(new TextDecoder().decode(entries[0].data));
    this.render();
});
```

**External service** (any URL outside the pathless build — JSON API, WebSocket, anything):

```js
// wire() is not involved — fetch and handle the response directly
fetch('https://api.example.com/data')
    .then(r => r.json())
    .then(data => {
        this.items = data.items;
        pathless.write('index', this.index);
        this.render();
    });
```

No changes to the pathless repo are needed in either case. The backend can aggregate from any source — databases, external APIs, file systems, message queues — and shape the response to exactly what the frame expects. The frame defines the contract; the backend delivers it; the universe manages everything else.

---

## Assets — `pathless.source()` and Blob URLs

`pathless.source(url)` returns a Promise of decoded entries and caches the Promise by URL. Calling it multiple times with the same URL makes only one fetch.

For binary assets that need a `src` attribute (images, audio), convert to a blob URL and cache it in `cache`:

```js
const src = img.dataset.src;
const blobKey = src + '#blob';
if (cache.has(blobKey)) {
    img.src = cache.get(blobKey);
} else {
    pathless.source(src).then(([entry]) => {
        const url = URL.createObjectURL(new Blob([entry.data], { type: entry.type }));
        cache.set(blobKey, url);
        img.src = url;
    });
}
```

The `#blob` suffix prevents key collision between the wire Promise and the blob URL string.

---

## Go Server Structure

```
fx/
  main.go      — Fx struct, NewFx(), Routes map
  circuit.go   — Encode(), Compress(), Load(), ToBytes(), read(), walk()
  forge.go     — Forge interface, One type, Build(), Builder(), Frames(),
                 element helpers, consolidateAssets()
  templates.go — Home(), Text(), Slides(), App(), Logo()
  templates/   — frame HTML templates (home, slides, text, app)
one/
  main.go      — One struct, binaryHandler, BuildHello(), Register(), Serve()
zero/
  main.go      — Zero struct, pathless.html minification + gzip
  core/
    pathless.html — universe shell
    keyboard.html — keyboard panel HTML
```

### Two asset registries

**`fx.Routes`** (`map[string][]byte`) holds binary asset bundles. `fx.Load(path)` reads a file or directory, encodes with `meta=true` (`0x00` format), compresses, and stores keyed by base name. `Register()` mounts every entry in `Routes` via `binaryHandler` on the API server.

**`forge.frames`** (`[]*One`) holds HTML frame strings. `Build()` and `Builder()` append to this slice. `Frames()` returns all accumulated frames as `[][]byte`. `BuildHello()` reads from `Frames()` — not from `Routes` — to assemble the hello bundle.

`BuildHello()` encodes with `Encode(values, false)` (`0x01` format) — keyboard panel HTML first, then all frame HTML blobs in order.