package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type failCreateRepo struct {
	Repository
}

func (f *failCreateRepo) Create(user *database.User) error {
	return errors.New("db create error")
}

type failUpdateRepo struct {
	Repository
}

func (f *failUpdateRepo) Update(user *database.User) error {
	return errors.New("db update error")
}

func TestAuthFeature_FullLifecycle(t *testing.T) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)

	repo := NewRepository(db)
	svc := NewService(repo, "test_secret")
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/logout", h.Logout)
	r.GET("/api/auth/profile", func(c *gin.Context) {
		if idStr := c.GetHeader("X-User-ID"); idStr == "999" {
			c.Set("user_id", uint(999))
		} else {
			c.Set("user_id", uint(3))
		}
		h.GetProfile(c)
	})
	r.PUT("/api/auth/profile", func(c *gin.Context) {
		if idStr := c.GetHeader("X-User-ID"); idStr == "999" {
			c.Set("user_id", uint(999))
		} else {
			c.Set("user_id", uint(3))
		}
		h.UpdateProfile(c)
	})

	// 1. Register invalid JSON
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString("{bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 2. Register missing fields
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"","email":"","password":""}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 3. Register success (becomes user ID 3)
	regBody, _ := json.Marshal(RegisterRequest{
		Username: "budi",
		Email:    "budi@delcom.org",
		Password: "password123",
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(regBody)))
	assert.Equal(t, http.StatusCreated, w.Code)

	// 4. Register duplicate username
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(regBody)))
	assert.Equal(t, http.StatusConflict, w.Code)

	// 5. Register duplicate email
	dupEmailBody, _ := json.Marshal(RegisterRequest{
		Username: "budi2",
		Email:    "budi@delcom.org",
		Password: "password123",
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(dupEmailBody)))
	assert.Equal(t, http.StatusConflict, w.Code)

	// 6. Login invalid JSON
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString("{bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 7. Login empty credentials
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"","password":""}`)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 8. Login non-existent user
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"ghost","password":"123"}`)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 9. Login wrong password
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"budi","password":"wrong"}`)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 10. Login success by username
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"budi","password":"password123"}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 11. Login success by email
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"budi@delcom.org","password":"password123"}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 12. Logout
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 13. GetProfile success & not found
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	reqNotFound := httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil)
	reqNotFound.Header.Set("X-User-ID", "999")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, reqNotFound)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 14. UpdateProfile invalid JSON
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString("{bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 15. UpdateProfile user not found
	reqUpdNotFound := httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"username":"x","email":"x@x.com"}`))
	reqUpdNotFound.Header.Set("X-User-ID", "999")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, reqUpdNotFound)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 16. UpdateProfile empty input
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"username":"","email":""}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 17. UpdateProfile conflict username & email (against seeded "admin")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"username":"admin","email":"budi@delcom.org"}`)))
	assert.Equal(t, http.StatusConflict, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"username":"budi","email":"admin@delcom.org"}`)))
	assert.Equal(t, http.StatusConflict, w.Code)

	// 18. UpdateProfile success with new username, email, and password
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"username":"budi_updated","email":"budi_new@delcom.org","password":"newpassword"}`)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthService_InternalErrors(t *testing.T) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)
	repo := NewRepository(db)

	// 1. bcryptHash error on Register and UpdateProfile
	origHash := bcryptHash
	bcryptHash = func(password string) ([]byte, error) {
		return nil, errors.New("bcrypt fail")
	}

	svc := NewService(repo, "sec")
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"u1","email":"u1@a.com","password":"123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Register(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("user_id", uint(1))
	c.Request = httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"username":"admin","email":"admin@delcom.org","password":"123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.UpdateProfile(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	bcryptHash = origHash

	// 2. Repo Create error on Register
	svcFailCreate := NewService(&failCreateRepo{Repository: repo}, "sec")
	_, err = svcFailCreate.Register(RegisterRequest{Username: "u2", Email: "u2@a.com", Password: "123"})
	assert.Error(t, err)

	// 3. Repo Update error on UpdateProfile
	svcFailUpdate := NewService(&failUpdateRepo{Repository: repo}, "sec")
	_, err = svcFailUpdate.UpdateProfile(1, UpdateProfileRequest{Username: "admin", Email: "admin@delcom.org"})
	assert.Error(t, err)

	// 4. signToken error on Login
	origSign := signToken
	signToken = func(token *jwt.Token, key []byte) (string, error) {
		return "", errors.New("sign error")
	}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	signToken = origSign
}
