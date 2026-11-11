package one

import "net/http"

type One struct {
	Pathless *http.ServeMux
	Circuit  *http.ServeMux
}

func NewOne(origin []byte, circuit map[string][]byte, pathlessURL string) *One {
	o := &One{
		Pathless: http.NewServeMux(),
		Circuit:  http.NewServeMux(),
	}
	o.origin(origin)
	o.circuit(circuit, pathlessURL)
	return o
}

func (o *One) Start() {
	go func() {
		if err := http.ListenAndServe(":1000", o.Pathless); err != nil {
			panic(err)
		}
	}()
	if err := http.ListenAndServe(":1001", o.Circuit); err != nil {
		panic(err)
	}
}
