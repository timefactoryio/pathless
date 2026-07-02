# pathless

### is a closed system.

**pathless** prepares one artifact to be used from memory for every request. It has no runtime state, no session awareness, no opinion about content. It is a boundary. It knows how to hold applications without knowing what those applications are. The server's responsibility ends at delivery.

A request arrives, bytes go out. That is the entirety of its runtime existence. The client arrives as those bytes.

## sequence

Three layers, one direction:

- **zero** — compiles the HTML shell and the universe payload from embedded sources
- **fx** — encodes content into the wire format, builds frames, manages routes
- **one** — serves the shell on `:1000` and the wire gateway on `:1001`

```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless() // dev; NewPathless("domain", "api.domain") for prod
    p.Home("./logo.svg", "the point of origin")
    p.Text("./readme.md")
    p.Slides("./pics")
    p.CustomHTML("./custom.html")
    p.Serve()
}
```

## primitives

#### space: where we observe frames.

Three spaces — `zero`, `fx`, `one` — compose into layouts (single, split, quad) driven by CSS. Each space maintains independent state for each frame.

#### frame: observable object.

Frames are a finite pool of simultaneously observable content: single `.html` files (style, shell, script) delivered over the wire and executed in a space. The domain is the application.

#### wire: order is the contract.

One binary format, both directions. The server encodes; the client decodes. Entries carry a type and bytes — no names, no paths. `[2B count]` then per entry `[1B typeLen][type][4B dataLen][data]`. Routes are encoded and gzip-compressed once at startup and served from memory. A route can be `Save`d as its wire encoding, synced to object storage, and `Load`ed back by URL.

## client

The shell boots `window.pathless` — a thin wire client (fetch, decode, render). Item 0 of the root payload is the **universe**: the controller that owns spaces, layout, state, and the modules attached alongside it (`p.universe`, `p.coordinates`, `p.input`, `p.keyboard`).

The keyboard is a live reflection of the system: which layout is active, which space holds focus. When a frame is focused, its script executes — it registers its own keys, reads its own state, and runs. When focus moves, those bindings are released. When focus returns, the script re-executes and resumes. Nothing is lost — state is preserved per space, per frame.

Only the client holds state, for the life of the session. The session is the application. The seams are invisible by design.

## documentation

[mechanics.md](mechanics.md) is the frame specification — the contract for building frames against known data, either through templates or `CustomHTML`.


# pathless

### is a closed system. 

## sequence

pathless is an entity with a single purpose.  

**pathless** prepares one artifact to be used from memory for every request. It has no runtime state, no session awareness, no opinion about content. It is a boundary. It knows how to hold applications without knowing what those applications are. The server's responsibility ends at delivery. 

### **pathless** allocates observable space.

`frame`'s are a finite pool of simulataneously observable content. 

The domain is the application.

A request arrives, bytes go out. That is the entirety of its runtime existence. The client arrives as those bytes. Two primitives govern the model.

#### **space**: where we observe frames.

Each space maintains independent state for each frame. 

#### **frame**: observable object. 

**[frame](https://github.com/timefactoryio/frame)** is the factory for building pathless frames. It handles the construction and delivery of frames, assets, and constants. The dependency flows one way.

The keyboard is a live reflection of the systems keys, which layout is active, which panel holds focus. When a frame is focused, its script executes. The frame registers its own keys, reads its own state, and runs. When focus moves, those keybindings are released. When focus returns, the script re-executes and resumes. Nothing is lost — state is preserved per space, per frame.

Inputs are the interface.
The keyboard is the thread that runs through all three.
Universal keys + frame keys that come and go with focus, state is what makes the system feel continuous. When focus moves, state is preserved. When a frame returns, it resumes. Only the client holds state for the life of the session. The session is the application. The seams are invisible by design.

The dependency flows are pre-established and then silent. What the user encounters is a single continuous surface: a domain, a keyboard, panels that respond.

## Documentation

The `window.pathless` object provides the API coordinating between `space`, `frame`, and `state`.    
