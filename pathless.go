package pathless

import (
	"github.com/timefactoryio/pathless/one"
)

type Pathless struct {
	one *one.One
}

func NewPathless(apiUrl string) *Pathless {
	return &Pathless{
		one: one.NewOne(apiUrl),
	}
}
