package middleware

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		defer func() {
			log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(startTime))
		}()
		next.ServeHTTP(w, r)
	})
}
