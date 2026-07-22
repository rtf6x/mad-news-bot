package handlers

import "net/http"

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if requested := r.Header.Get("Access-Control-Request-Headers"); requested != "" {
			w.Header().Set("Access-Control-Allow-Headers", requested)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
