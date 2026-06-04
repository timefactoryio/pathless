package pathless

import (
	"github.com/timefactoryio/pathless/one"
)

type Pathless struct {
	*one.One
}

func NewPathless(apiUrl string) *Pathless {
	return &Pathless{
		One: one.NewOne(apiUrl),
	}
}
