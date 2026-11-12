package one

import (
	"net/http"

	"github.com/timefactoryio/pathless/fx"
)

type One struct {
	fx.Fx
	Pathless *http.ServeMux
	Circuit  *http.ServeMux
	frames   fx.N
	panels   fx.N
}

func NewOne() *One {
	o := &One{
		Fx:       fx.NewFx(),
		Pathless: http.NewServeMux(),
		Circuit:  http.NewServeMux(),
		frames:   fx.N{},
		panels:   fx.N{},
	}

	return o
}
