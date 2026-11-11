This is a **spatial, stateful runtime**: content changes within independently navigable screen regions instead of moving between URL-based pages.

**Core Model**
- **Universe:** the whole application surface.
- **Space:** a record containing a persistent DOM element, navigation index, frame collection, and per-index state maps. There are three main spaces and one panel space.
- **Frame:** executable HTML mounted into a space.
- **Panel:** a frame from a separate collection, displayed in the fourth space.
- **Focus:** the space that receives keyboard input and navigation commands when no target is specified. The focus cycle includes every visible space, including the panel.
- **Layout:** the arrangement of visible spaces, independent of their content.

**Browser Layers**
1. **Pathless owns content, composition, and state.** [zero/pathless.js](zero/pathless.js) discovers the space elements, fetches frame and panel HTML, converts it into reusable fragments, and mounts clones. It also controls visibility, focus, layouts, circular navigation, and per-frame state. There is no separate Universe class.

2. **Input recognizes and dispatches interactions.** [zero/input.js](zero/input.js) installs delegated pointer, click, and keyboard listeners and exposes `register`, `x`, `y`, and `click` on the shared `pathless` instance. Bindings belong to DOM elements; an active pointer interaction retains its originating space and mounted frame identity.

3. **The shell supplies geometry and default controls.** [zero/zero.html](zero/zero.html) defines the four space containers, flex layouts, bottom panel region, and startup sequence. Application-wide input bindings are registered here rather than inside Input.

**Serving and Startup**
- The Go entry point, [pathless.go](pathless.go), embeds `fx.Fx`, exposing `Input`, `Frame`, `Save`, and `Circuit` on `Pathless`. Its own `Start` method assembles the HTTP layer after content registration.
- [zero/zero.go](zero/zero.go) embeds the HTML shell and both JavaScript files. `Origin(circuitURL string)` renders the supplied circuit URL into the shell and returns uncompressed HTML bytes. Zero does not create an HTTP handler or open a network listener.
- `NewPathless` selects `http://localhost:1001` as the circuit and `*` as the allowed shell origin with no arguments. Two arguments specify shell and circuit hosts, each prefixed with `https://`; the allowed origin is passed to `one` for CORS. The browser replaces `localhost` in the circuit URL with its current hostname.
- [fx/fx.go](fx/fx.go) owns content import, frame/panel collections, and serialization. `Circuit()` returns a snapshot of gzip-encoded payloads: the empty key contains JSON `[frames, panels]`, and the remaining keys contain public resource payloads. It owns no mux, HTTP registration, or server.
- [one/one.go](one/one.go) owns the shell and circuit muxes. `NewOne(origin, circuit, pathlessURL)` registers the supplied payloads and HTTP behavior; `Start` serves the shell on `:1000` and blocks serving the circuit on `:1001`. The shell redirects non-root paths and nonempty queries to `/`; circuit responses retain gzip headers and CORS.
- On `DOMContentLoaded`, the browser creates `window.pathless`, constructs Input, registers default controls, then calls `pathless.init()`. Helpers are therefore available before any frame scripts execute.
- Initialization assigns the main collection to the first three spaces and the panel collection to the fourth, then mounts index zero in every space with available content, including hidden spaces. Initially only the first main space is visible. Empty collections are left unmounted.

**Composition Rules**
- `one(space = pathless.focused)` selects a space and toggles between single-space mode and the previous split arrangement, when one exists.
- `two()` shows the first two main spaces and alternates between side-by-side (`x`) and stacked (`y`) arrangements. If the third main space was focused, focus moves to the first.
- `three()` shows all three main spaces and cycles the `up`, `right`, `down`, and `left` arrangements.
- `pathless.layout` stores `[current, previous]`. The current value is `null` for single-space mode, `0` or `1` for the two-space arrangements, and `2` through `5` for the three-space arrangements. There is no public `layout()` method.
- `pathless.observable` returns all non-hidden spaces. The `show` and `hide` setters accept a space or array of spaces; `hide` never hides the focused space.
- `panel()` toggles the fourth space; `panel(true)` shows it and `panel(false)` requests hiding it. The panel occupies a separate bottom region. Because it can receive focus, a focused panel cannot be hidden until focus moves elsewhere.
- `focus(direction = 1, space = pathless.focused)` advances through observable spaces; `focus(0, space)` selects a visible space. `navigate(direction = 1, space = pathless.focused)` wraps within that space's collection and remounts only when its index changes.
- Navigation reconstructs frame DOM while retaining state for each space/frame-index pair. Hiding a populated space retains its DOM.

**Input Rules**
- `pathless.register(triggers, callback, el = pathless.el)` registers case-insensitive trigger aliases. Frame bindings take precedence over universe bindings; explicit click targets resolve their own binding. Registration returns `pathless` for chaining.
- `x(left, right)` optionally binds `a`/`left` and `d`/`right` to the current frame element. `y(up, down)` optionally binds `w`/`up` and `s`/`down`. Frames may use either helper, both, or neither.
- `click(el, callback)` registers delegated activation on an element. The nearest registered ancestor owns the click; these targets do not start pointer gestures.
- Default controls are Tab for focus, `q`/`e` and horizontal swipes for navigation, `z` and bottom-corner taps for the panel, and `1`/`2`/`3` for `one()`/`two()`/`three()`.
- Pointer gestures operate on their originating visible space. Successful non-corner recognition focuses that space at completion, not at pointer-down. Bottom-corner activation preserves the existing focus. Other taps invoke directional bindings relative to the space's center; dominant vertical movement is ignored.
- Input accepts one primary pointer interaction at a time and uses pointer capture. Cancellation, capture loss, a replaced frame, a hidden originating space, or changed universe geometry discards the gesture. Internal space movement is not separately checked.
- Prevented events and editable controls are ignored. Keyboard handling also ignores composition and Ctrl/Meta/Alt shortcuts. Matched callbacks prevent the browser default before invocation.

**Frame Contract**
- `pathless.frame(space)` returns `{ space, el, state }`, where `el` is the space's first element child and `state` is a lazily created `Map` for its current frame index. Without an explicit space, it uses the synchronously rendering space, otherwise the focused space.
- Capture that context during synchronous initialization for use after an `await` or in later callbacks. Navigation defaults to focus unless a space is passed explicitly.
- Give each frame an element root. New frame DOM needs new input registrations; element-keyed bindings are held in a `WeakMap`, not in persistent frame state.
- Input callbacks receive `{ trigger, space, el, start, end }`. The trigger is lowercase, `el` is the resolved binding owner, and normal functions receive that element as `this`. Gestures supply universe-normalized coordinate pairs; keyboard and click callbacks receive `null` for `start` and `end`.
- `pathless.source(path, raw = false)` returns a `Promise<ArrayBuffer>` for a circuit resource. By default it caches the request by path and removes failed requests from the cache; `raw = true` bypasses caching. It does not decode content trees or create blob URLs.
- Frames execute trusted code in the shared document. Rendering does not automatically clean up custom listeners, timers, or observers.

The central distinction is: **spaces own navigation and state; frames supply content and behavior; layouts supply geometry; Input supplies dispatch.** Pathless manages the spatial runtime, and the shared `pathless` object connects it to frames through chainable controls, input registration, frame context, and resource requests.