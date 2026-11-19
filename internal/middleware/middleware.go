package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func Auth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.AbortFail(c, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.AbortFail(c, http.StatusUnauthorized, "Invalid Authorization format")
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims := &UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			response.AbortFail(c, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		c.Set("user", claims)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("user")
		if !exists {
			response.AbortFail(c, http.StatusUnauthorized, "Authentication required")
			return
		}

		claims, ok := val.(*UserClaims)
		if !ok {
			response.AbortFail(c, http.StatusUnauthorized, "Invalid user context")
			return
		}

		for _, r := range roles {
			if strings.EqualFold(claims.Role, r) {
				c.Next()
				return
			}
		}

		response.AbortFail(c, http.StatusForbidden, "Forbidden: insufficient role permissions")
	}
}

type clientLimiter struct {
	count     int
	expiresAt time.Time
}

func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		clients = make(map[string]*clientLimiter)
	)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists || now.After(cl.expiresAt) {
			clients[ip] = &clientLimiter{
				count:     1,
				expiresAt: now.Add(window),
			}
			mu.Unlock()
			c.Next()
			return
		}

		if cl.count >= maxRequests {
			mu.Unlock()
			response.AbortFail(c, http.StatusTooManyRequests, "Too many requests, please try again later")
			return
		}

		cl.count++
		mu.Unlock()
		c.Next()
	}
}
