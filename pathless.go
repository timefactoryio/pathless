package pathless

import (
	"fmt"
	"net/http"

	"github.com/timefactoryio/pathless/one"
)

type Pathless struct {
	*one.One
	pathlessURL string
}

func NewPathless(args ...string) *Pathless {
	pathlessURL := "*"
	circuitURL := "http://localhost:1001"
	switch len(args) {
	case 0:
	case 2:
		pathlessURL = "https://" + args[0]
		circuitURL = "https://" + args[1]
	default:
		panic(fmt.Sprintf(
			"pathless.NewPathless: expected 0 or 2 arguments, got %d",
			len(args),
		))
	}
	return &Pathless{
		One:         one.NewOne(circuitURL),
		pathlessURL: pathlessURL,
	}
}

func (p *Pathless) Start(circuitURL string) {
	go func() {
		if err := http.ListenAndServe(":1000", p.Pathless); err != nil {
			panic(err)
		}
	}()
	if err := http.ListenAndServe(":1001", p.Circuit); err != nil {
		panic(err)
	}

}
