// This is a **mock authentication service**, designed to provide JWT tokens
// for the company service, simulating user authentication.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultPort   = "8081"       // Default port for the authentication service
	defaultSecret = "jwt_secret" // Secret for signing JWT
)

// TokenResponse represents the response structure
type TokenResponse struct {
	Token string `json:"token"`
}

// tokenHandler generates a JWT and returns it in JSON response
func tokenHandler(w http.ResponseWriter, _ *http.Request) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = defaultSecret
	}

	// Simulate a user ID for the token
	userID := "12345"

	token, err := generateToken(userID, secret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	resp := TokenResponse{Token: token}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "Failed to encode token", http.StatusInternalServerError)
	}
}

func main() {
	// TODO: move to env or config
	port := defaultPort
	http.HandleFunc("/token", tokenHandler)

	log.Printf("Authentication service running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func generateToken(userID string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,                                // Subject (User ID)
		"exp": time.Now().Add(time.Hour * 24).Unix(), // Expiration time
		"iat": time.Now().Unix(),                     // Issued at time
		"iss": "auth-service",                        // Issuer
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
