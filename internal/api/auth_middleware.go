package api

import (
	"log/slog"
	"net/http"

	"github.com/kenichiLyon/loong64-b1-go/internal/authn"
	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
)

func sessionRefreshMiddleware(service *authn.Service, logger *slog.Logger) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if service != nil {
				if err := service.RefreshSessionIfDue(r.Context(), r, w); err != nil && logger != nil {
					logger.Warn("session refresh failed", "error", err, "path", r.URL.Path)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
