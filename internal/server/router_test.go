package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/config"
	"app/internal/database"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewRouter_AllRoutes(t *testing.T) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)

	cfg := &config.Config{
		AppKey:               "test_secret_key",
		SearchEngineEndpoint: "http://localhost:9999/search",
		OpenAIEndpoint:       "http://localhost:9999",
		OpenAIKey:            "key",
		OpenAIModel:          "model",
	}

	r := NewRouter(db, cfg)

	// 1. Test /api.json (HTTP and HTTPS via X-Forwarded-Proto)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api.json", nil)
	req.Host = "localhost:8081"
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"swagger": "2.0"`)

	wHttps := httptest.NewRecorder()
	reqHttps := httptest.NewRequest(http.MethodGet, "/api.json", nil)
	reqHttps.Header.Set("X-Forwarded-Proto", "https")
	r.ServeHTTP(wHttps, reqHttps)
	assert.Equal(t, http.StatusOK, wHttps.Code)
	assert.Contains(t, wHttps.Body.String(), `"https"`)

	// 2. Test /swagger/index.html
	wSwag := httptest.NewRecorder()
	r.ServeHTTP(wSwag, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(t, http.StatusOK, wSwag.Code)

	// 3. Test Public routes
	wPubCat := httptest.NewRecorder()
	r.ServeHTTP(wPubCat, httptest.NewRequest(http.MethodGet, "/api/public/categories", nil))
	assert.Equal(t, http.StatusOK, wPubCat.Code)

	wPubPosts := httptest.NewRecorder()
	r.ServeHTTP(wPubPosts, httptest.NewRequest(http.MethodGet, "/api/public/posts", nil))
	assert.Equal(t, http.StatusOK, wPubPosts.Code)

	// 4. Test Login as Admin to get JWT token
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`)))
	assert.Equal(t, http.StatusOK, wLogin.Code)

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token
	assert.NotEmpty(t, token)

	// 5. Test Protected Post creation & Admin route
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"Router Test","category":"Teknologi","content":"Hello world","is_published":true}`))
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, reqCreate)
	assert.Equal(t, http.StatusCreated, wCreate.Code)

	reqAdmin := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+token)
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)
	assert.Equal(t, http.StatusOK, wAdmin.Code)

	// 6. Test 404 NoRoute
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, httptest.NewRequest(http.MethodGet, "/api/unknown-endpoint", nil))
	assert.Equal(t, http.StatusNotFound, w404.Code)
	var errResp response.APIResponse
	_ = json.Unmarshal(w404.Body.Bytes(), &errResp)
	assert.Equal(t, "fail", errResp.Status)
	assert.Equal(t, "Endpoint not found", errResp.Message)
}
