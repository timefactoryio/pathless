package pathless

import (
	"log"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/one"
	"github.com/timefactoryio/pathless/zero"
)

// Pathless is the top-level application, composed of its three layers:
//   - zero: compiles the HTML shell, assets, and templates
//   - fx:   encodes content into the wire format and manages routes
//   - one:  serves the HTML shell on :1000 and the wire gateway on :1001
type Pathless struct {
	*zero.Zero
	*fx.Fx
	*one.One
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
	case 2:
		origin = "https://" + args[0]
		circuit = "https://" + args[1]
	default:
		log.Fatalf("NewPathless: expected 0 or 2 arguments, got %d", len(args))
	}
	z := zero.NewZero(origin, circuit)
	x := fx.NewFx()
	return &Pathless{Zero: z, Fx: x, One: one.NewOne(z, x)}
}
