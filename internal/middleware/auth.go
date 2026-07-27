package middleware

import (
	"net/http"
	"strings"

	"github.com/ErenKarakus1/API-Gateway/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const UserIDKey = "user_id"
const RolesKey = "roles"

type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			response.Error(c, http.StatusUnauthorized, "missing_bearer_token", "missing bearer token")
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, "invalid_bearer_token", "invalid bearer token")
			return
		}

		if subject := claims.Subject; subject != "" {
			c.Set(UserIDKey, subject)
			c.Request.Header.Set("X-User-ID", subject)
		}
		c.Set(RolesKey, claims.Roles)

		c.Next()
	}
}

func RequireRoles(requiredRoles []string) gin.HandlerFunc {
	required := make(map[string]struct{}, len(requiredRoles))
	for _, role := range requiredRoles {
		required[role] = struct{}{}
	}

	return func(c *gin.Context) {
		roles, ok := c.Get(RolesKey)
		if !ok {
			response.Error(c, http.StatusForbidden, "missing_roles", "missing roles")
			return
		}

		for _, role := range roles.([]string) {
			if _, ok := required[role]; ok {
				c.Next()
				return
			}
		}

		response.Error(c, http.StatusForbidden, "insufficient_role", "insufficient role")
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
