package one

import "net/http"

func (o *One) HandlePathless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.Zero.One)
}

func (o *One) Serve() {
	http.ListenAndServe(":1000", o.Pathless)
}
