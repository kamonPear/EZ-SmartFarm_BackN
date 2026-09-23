package auth

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenTTL is how long an issued JWT stays valid (7 days, per the auth contract).
const TokenTTL = 168 * time.Hour

// devJWTSecret is used ONLY when the JWT_SECRET environment variable is not set.
// This exists so local development doesn't hard-fail without a .env file, but it is
// NOT safe for any real deployment - every real environment MUST set JWT_SECRET to
// its own random value (see .env / .env.example), otherwise anyone who reads this
// source file can forge valid tokens for any user.
const devJWTSecret = "dev-only-insecure-jwt-secret-do-not-use-in-production-3f8a2c91"

var (
	secretOnce sync.Once
	secretWarn sync.Once
)

func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	secretWarn.Do(func() {
		log.Println("=====================================================================")
		log.Println("WARNING: JWT_SECRET is not set in the environment. Falling back to a")
		log.Println("fixed, publicly-known development secret. This is INSECURE and must")
		log.Println("NEVER be used in a real deployment - set JWT_SECRET in .env / the")
		log.Println("environment before deploying.")
		log.Println("=====================================================================")
	})
	return []byte(devJWTSecret)
}

// Claims is the JWT payload issued for an authenticated user.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken issues a signed HS256 JWT for the given user, valid for TokenTTL.
func GenerateToken(userID int, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ParseToken validates tokenStr's signature and expiry and returns its claims.
func ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	return claims, nil
}

// UserID parses the claims' subject back into an int user id.
func (c *Claims) UserID() (int, error) {
	return strconv.Atoi(c.Subject)
}
