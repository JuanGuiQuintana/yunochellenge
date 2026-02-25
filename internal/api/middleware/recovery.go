package middleware

import (
	"log/slog"
	"net/http"

	"github.com/juanatsap/chargeback-api/internal/api/response"
)

// Recovery catches panics and returns 500 instead of crashing the server.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err)
				response.WriteError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
