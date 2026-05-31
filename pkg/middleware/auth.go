package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"art-skill-wallet/pkg/response"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "artskilwallet-change-in-production-secret-key"
	}
	return []byte(s)
}

// ── JWTAuth ───────────────────────────────────────────────────────────────────

// JWTAuth validates the Bearer token from the Authorization header.
// On success it sets "user_id" and "role" in the Gin context.
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")

		// Must be exactly "Bearer <token>"
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, "Authorization header missing or malformed")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(
			tokenStr,
			func(t *jwt.Token) (interface{}, error) {
				// Strictly enforce HS256 — prevent algorithm downgrade / "alg:none" attacks
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return jwtSecret(), nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
		)

		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				response.Error(c, http.StatusUnauthorized, "Access token expired")
			} else {
				response.Error(c, http.StatusUnauthorized, "Invalid token")
			}
			c.Abort()
			return
		}

		if !token.Valid {
			response.Error(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "Invalid token claims")
			c.Abort()
			return
		}

		// Attach identity to context — downstream handlers read from here
		c.Set("user_id", claims["sub"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

// ── CheckAdmin ────────────────────────────────────────────────────────────────

// CheckAdmin is an authorization middleware that enforces the "admin" role.
// Must be used AFTER JWTAuth() in the middleware chain.
// Returns 403 Forbidden if the role is not exactly "admin".
func CheckAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")

		// Strict equality check — no type coercion, no fuzzy matching
		if !exists || role != "admin" {
			response.Error(c, http.StatusForbidden, "Forbidden: admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
