## Plan: Space-Scoped Input

`Pathless` loads and starts the root; `Universe` owns spaces and render context; `Input` exposes a fixed set of directional/gesture hooks that frames cherry-pick from — a frame never binds a raw key or pointer name. Gestures and held keys belong to the space where they began, and every rerender resolves that space's outstanding input before frame scripts register replacements.

### 1. Ownership graph

`Universe` receives the `Pathless` instance; `Input` receives the `Universe` instance. Both classes read their dependencies from constructor arguments, not the `pathless` global. The root fragment constructs `pathless.universe` first, then `pathless.input`. `Pathless.init()` loads the root and frames and executes the root; startup after `init()` only triggers `universe.sync()` — no shell binding table, no retained/unused panels collection.

`bind()` is a private implementation detail `Input` uses internally. It is never frame-facing; frames only ever call the public hooks in §3.

**Where `Universe` and `Input` actually meet**, and how that overlap is consolidated rather than duplicated across both classes:

1. **Space context is resolved once, not per hook.** Every registration (`up`, `swipe.left`, …) and every dispatch needs "which space is this for right now," and `Input` never decides that itself — it always asks `universe.space` (`#rendering` during frame exec, `active`/focused otherwise). This is the single most-repeated interaction in the class, so all ten public hook methods funnel through one private `Input.#register(name, fn)` helper that resolves `universe.space` and stores against it once, rather than each method re-deriving the same thing.
2. **Per-space storage lives on the space object, not in a second map.** Binding tables (`up/down/left/right/swipe/horizontal/vertical`) are space-scoped data, and `Universe` already owns the canonical space objects (`this.spaces`) and already attaches other per-space state to them (`frame`, `state`). So the bindings table is a property on each space object itself (`space.bindings`) rather than a parallel `Map<space, bindings>` inside `Input`. One canonical list of spaces, with everything attached to them — not two classes each indexing the same three objects separately.
3. **Geometry stays entirely in `Universe`, split into two small queries instead of one combined one.** Coordinate normalization depends on `Universe.rect` (wired to `ResizeObserver`), so only `Universe` should ever touch it. The original `point()` conflated hit-testing, normalization, *and* focus mutation in one call; this plan already strips the focus mutation out (`Input.pointerdown()` does that explicitly, per §4), and goes one step further by splitting the rest: `Universe.hitTest(clientX, clientY)` (which space is under this point) and `Universe.normalize(clientX, clientY)` (pure coordinate math) are separate, independently-callable methods. `Input` calls `hitTest` once, at `pointerdown`, to establish ownership; it calls `normalize` on every `pointermove` for the continuous `live`/`traveled` tracking §4/§5 need. Neither call depends on the other, so there's no reason to force them through one method the way `point()` used to.
4. **Rerender is the one place `Universe` calls into `Input`.** `Universe.sync()` calls `Input.clear(space)` directly, right after `#rendering = current`, before `Pathless.exec()` — the sole point where the direction of the call reverses. `Input` never indexes its held-key/active-pointer state *by* space (there are only ever a handful of live gestures at once), so `clear(space)` is a cheap filter over "whichever keys/pointers are currently owned by this space," not a lookup into some second per-space structure — no reason to add one just for this.
5. **Built-in commands are a thin, one-directional adapter.** `1`/`2`/`3`/Tab are the only place `Input` calls straight into `Universe`'s own methods (`layout()`, `focus()`) rather than routing through a space's hook table (§7). Not shared machinery in the same sense as 1–4 — just the third, narrowest category of interaction between the two classes.

Net boundary: **`Universe` owns spaces, geometry, and render/focus state; `Input` owns gesture/key state and event routing.** The only things that cross the line are the space object itself (as a lookup key and storage target) and the two narrow geometry queries (`hitTest`, `normalize`) — everything else on either side stays put.

### 2. The one shared mechanic: single-axis accumulation

Every gesture in this system — tap, swipe, and two-pointer pinch/spread — reduces to the same rule: compare `abs(x)` to `abs(y)` on whatever vector is relevant, let the larger one win as the **dominant axis**, and discard the other axis entirely. Nothing ever resolves diagonally.

Movement is never captured as a stored start point subtracted from a later point. Instead, `Input` keeps one **running signed accumulator** per gesture, fed by each `pointermove`'s dominant-axis delta since the previous tick (the losing axis's delta for that tick is simply dropped). Swipe and pinch/spread both use this exact primitive — a pointer's own accumulator for swipe, the *separation* between two pointers' positions for pinch/spread — they differ only in what feeds it and when it's read.

One tunable constant, `SWIPE_THRESHOLD` (normalized distance, default `0.25`), is the sole magnitude threshold behind all of it:
- **Swipe** — has the pointer's accumulator, read once at release, passed `SWIPE_THRESHOLD` on its dominant axis?
- **Tap dead zone** — is the release point's *absolute* position (plane-center-relative, not accumulated — tap cares where you lifted off, not how you got there) past `SWIPE_THRESHOLD` on its dominant axis? Below it, a tap is inert — it does not force an arbitrary direction.
- **Pinch/zoom step size** — has the two-pointer separation's accumulator reached `SWIPE_THRESHOLD` on its dominant axis? (Read continuously, not just at release — see §5.)

