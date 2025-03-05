package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func HTTPMiddleware(next http.Handler, jwtSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for non-protected endpoints
		if !isProtectedRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		tokenString, err := extractTokenFromHeader(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := validateToken(tokenString, jwtSecret)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), userContextKey, claims)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// Add these helper functions
func extractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("invalid authorization format")
	}

	return tokenString, nil
}

func isProtectedRequest(r *http.Request) bool {
	// Map your HTTP routes to protected gRPC methods
	protectedRoutes := map[string]string{
		"POST":   "/v1/companies",  // CreateCompany
		"PATCH":  "/v1/companies/", // UpdateCompany
		"DELETE": "/v1/companies/", // DeleteCompany
	}

	expectedPath := protectedRoutes[r.Method]
	return strings.HasPrefix(r.URL.Path, expectedPath)
}
