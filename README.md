# pathless

### is a closed system.

```go
func main() {
    p := pathless.NewPathless() // pathless.NewPathless("domain", "api.domain")
    // <- frames ->
    p.Serve()
}
```
```go
p := pathless.NewPathless()
// source bytes using p.Input() 
// serve bytes using p.Input() -> p.Route()
p.Serve() 
```
```go
p := pathless.NewPathless()
// p.Frame() accepts path to html file, local or https
// Templates via p.Home(), p.Text(), p.Slides()...
p.Serve() 
```
```go
package main

import "github.com/timefactoryio/pathless"

func main() {
    p := pathless.NewPathless()
    p.Home("./logo.svg", "the point of origin")
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
