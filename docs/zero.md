# zero

`zero` constructs closed system.


---

```go
type Zero struct {
    Pathless []byte // point of origin
    Universe []byte // closed system
}

func NewZero(circuit string) *Zero
```

`NewZero` executes `pathless.html` as a Go template with `{{.CIRCUIT}}` substituted (baking `circuit` in as `window.circuit`).

