package pathless

import (
	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/one"
	"github.com/timefactoryio/pathless/zero"
)

// Pathless is the top-level application, composed of its three layers:
//   - zero: compiles the HTML shell, assets, and templates
//   - fx:   encodes content into the wire format and manages routes
//   - one:  serves the HTML shell on :1000 and the wire gateway on :1001
type Pathless struct {
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
	z := zero.NewZero(args...)
	x := fx.NewFx(z)
	return &Pathless{One: one.NewOne(x)}
}
