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

---

![layouts](https://raw.githubusercontent.com/timefactoryio/pathless/main/content/layout.gif)