I'm in Ask mode — I can't write files. Here's the content to paste into `mechanicsv2.md`:

```markdown
# pathless — Mechanics

## What is pathless?

pathless is a Go + HTML framework for fullscreen applications delivered as a single binary payload over HTTP. No JavaScript build step, no bundler, no framework runtime.

The shell is `pathless.html` — served once, never changes. It bootstraps a `Pathless` instance that fetches a binary hello bundle, mounts an input script, and renders frames into spaces.

---

## Visual Structure

`#universe` contains three **spaces**: `#zero`, `#fx`, `#one`. Each space holds one **frame** at a time. Layout and visibility are controlled via `data-layout` and `data-variant` attributes on `#universe`.

```html
<div id="universe" data-layout="0" data-variant="0">
    <div id="zero"></div>
    <div id="fx" hidden></div>
    <div id="one" hidden></div>
</div>
```

---

## Layouts

`#layouts` is a static 3×3 positional config — each row maps `[zero-hidden, fx-hidden, one-hidden]` to a layout index.

| layout | spaces visible  | variants                  |
| ------ | --------------- | ------------------------- |
| `0`    | zero only       | 1                         |
| `1`    | zero + fx       | 2 (side-by-side, stacked) |
| `2`    | zero + fx + one | 4 (grid arrangements)     |

`layout(l, v)` writes `data-layout`/`data-variant` to `#universe` and sets `.hidden` on each space via `#layouts[l].forEach((h, i) => space[i].el.hidden = h)`.

`cycle(l)` computes the next variant for layout `l`, wrapping at the variant count `[1, 2, 4][l]`. Called with no argument it reads the current layout from `dataset`. If called with `l === 0` and `prev` exists, it restores the previous layout instead.

`prev` is a `[l, v]` tuple stored on `universe` whenever `layout(l > 0)` is called. It enables a fullscreen→restore cycle.

---

## Universe Object

```js
universe = {
    el,        // #universe element
    cache,     // Map — shared fetch cache across all frames
    prev,      // [l, v] | null — last non-zero layout
    space,     // [{ el, frame }, { el, frame }, { el, frame }]
    panel,     // hidden div for the input panel
    frames,    // [{ el: htmlString, state: Map }] — set during init
}
```

`space` is built from `['zero', 'fx', 'one'].map(id => ({ el: getElementById(id), frame: null }))`. Each entry holds a direct element reference and the current frame object. `space[i]` is the canonical address — layout uses index, nav uses index, sync uses index.

---

## Frame Object

```js
frame = {
    el,     // raw HTML string of the frame
    state,  // Map<spaceId, Map<k, v>> — per-space persisted state
}
```

All spaces share frame objects from `universe.frames`. Multiple spaces can point to the same frame. A frame's `state` map is keyed by space id (`'zero'`, `'fx'`, `'one'`) so each space has independent state even when displaying the same frame.

---

## `window.pathless` API

| Method                      | Description                                                                                       |
| --------------------------- | ------------------------------------------------------------------------------------------------- |
| `pathless.layout(l, v = 0)` | Go to layout `l`, variant `v`. Saves `prev` if `l > 0`.                                           |
| `pathless.cycle(l?)`        | Advance variant for layout `l` (default: current). Restore `prev` if `l === 0` and `prev` exists. |
| `pathless.nav(dir, i)`      | Move space `i` to the next/prev frame by direction `±1`. Wraps.                                   |
| `pathless.read(i)`          | Returns the state `Map` for space `i`'s current frame.                                            |
| `pathless.write(k, v, i)`   | Sets `k → v` in space `i`'s frame state.                                                          |
| `pathless.source(url)`      | Fetch + wire-decode a binary route. Cached by URL. Returns a Promise.                             |
| `pathless.sync(...i?)`      | Re-render spaces. No args = all visible spaces.                                                   |

---

## Frame Lifecycle

1. `sync(i)` is called for space `i`
2. `space.el.innerHTML = frame.el` — raw HTML string injected
3. `exec(space.el)` — all `<script>` and `<style>` tags cloned fresh so they execute
4. Frame constructor runs synchronously
5. On next `sync()`, the DOM is fully replaced — no cleanup callbacks

**Frames have no persistent DOM.** Use `pathless.read(i)` / `pathless.write(k, v, i)` to survive re-renders.

---

## Wire Protocol

`wire(buf)` decodes the binary hello bundle. Format: 2-byte big-endian count, then N entries each with:
- 1-byte name length + name bytes (`undefined` if length is 0)
- 1-byte type length + type bytes
- 4-byte big-endian u32 data length + data bytes

Entry 0 is the input script HTML. Entries 1..n are frame HTML strings.

`source(url)` wraps `wire()` with a fetch and caches the Promise by URL.

---

## Init Sequence

1. Fetch hello bundle from `window.apiUrl`
2. Decode with `wire()`
3. Extract entry 0 as input HTML — append a temp div to `body`, set `innerHTML`, call `exec()`, remove div (scripts must be live in the DOM to execute)
4. Map remaining entries to frame objects via `#newFrame()`
5. Point all spaces to `frames[0]`
6. Call `sync()` to render

---

## CSS — The Space Handles Sizing

```css
:is(#zero, #fx, #one) {
    flex: 1; overflow: hidden; container-type: size;
    position: relative; isolation: isolate;
    display: flex; flex-direction: column;
}
:is(#zero, #fx, #one) > * { width: 100%; }
:is(#zero, #fx, #one) > :first-child { flex: 1; min-height: 0; }
```

The frame root is the first child of its space. It gets `width: 100%`, full height, `box-sizing: border-box`, and `overflow: hidden` for free. Use `cqw`/`cqh` for all responsive sizing — each space has a `container-name` and `container-type: size`, so container units resolve to the space's own dimensions and stay correct when layout changes.

---

## Canonical Frame Pattern

```html
<style>
    .frame-name {
        display: flex;
        flex-direction: column;
        padding: 2cqw;
    }
</style>
<script>
    {
        class FrameName {
            constructor(i) {
                this.i = i;
                this.root = document.querySelector('.frame-name');
                const s = pathless.read(i);
                this.index = s.get('index') ?? 0;
                pathless.source('/my-data').then(entries => {
                    this.data = entries;
                    this.render();
                });
            }
            render() { /* update DOM */ }
        }
        new FrameName(/* space index passed by input handler */);
    }
</script>
<div class="frame-name"></div>
```

### Rules

- Always use a block scope `{ }` — frame scripts share global scope
- Never use `id` on frame elements — IDs must be unique; frames re-render
- Use classes for all selectors
- `pathless.read(i)` returns a `Map` — use `.get(k)` / default with `?? fallback`
- Load async data in `.then()` after synchronous setup
