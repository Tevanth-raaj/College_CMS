package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"
)

// RecoveryMiddleware catches panics and logs them
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"Internal server error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	allowedOrigins := getAllowedOrigins()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if allowedOrigins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		} else {
			// Non-browser clients (curl/internal calls) typically do not send Origin.
			w.Header().Set("Access-Control-Allow-Origin", "https://academics.bitsathy.ac.in")
		}

		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getAllowedOrigins() map[string]bool {
	envOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	origins := []string{"https://academics.bitsathy.ac.in", "http://localhost:3000"}

	if envOrigins != "" {
		origins = strings.Split(envOrigins, ",")
	}

	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		normalized := strings.TrimSpace(origin)
		if normalized == "" {
			continue
		}
		allowed[normalized] = true
	}

	return allowed
}
