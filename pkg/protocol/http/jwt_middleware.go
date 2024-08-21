package httpprotocol

import (
	"context"
	"net/http"
	"strings"

	pb_auth "github.com/purnasatria/library-management/api/gen/auth"
)

func JWTAuthMiddleware(authClient pb_auth.AuthServiceClient, exemptPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the path is exempt
			for _, exemptPath := range exemptPaths {
				if strings.HasPrefix(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
			if bearerToken == authHeader {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			// Call the auth service to verify the token
			resp, err := authClient.VerifyToken(r.Context(), &pb_auth.VerifyTokenRequest{Token: bearerToken})
			if err != nil {
				http.Error(w, "Failed to verify token", http.StatusInternalServerError)
				return
			}

			if !resp.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add user ID to the request context
			ctx := context.WithValue(r.Context(), "user_id", resp.UserId)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
