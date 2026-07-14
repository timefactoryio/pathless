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

One binary format for a sequence of values: the server encodes, the client decodes — there is no server-side decode. A small type table is written once, up front, followed by an entry count so the client can allocate its result up front instead of growing it; each entry then carries a 4-byte length and its bytes, plus a 1-byte type id unless every entry in the call shares one type — no names, no paths. Every route is served as one encoded value, its type carried in-band in the table, so the transport is opaque bytes — no `Content-Type` to trust, the client just decodes what it receives (a leaf, or a bundle whose body is one nested sequence typed `application/x-bundle`). Everything is gzip-compressed once at startup and served from memory. A route's `Value` can be `Save`d — gob-encoded to bytes, handed back for the caller to persist however it likes (e.g. object storage) — and re-sourced later via `ToValue` of that URL.

## client

The shell boots `window.pathless` — a thin wire client (fetch, decode, render). Every route resolves to exactly one `Value`; the root is a bundle whose children are the **universe** — the controller that owns spaces, layout, state, and the modules attached alongside it (`p.universe`, `p.input`, `p.keyboard`) — followed by the frame and panel pools, each a nested bundle a frame reads through the same `.children` as any directory route.

The keyboard is a live reflection of the system: which layout is active, which space holds focus. When a frame is focused, its script executes — it registers its own keys, reads its own state, and runs. When focus moves, those bindings are released. When focus returns, the script re-executes and resumes. Nothing is lost — state is preserved per space, per frame.

Only the client holds state, for the life of the session. The session is the application. The seams are invisible by design.

## documentation

[mechanics.md](mechanics.md) is the frame specification — the contract for building frames against known data, either through the built-in templates or a hand-authored `p.Frame`.
