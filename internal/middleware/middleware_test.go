package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func generateTestToken(secret string, userID uint, username, role string) string {
	claims := UserClaims{
		UserID:   userID,
		Username: username,
		Email:    username + "@delcom.org",
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// Test normal GET
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Test OPTIONS preflight
	reqOpt := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	wOpt := httptest.NewRecorder()
	r.ServeHTTP(wOpt, reqOpt)
	assert.Equal(t, http.StatusNoContent, wOpt.Code)
}

func TestAuth(t *testing.T) {
	secret := "test_secret"
	r := gin.New()
	r.Use(Auth(secret))
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 1. Missing header
	req1 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// 2. Invalid format
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "InvalidFormat")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// 3. Invalid token string
	req3 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req3.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusUnauthorized, w3.Code)

	// 4. Unexpected signing method (RSA instead of HMAC)
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rsaToken := jwt.NewWithClaims(jwt.SigningMethodRS256, UserClaims{UserID: 1})
	rsaSigned, _ := rsaToken.SignedString(privKey)
	req4 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req4.Header.Set("Authorization", "Bearer "+rsaSigned)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusUnauthorized, w4.Code)

	// 5. Valid token
	validToken := generateTestToken(secret, 1, "member", "member")
	req5 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req5.Header.Set("Authorization", "Bearer "+validToken)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusOK, w5.Code)
}

func TestRequireRole(t *testing.T) {
	// 1. Missing user in context
	r1 := gin.New()
	r1.Use(RequireRole("admin"))
	r1.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })
	w1 := httptest.NewRecorder()
	r1.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/admin", nil))
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// 2. Invalid user type in context
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		c.Set("user", "not-a-claims-struct")
		c.Next()
	}, RequireRole("admin"))
	r2.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/admin", nil))
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// 3. Forbidden role
	r3 := gin.New()
	r3.Use(func(c *gin.Context) {
		c.Set("user", &UserClaims{UserID: 2, Role: "member"})
		c.Next()
	}, RequireRole("admin"))
	r3.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })
	w3 := httptest.NewRecorder()
	r3.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/admin", nil))
	assert.Equal(t, http.StatusForbidden, w3.Code)

	// 4. Allowed role
	r4 := gin.New()
	r4.Use(func(c *gin.Context) {
		c.Set("user", &UserClaims{UserID: 1, Role: "admin"})
		c.Next()
	}, RequireRole("admin"))
	r4.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })
	w4 := httptest.NewRecorder()
	r4.ServeHTTP(w4, httptest.NewRequest(http.MethodGet, "/admin", nil))
	assert.Equal(t, http.StatusOK, w4.Code)
}

func TestRateLimiter(t *testing.T) {
	r := gin.New()
	r.Use(RateLimiter(2, 50*time.Millisecond))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	// First request -> 200
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/test", nil))
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request -> 200
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/test", nil))
	assert.Equal(t, http.StatusOK, w2.Code)

	// Third request -> 429 Too Many Requests
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/test", nil))
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)

	// Wait for window expiry -> 200
	time.Sleep(60 * time.Millisecond)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, httptest.NewRequest(http.MethodGet, "/test", nil))
	assert.Equal(t, http.StatusOK, w4.Code)
}
