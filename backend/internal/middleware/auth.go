package middleware

import (
	"context"
	"net/http"
	"strings"

	jwtutil "github.com/go-kasir/backend/internal/pkg/jwt"
	"github.com/go-kasir/backend/internal/pkg/response"
)

type ctxKey string

const OwnerIDKey ctxKey = "owner_id"
const UsernameKey ctxKey = "username"

func Auth(jwtMgr *jwtutil.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				response.Unauthorized(w, "token tidak ditemukan")
				return
			}
			token := strings.TrimPrefix(h, "Bearer ")
			claims, err := jwtMgr.Parse(token)
			if err != nil || claims.TokenTyp != "access" {
				response.Unauthorized(w, "token tidak valid atau kadaluarsa")
				return
			}
			ctx := context.WithValue(r.Context(), OwnerIDKey, claims.OwnerID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetOwnerID(r *http.Request) uint {
	v, _ := r.Context().Value(OwnerIDKey).(uint)
	return v
}
