package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/service"
)

type contextKey string

const ActorKey contextKey = "actor"

type RoleLoader interface {
	ListRoles(ctx context.Context, userID string) ([]domain.Role, error)
}

func Auth(secret string, roles RoleLoader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing authorization header")
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header")
				return
			}
			actor, err := service.ParseToken(secret, parts[1])
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			if roles != nil {
				fresh, err := roles.ListRoles(r.Context(), actor.UserID)
				if err != nil {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
					return
				}
				actor.Roles = fresh
			}
			ctx := context.WithValue(r.Context(), ActorKey, actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := ActorFromContext(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
				return
			}
			for _, role := range roles {
				if actor.HasRole(role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSONError(w, http.StatusForbidden, "forbidden", "forbidden")
		})
	}
}

func ActorFromContext(ctx context.Context) (domain.Actor, bool) {
	actor, ok := ctx.Value(ActorKey).(domain.Actor)
	return actor, ok
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		next.ServeHTTP(w, r)
	})
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
