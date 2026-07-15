package pathless

import (
	"log"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/one"
	"github.com/timefactoryio/pathless/zero"
)

// Pathless is the top-level application, composing its three layers by
// embedding the two that carry the authoring API:
//   - zero: compiles the HTML shell and universe payload
//   - fx:   sources content into Values, builds frames/panels/routes
//   - one:  encodes the wire format and serves it (constructed at Serve time)
//
// Embedding *zero.Zero and *fx.Fx promotes their methods (Home, Frame, Text,
// Slides, Logo, Keyboard, Input, …) onto Pathless, so a program calls them
// directly on the value returned by NewPathless.
type Pathless struct {
	*zero.Zero
	*fx.Fx
	origin string
}

// NewPathless constructs a Pathless application.
//
// Development — call with no arguments:
//
//	p := pathless.NewPathless()
//
// Ports 1000 and 1001 are used on localhost. CORS is open (*).
//
// Production — call with origin and circuit:
//
//	p := pathless.NewPathless("timefactory.io", "api.timefactory.io")
//
// The HTML shell is served from origin; the wire gateway is served from circuit.
// CORS on the gateway is restricted to origin. HTTPS is assumed.
func NewPathless(args ...string) *Pathless {
	var origin, circuit string
	switch len(args) {
	case 0:
		origin = "*"
		circuit = "http://localhost:1001"
	case 2:
		origin = "https://" + args[0]
		circuit = "https://" + args[1]
	default:
		log.Fatalf("NewPathless: expected 0 or 2 arguments, got %d", len(args))
	}

	return &Pathless{
		Zero:   zero.NewZero(circuit),
		Fx:     fx.NewFx(),
		origin: origin,
	}
}

// Serve assembles the wire payload from zero's assets and fx's pools, then
// starts the two listeners (shell on :1000, wire gateway on :1001).
func (p *Pathless) Serve() {
	one.NewOne(p.origin, p.Pathless, p.Universe, p.Fx).Serve()
}
