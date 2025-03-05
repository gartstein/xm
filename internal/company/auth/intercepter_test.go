package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor(t *testing.T) {
	const (
		validSecret   = "test-secret"
		invalidSecret = "wrong-secret"
		userID        = "test-user"
	)

	// Helper to generate test tokens
	generateToken := func(secret string, expiresAt time.Time) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": userID,
			"exp": expiresAt.Unix(),
		})
		tokenString, _ := token.SignedString([]byte(secret))
		return tokenString
	}

	tests := []struct {
		name        string
		fullMethod  string
		token       string
		wantError   bool
		expectedErr codes.Code
	}{
		{
			name:        "protected method valid token",
			fullMethod:  "/definition.v1.CompanyService/CreateCompany",
			token:       generateToken(validSecret, time.Now().Add(1*time.Hour)),
			wantError:   false,
			expectedErr: codes.OK,
		},
		{
			name:        "protected method invalid token",
			fullMethod:  "/definition.v1.CompanyService/CreateCompany",
			token:       generateToken(invalidSecret, time.Now().Add(1*time.Hour)),
			wantError:   true,
			expectedErr: codes.Unauthenticated,
		},
		{
			name:        "protected method expired token",
			fullMethod:  "/definition.v1.CompanyService/CreateCompany",
			token:       generateToken(validSecret, time.Now().Add(-1*time.Hour)),
			wantError:   true,
			expectedErr: codes.Unauthenticated,
		},
		{
			name:        "protected method missing metadata",
			fullMethod:  "/definition.v1.CompanyService/CreateCompany",
			wantError:   true,
			expectedErr: codes.Unauthenticated,
		},
		{
			name:        "unprotected method no token",
			fullMethod:  "/definition.v1.CompanyService/GetCompany",
			wantError:   false,
			expectedErr: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewAuthInterceptor(validSecret)
			unaryInterceptor := interceptor.Unary()

			// Create context with metadata if token is provided
			ctx := context.Background()
			if tt.token != "" {
				md := metadata.Pairs("authorization", "Bearer "+tt.token)
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Mock handler that checks for claims in context
			handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
				if tt.fullMethod == "/definition.v1.CompanyService/CreateCompany" {
					claims, ok := ctx.Value(userContextKey).(jwt.MapClaims)
					if !ok || claims["sub"] != userID {
						return nil, status.Error(codes.Unauthenticated, "claims not in context")
					}
				}
				return "response", nil
			}

			info := &grpc.UnaryServerInfo{FullMethod: tt.fullMethod}
			resp, err := unaryInterceptor(ctx, nil, info, handler)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if status.Code(err) != tt.expectedErr {
					t.Errorf("expected error code %v, got %v", tt.expectedErr, status.Code(err))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp != "response" {
					t.Error("handler response mismatch")
				}
			}
		})
	}
}

func TestExtractTokenFromMetadata(t *testing.T) {
	tests := []struct {
		name        string
		metadata    metadata.MD
		wantToken   string
		wantErrCode codes.Code
	}{
		{
			name:        "valid authorization header",
			metadata:    metadata.Pairs("authorization", "Bearer valid-token"),
			wantToken:   "valid-token",
			wantErrCode: codes.OK,
		},
		{
			name:        "missing authorization header",
			metadata:    metadata.MD{},
			wantErrCode: codes.Unauthenticated,
		},
		{
			name:        "malformed authorization header",
			metadata:    metadata.Pairs("authorization", "InvalidPrefix valid-token"),
			wantErrCode: codes.Unauthenticated,
		},
		{
			name:        "empty bearer token",
			metadata:    metadata.Pairs("authorization", "Bearer "),
			wantErrCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractTokenFromMetadata(tt.metadata)

			if tt.wantErrCode != codes.OK {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if status.Code(err) != tt.wantErrCode {
					t.Errorf("expected error code %v, got %v", tt.wantErrCode, status.Code(err))
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if token != tt.wantToken {
				t.Errorf("expected token %q, got %q", tt.wantToken, token)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	const validSecret = "test-secret"
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	validTokenString, _ := validToken.SignedString([]byte(validSecret))

	tests := []struct {
		name        string
		tokenString string
		secret      string
		wantValid   bool
	}{
		{
			name:        "valid token",
			tokenString: validTokenString,
			secret:      validSecret,
			wantValid:   true,
		},
		{
			name:        "invalid signature",
			tokenString: validTokenString,
			secret:      "wrong-secret",
			wantValid:   false,
		},
		{
			name: "expired token",
			tokenString: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"exp": time.Now().Add(-1 * time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString([]byte(validSecret))
				return tokenString
			}(),
			secret:    validSecret,
			wantValid: false,
		},
		{
			name:        "malformed token",
			tokenString: "invalid.token.string",
			secret:      validSecret,
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validateToken(tt.tokenString, tt.secret)

			if tt.wantValid {
				if err != nil {
					t.Errorf("expected valid token, got error: %v", err)
				}
				if claims["sub"] != "user123" {
					t.Error("claims not properly parsed")
				}
			} else {
				if err == nil {
					t.Error("expected invalid token, got no error")
				}
			}
		})
	}
}

func TestNewAuthInterceptor(t *testing.T) {
	secret := "test-secret"
	interceptor := NewAuthInterceptor(secret)

	if interceptor.jwtSecret != secret {
		t.Errorf("expected secret %q, got %q", secret, interceptor.jwtSecret)
	}

	protectedMethods := []string{
		"/definition.v1.CompanyService/CreateCompany",
		"/definition.v1.CompanyService/UpdateCompany",
		"/definition.v1.CompanyService/DeleteCompany",
	}

	for _, method := range protectedMethods {
		if !interceptor.protectedMethods[method] {
			t.Errorf("missing protected method: %s", method)
		}
	}
}
