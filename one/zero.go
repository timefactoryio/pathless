package one

import "net/http"

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
