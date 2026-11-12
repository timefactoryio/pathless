package fx

import (
	"net/http"
	"time"
)

type Fx interface {
	Input(string) (N, error)
	InputPayload(string) (N, error)
	InputDir(string) (N, error)
	Load(string) (N, error)
	Encode(...N) []byte
	Decode([]byte) (N, error)
}

type fx struct {
	client *http.Client
}

func NewFx() Fx {
	return &fx{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}