Coordinates are normalized against the whole `#universe` rect, not the individual space's rect — universe-wide, on purpose. `SWIPE_THRESHOLD` is sized so that any layout's smallest space is still large enough for a meaningful gesture to clear it under universe-wide coordinates.

Classification is distance-only everywhere; no velocity or timing is measured anywhere in this system.

### 3. Public API

```
p.input.up(fn)      // fn(isOn: boolean)
p.input.down(fn)    // fn(isOn: boolean)
p.input.left(fn)    // fn(isOn: boolean)
p.input.right(fn)   // fn(isOn: boolean)

p.input.swipe.up(fn)     // fn()
p.input.swipe.down(fn)   // fn()
p.input.swipe.left(fn)   // fn(), also bound to Q
p.input.swipe.right(fn)  // fn(), also bound to E

p.input.horizontal(fn)   // fn(direction: 1 | -1)
p.input.vertical(fn)     // fn(direction: 1 | -1)
```

All ten hooks register through one private `#register(name, fn)` helper (§1.1) that resolves `universe.space` and writes into that space's own `bindings` object (§1.2). Each hook is independent and optional — a frame wires only the subset it needs (e.g. only `up`/`down` for scroll-hold, or only `swipe.left`/`swipe.right` for paging). Unregistered hooks stay `null` and dispatch no-ops.

Three callback shapes, one per gesture family:
- **`up`/`down`/`left`/`right`** — an on/off boolean. Becoming "on" (keydown, or a tap once classified) calls `fn(true)` once; becoming "off" (keyup, or a tap's synthesized release) calls `fn(false)` once. There is no `{start, hold, end}` phase table and no native-repeat re-firing (`event.repeat` is ignored) — a frame that wants continued action while held polls its own `true` state.
- **`swipe.*`** — a single momentary `fn()`, no arguments, no on/off state. Triggered by a single-pointer drag on all four directions. `swipe.left`/`swipe.right` are additionally triggered by `Q`/`E` keydown (non-repeat) — same hook, same momentary fire, just a second input source for the horizontal pair only. There is no `swipe.up`/`swipe.down` key equivalent.
- **`horizontal`/`vertical`** — a discrete `fn(1 | -1)` step, fired by `Input` itself each time §2's threshold is crossed (not driven by the frame). A frame commonly pairs `up`/`down` with `left`/`right`, or `vertical` with `horizontal`, as two independent array-navigation axes — a usage convention the API enables, not something `Input` enforces.

### 4. Single-pointer gestures (tap / swipe)

`Input.pointerdown()` calls `Universe.hitTest()` to find and focus the hit space, captures the pointer, and initializes that pointer's state: `live` (its normalized position, obtained via `Universe.normalize()` and updated the same way on every subsequent `pointermove`), a dominant-axis `traveled` accumulator seeded at zero, and the owner space (whatever's focused at that instant). There is no separate "press point" stored — `live` and `traveled` together are sufficient for everything downstream, including tap, which only ever needs `live`'s absolute value at release, never where it started.

On release, apply §2's rule:
- `traveled` past `SWIPE_THRESHOLD` on its dominant axis → **swipe**: fire `swipe.<direction>()` once, direction from `traveled`'s sign. This is the only place `traveled` is read for a single pointer — it accumulates the whole gesture but is never dispatched mid-drag.
- Otherwise, `live`'s absolute position past `SWIPE_THRESHOLD` on its dominant axis → **tap**: fire `fn(true)` then `fn(false)` back-to-back (a tap is only known after release, so both phases fire together).
- Otherwise → dead zone, nothing fires.

Always dispatches to the starting (owner) space, regardless of where release occurs. `pointercancel` discards all of that pointer's state with no dispatch.

### 5. Two-pointer gestures (pinch / spread)

If a second pointer touches down while a first is still tracking a potential tap/swipe, that single-pointer gesture's state is discarded with no dispatch (as if cancelled), and the pair becomes a two-pointer gesture owned by whatever space is focused at that moment.

While both pointers are down, each `pointermove` recomputes horizontal separation (`abs(live₁.x - live₂.x)`) and vertical separation (`abs(live₁.y - live₂.y)`) from the two pointers' `live` values (§4), and feeds the change in each since the last tick into that axis's own accumulator (§2) — the same primitive a single pointer uses for `traveled`, just fed a separation instead of a raw position. §2's dominant-axis rule picks a winner per tick — only the winning axis's accumulator advances, the other's delta is dropped, so `horizontal` and `vertical` can never both progress from the same movement.

This is where pinch/spread differs from swipe: **its accumulator is read continuously, not deferred to release.** Each axis fires `fn(1)` (spreading) or `fn(-1)` (pinching) the instant its accumulator reaches `SWIPE_THRESHOLD`, then resets by one threshold-width (remainder carried forward) — so a single long, axis-aligned gesture fires several steps in a row, live, as it happens.

