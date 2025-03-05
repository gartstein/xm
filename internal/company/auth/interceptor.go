// Package auth provides a gRPC unary interceptor and JWT token validation
// to secure protected gRPC methods.
package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Interceptor holds the JWT secret and a map of protected methods.
type Interceptor struct {
	jwtSecret        string
	protectedMethods map[string]bool
}

type contextKey string

const (
	userContextKey contextKey = "user"
)

// NewAuthInterceptor creates a new Interceptor with the given secret and
// default protected methods.
func NewAuthInterceptor(jwtSecret string) *Interceptor {
	protected := map[string]bool{
		"/company.v1.CompanyService/CreateCompany": true,
		"/company.v1.CompanyService/UpdateCompany": true,
		"/company.v1.CompanyService/DeleteCompany": true,
	}

	return &Interceptor{
		jwtSecret:        jwtSecret,
		protectedMethods: protected,
	}
}

// Unary returns a gRPC unary interceptor for token validation on protected methods.
func (i *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if i.protectedMethods[info.FullMethod] {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, status.Error(codes.Unauthenticated, "metadata missing")
			}

			tokenString, err := extractTokenFromMetadata(md)
			if err != nil {
				return nil, err
			}

			claims, err := validateToken(tokenString, i.jwtSecret)
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
			}

			ctx = context.WithValue(ctx, userContextKey, claims)
		}

		return handler(ctx, req)
	}
}

// extractTokenFromMetadata retrieves a Bearer token from gRPC metadata.
func extractTokenFromMetadata(md metadata.MD) (string, error) {
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization header missing")
	}

	headerValue := authHeaders[0]
	if !strings.HasPrefix(headerValue, "Bearer ") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format: missing Bearer prefix")
	}

	tokenString := strings.TrimPrefix(headerValue, "Bearer ")
	if tokenString == "" {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format: empty token")
	}

	return tokenString, nil
}

// validateToken checks the token signature and returns parsed claims if valid.
func validateToken(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}
