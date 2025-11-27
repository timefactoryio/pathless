# pathless

Viewport allocator for a pathless domain. 

![layouts](./content/layout.gif)

## Overview

Within **pathless**, the viewport is a closed system with limits defined by a perimeter border. **pathless** establishes an unobstructed interface with two universal components:

 - `panel`: space in the system
 - `frame`: object in space 

`frame`'s are a finite pool of simulataneously observable content, cached after first fetch. 

## Getting Started

| Key   | Action         | Toggle                                     |
| ----- | -------------- | ------------------------------------------ |
| `1`   | One panel      | fullscreen <-> previous layout             |
| `2`   | Two panel      | horizontal <-> vertical                    |
| `3`   | Three panel    | large panel left -> top -> right -> bottom |
| `Tab` | Cycle focus    | panel zero -> one -> two                   |
| `q`   | previous frame |                                            |
| `e`   | next frame     |                                            |

When in a multipanel layout, press `1` to make the focused panel fullscreen, press `1` again to return to the previous layout. Press `2` to toggle between side-by-side (vertical split) and stacked (horizontal split). Press `3` to cycle through 50/25/25 layouts.

![nav](./content/nav.gif)

## Documentation

The `window.pathless` object provides the API coordinating between `panels`, `frames`, and `state`.    

#### `pathless.context()`
Returns the DOM element of the focused panel, DOM element of the current frame, and panel specifasdasdasdic frame state.

#### `pathless.fetch(url, opts)`  
Returns the parsed response `{ data, headers }`. Caching and request deduplication available using `opts.key` where a single successful round-trip makes a `value` available to all panels.

#### `pathless.onKey(handler)`
Event handler used to register `frame` keybinds, automatically scoped to the focused panel.

#### `pathless.update(key, value)`
Key-value pair's used to persist state through layout changes and navigation, automatically scoped to the `frame` of the focused `panel`.

### Architecture

Pathless is a lightweight Go HTTP server that embeds a frictionless viewport allocator. The server processes, minifies, and compresses an HTML template once at startup, then serves it from memory with maximum efficiency. 

**Server responsibilities:**
- Client delivery
- Redirect all non-root paths to `/`

**Client responsibilities:**
- Fetch frames from a configurable API endpoint
- Manage multi-panel layouts with keyboard navigation
- Cache frames and deduplicate requests
- Provide state management for loaded frames

## What It Does

Pathless is a lightweight web server that:

1. **Builds template** with Title, ApiUrl, and Favicon environment values
2. **Minifies the HTML** by removing comments, whitespace, and newlines
3. **Compresses with gzip** for optimal transfer size
4. **Serves from memory** - All processing happens **once** during initialization. 

### Client (JavaScript)

```
┌──────────┐    ┌──────────┐    ┌──────────┐
│   One    │───▶│    Fx    │───▶│   Zero   │
│Controller│    │  Layout  │    │  Cache   │
└──────────┘    └──────────┘    └──────────┘
     │               │                │
     │               │                │
  Keyboard      Panel State      HTTP Fetch
  Events        Management       & Caching
```