The gesture ends the instant either pointer lifts or cancels; the remaining pointer, if any, does not resume single-pointer tracking — a fresh `pointerdown` cycle is required to start anything new.

This is the only multi-touch gesture in this pass (no rotation, no three-plus-finger gestures). A future pass may have `Universe` itself consume the same underlying two-pointer tracking to drive layout/variant switching directly — that is a separate, later built-in, not routed through `horizontal`/`vertical`, and out of scope here.

### 6. Clearing on rerender

`Input.clear(space)` runs from `Universe.sync()` right after `#rendering = current`, before `Pathless.exec()`, and resolves that space's outstanding input before its `bindings` are cleared off the space object (§1.2):
- A key currently "on" and owned by the space fires `fn(false)`.
- A single-pointer gesture in progress is resolved through the exact §4 release logic, using that pointer's current `live`/`traveled` state as if this instant were the release — clearing reuses the same read, not a separate cancellation rule.
- A two-pointer gesture simply stops receiving further steps; each axis's accumulator, whatever it's currently holding below `SWIPE_THRESHOLD`, is discarded, same as a tap's dead zone.

### 7. Built-ins vs. frame controls

`1`/`2`/`3` map directly to `Universe.layout(0..2)`; Tab advances focus only among `Universe.visible` and prevents default. Both are global commands handled outside any per-space binding map, and are the only such global commands.

All other keyboard input routes through the *focused* space's hooks: Arrow keys/WASD to `up/down/left/right`, and Q/E to `swipe.left`/`swipe.right`. Q/E fire the single-shot `swipe.*` callback directly on keydown, targeting whichever space is focused at that instant (no on/off pairing, no repeat-on-hold, no owner tracking needed since there's nothing to release later). Arrow/WASD are different: the owner space is captured at keydown and a later keyup still fires `false` there even if focus has since moved on, since `up/down/left/right` has real held state to close out (§6 covers this at clear time; the same rule applies on an ordinary keyup). There is no `forward`/`backward`-named method — `swipe.left`/`swipe.right` (mouse/touch or Q/E) fully covers that need.

### 8. Frame migration

- `/frames/zero/frame/slides.html` — call `swipe.left`/`swipe.right` (and/or `left`/`right` for tap/key paging) directly; handlers are now isolated to the rendering space instead of overwriting other frames globally.
- `/frames/zero/frame/text.html` — replace its obsolete `down`/`up` phase-object keys with `up(fn)`/`down(fn)` using the on/off boolean form.

**Relevant files**
- `/pathless/zero/one.html` — `Universe`/`Input` implementation: dependencies, geometry (`hitTest`/`normalize`), render lifecycle, space objects carrying their own `bindings`, per-pointer `live`/accumulator state, single- and two-pointer gesture handling.
- `/pathless/zero/zero.html` — `Pathless.init()`, removal of the shell binding table and unused panels state.
- `/frames/zero/frame/slides.html` — verify against the hook API.
- `/frames/zero/frame/text.html` — migrate to the on/off form.

**Verification**
1. Editor diagnostics on all four touched files.
2. Layout zero renders initially; `1/2/3` switch/cycle layouts; Tab changes focus only when multiple spaces are visible.
3. With conflicting input consumers in two or three visible spaces: WASD/arrows act only on the focused space; tapping a different space focuses it and fires only its own `on`/`off` pair.
4. Hold a directional key, change focus, release — original owner space gets `off(false)`, not the new focus. Rerender while held — `Input.clear()` resolves it the same way.
5. Tap inside the dead zone — nothing fires. Tap outside it — one `on`/`off` pair, classified by release position. Drag past threshold in each cardinal direction, including releasing over another space — exactly one `swipe.*` call to the starting space, no tap also firing. Press Q and E on the focused space — `swipe.left`/`swipe.right` fire once each, immediately, with no drag needed; holding either does not repeat-fire.
6. Start a drag, rerender mid-gesture before releasing — `Input.clear()` resolves it via the live tracked position, no drop, no double-fire.
7. Start a drag, then touch a second pointer before releasing — first gesture discarded with no dispatch, two-pointer gesture begins.
8. Spread/pinch on a space bound to `horizontal`/`vertical` — steps fire only on threshold crossings, not every tick; a long axis-aligned gesture fires several steps in a row with correct sign; stepping stops the instant either pointer lifts; sub-threshold movement fires nothing; a diagonal drag never fires both axes; a frame using both hooks can step two independent arrays.
9. Navigate from a frame with bindings to one without — the replaced frame's controls no longer respond.

**Scope boundaries**
- No panel/keyboard-overlay feature restored — only unused panel retention in `Pathless` is removed.
- No gesture framework beyond §2–§5's tap/swipe/pinch classification.
- No multi-touch beyond the two-pointer `horizontal`/`vertical` gesture; no `Universe`-level layout/variant pinch in this pass.
- No changes to binary decoding, source caching, frame state persistence, layouts, or visual styling.