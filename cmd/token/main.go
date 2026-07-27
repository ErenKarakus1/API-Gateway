package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

func main() {
	secret := flag.String("secret", "change-me-in-production", "JWT signing secret")
	subject := flag.String("sub", "user-123", "JWT subject")
	roles := flag.String("roles", "user", "Comma-separated roles")
	ttl := flag.Duration("ttl", time.Hour, "Token lifetime")
	flag.Parse()

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Roles: splitRoles(*roles),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   *subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(*ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}).SignedString([]byte(*secret))
	if err != nil {
		log.Fatalf("sign token: %v", err)
	}

	fmt.Println(token)
}

func splitRoles(value string) []string {
	parts := strings.Split(value, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		role := strings.TrimSpace(part)
		if role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}
