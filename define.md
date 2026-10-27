This is a **spatial, stateful runtime**: content changes within independently navigable screen regions instead of moving between URL-based pages.

**Core Model**
- **Universe:** the whole application surface.
- **Space:** a persistent container with its own navigation position and frame states.
- **Frame:** executable HTML mounted into a space.
- **Panel:** a frame from a separate collection, displayed in the fourth space.
- **Focus:** the visible space that receives commands when no target is specified.
- **Layout:** the arrangement of visible spaces, independent of their content.

**Three Layers**
1. **Pathless loads and executes content.** It fetches and caches binary content trees, decodes HTML, exposes other assets as blob URLs, and replaces a container’s contents with an executable HTML fragment. Defined in pathless/zero/zero.html.

2. **Universe manages composition and state.** It controls visibility, focus, layouts, and circular navigation. Each space retains a `Map` for every frame index it visits. Navigating reconstructs the frame’s DOM, but preserves that stored state. Defined in pathless/zero/universe.html.

3. **Pointer translates input into commands.** Pressing a space focuses it; horizontal swipes navigate it; tapping the universe’s center toggles the panel; other taps invoke frame-local directional bindings. Registered buttons receive delegated clicks without starting gestures.

**Composition Rules**
- Layout `0`: one selected space, with restoration of the previous split layout.
- Layout `1`: two content spaces, with horizontal and vertical variants.
- Layout `2`: three content spaces, with four arrangements.
- The fourth space is independently toggled as a panel and can occupy the whole surface when selected alone.

The central distinction is: **spaces own navigation and state; frames supply content and behavior; layouts supply geometry.** The shared `pathless` object connects those concerns through a small, chainable API.