package one

import (
	"net/http"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/zero"
)

type One struct {
	*zero.Zero
	fx.Fx
	Pathless *http.ServeMux
	Circuit  *http.ServeMux
	frames   fx.N
	panels   fx.N
}

func NewOne(circuitURL string) *One {
	o := &One{
		Zero:     zero.NewZero(circuitURL),
		Fx:       fx.NewFx(),
		Pathless: http.NewServeMux(),
		Circuit:  http.NewServeMux(),
		frames:   fx.N{},
		panels:   fx.N{},
	}

	return o
}

func (o *One) origin(payload []byte) {
	o.Pathless.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.URL.RawQuery != "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(payload)
	})
}
