package one

import "net/http"

func (o *One) circuit(payloads map[string][]byte, pathlessURL string) {
	for key, payload := range payloads {
		contentType := "application/octet-stream"
		if key == "" {
			contentType = "application/json"
		}
		o.Circuit.HandleFunc("/"+key, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", pathlessURL)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(payload)
		})
	}
}
