# pathless

### is a closed system.

**pathless** prepares one artifact to be used from memory for every request. It has no runtime state, no session awareness, no opinion about content. It is a boundary. It knows how to hold applications without knowing what those applications are. The server's responsibility ends at delivery.

## sequence

Three layers, one direction:

- **zero** — compiles the HTML shell and the universe payload from embedded sources
- **fx** — sources content into frames, panels, and routes; no wire format, no HTTP
- **one** — encodes the wire format, then serves the shell on `:1000` and the wire gateway on `:1001`

```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless() // dev; NewPathless("domain", "api.domain") for prod
    p.Home("./logo.svg", "the point of origin")
    p.Text("./readme.md")
    p.Slides("./pics")
    p.Frame("./custom.html")
    p.Keyboard()
    p.Serve()
}
```

## primitives

#### space: where we observe frames.

Three spaces — `zero`, `fx`, `one` — compose into layouts (single, split, quad) driven by CSS. Each space maintains independent state for each frame.

#### frame: observable object.

Frames are a finite pool of simultaneously observable content: single `.html` files (style, shell, script) delivered over the wire and executed in a space. The domain is the application.

#### wire: order is the contract.

One binary format, both directions. The server encodes; the client decodes. A small type table is written once, up front; each entry then carries a 1-byte type id, a 4-byte length, and its bytes — no names, no paths, no entry count (the response's length is the terminator). Routes are encoded and gzip-compressed once at startup and served from memory. A route can be `Save`d (gob) to `s3/`, synced to object storage, and re-sourced later via `ToValue` of the bucket URL.

## client

The shell boots `window.pathless` — a thin wire client (fetch, decode, render). Following a 1-byte frame-count header, the root payload leads with the **universe**: the controller that owns spaces, layout, state, and the modules attached alongside it (`p.universe`, `p.input`, `p.keyboard`).

The keyboard is a live reflection of the system: which layout is active, which space holds focus. When a frame is focused, its script executes — it registers its own keys, reads its own state, and runs. When focus moves, those bindings are released. When focus returns, the script re-executes and resumes. Nothing is lost — state is preserved per space, per frame.

Only the client holds state, for the life of the session. The session is the application. The seams are invisible by design.

## documentation

[mechanics.md](mechanics.md) is the frame specification — the contract for building frames against known data, either through the built-in templates or a hand-authored `p.Frame`.
