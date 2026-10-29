This is a **spatial, stateful runtime**: content changes within independently navigable screen regions instead of moving between URL-based pages.

**Core Model**
- **Universe:** the whole application surface.
- **Space:** a persistent DOM container with its own navigation index and frame states. There are three main spaces and one panel space.
- **Frame:** executable HTML mounted into a space.
- **Panel:** a frame from a separate collection, displayed in the fourth space.
- **Focus:** the visible main space that receives keyboard input and navigation commands when no target is specified. The panel is excluded from the focus cycle.
- **Layout:** the arrangement of visible spaces, independent of their content.

**Browser Layers**
1. **Pathless loads and executes content.** It owns the root and space elements, fetches and caches typed content trees, decodes HTML, exposes assets as blob URLs, and inserts executable HTML fragments. Defined in [zero/zero.html](zero/zero.html).

2. **Universe manages composition and state.** It controls visibility, focus, layouts, and circular navigation. Each space retains a `Map` for every frame index it visits. Navigation reconstructs frame DOM while preserving that state; hiding a populated space retains its DOM. Defined in [zero/universe.html](zero/universe.html).

3. **Input translates clicks, keys, and pointer gestures into commands.** A non-bottom press focuses a visible main space; horizontal swipes navigate the pressed space; bottom-edge taps control layouts. Other taps invoke directional bindings relative to the space's center. Registered click targets receive delegated clicks without starting gestures. Defined in [zero/input.html](zero/input.html).

**Serving and Startup**
- The Go entry point, [pathless.go](pathless.go), embeds `fx.Fx`, exposing `Input`, `Frame`, `Save`, and `Start` on `Pathless`.
- [zero/zero.go](zero/zero.go) renders the three browser templates into one compressed HTML payload and starts the shell server on `:1000` during construction.
- [fx/fx.go](fx/fx.go) registers frames, panels, and public resource routes. `Start` serves the circuit on `:1001`; its bootstrap contains frame and panel collections, not runtime HTML.
- The browser creates `window.pathless`, loads those collections, initializes Input, then initializes Universe and renders the first main frame. At least one main frame is required.

**Composition Rules**
- Layout `0`: one selected main space. Calling `layout(0)` without a space toggles between single-space mode and the remembered split layout, when one exists.
- Layout `1`: the first two main spaces, with stacked and side-by-side variants.
- Layout `2`: all three main spaces, with four arrangements.
- Repeating the current split-layout command cycles its variants. The second argument to `layout` selects a space in layout zero, not a variant.
- `panel()` toggles the fourth space as a separate bottom region; `panel(true)` and `panel(false)` set its visibility explicitly. It does not replace the main spaces or become focused.
- Each space navigates independently, wrapping within the main frame or panel collection as appropriate.

**Frame Contract**
- `pathless.frame()` returns `{ space, el, state }` for the synchronously rendering space, otherwise the focused main space. State belongs to a space/frame-index pair, not to the frame globally.
- Capture that context during synchronous initialization for use after an `await` or in later callbacks. Navigation defaults to focus unless a space is passed explicitly.
- Give each frame an element root. Register delegated clicks with `click`, left/right taps and `a`/`d` keys with `x`, and up/down taps and `w`/`s` keys with `y`. New frame DOM needs new registrations.
- `1`/`2`/`3` select layouts, Tab cycles focus, `q`/`e` navigate, and `z` toggles the panel. Bottom-left advances the layout number, bottom-center calls `layout(0)`, and bottom-right repeats the current layout command.
- Frames execute trusted code in the shared document. Rendering does not automatically clean up custom listeners, timers, or observers.

The central distinction is: **spaces own navigation and state; frames supply content and behavior; layouts supply geometry; Input supplies dispatch.** The shared `pathless` object connects them through chainable controls and input registration, with separate context and resource-loading methods. See [README.md](README.md) for the API reference.