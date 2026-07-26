# pathless

### is a closed system.

```go
p := pathless.NewPathless()
// <- frames ->
// p.Frame() accepts local path to html file or https
// Templates via p.Text(), p.Slides()...
// <- frames ->
p.Serve()
```
```go
p := pathless.NewPathless()
// source bytes using p.Input() 
// serve bytes using p.Input() -> p.Route()
p.Serve() 
```
```go
p := pathless.NewPathless() // pathless.NewPathless("domain", "api.domain")
p.Serve() 
```
```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless()
    p.Text("./readme.md")
    p.Slides("./pics")
    p.Frame("./custom.html")
    p.Serve() // execute everything after p := pathless.NewPathless()
}
```

## primitives

#### space: where we observe objects.
#### frame: observable object.
#### frames: simultaneously observable objects.


## sequence

Three layers, one direction:

- **zero** — compiles the HTML shell and the universe payload from embedded sources
- **fx** — sources content into frames, panels, and routes; no wire format, no HTTP
- **one** — encodes the wire format, then serves the shell on `:1000` and the wire gateway on `:1001`
