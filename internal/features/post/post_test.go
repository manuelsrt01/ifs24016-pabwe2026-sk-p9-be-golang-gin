package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockErrorRepo struct {
	Repository
	failFindPublished bool
	failFindByUser    bool
	failCreate        bool
	failUpdate        bool
	failDelete        bool
	failSlugExists    bool
	failCountStats    bool
	failFindAllUsers  bool
	failFindAllPosts  bool
}

func (m *mockErrorRepo) FindPublished(query, category string) ([]database.Post, error) {
	if m.failFindPublished {
		return nil, errors.New("find published err")
	}
	return m.Repository.FindPublished(query, category)
}

func (m *mockErrorRepo) FindByUser(userID uint) ([]database.Post, error) {
	if m.failFindByUser {
		return nil, errors.New("find by user err")
	}
	return m.Repository.FindByUser(userID)
}

func (m *mockErrorRepo) Create(post *database.Post) error {
	if m.failCreate {
		return errors.New("create err")
	}
	return m.Repository.Create(post)
}

func (m *mockErrorRepo) Update(post *database.Post) error {
	if m.failUpdate {
		return errors.New("update err")
	}
	return m.Repository.Update(post)
}

func (m *mockErrorRepo) Delete(post *database.Post) error {
	if m.failDelete {
		return errors.New("delete err")
	}
	return m.Repository.Delete(post)
}

func (m *mockErrorRepo) SlugExists(slug string, excludeID uint) (bool, error) {
	if m.failSlugExists {
		return false, errors.New("slug exists err")
	}
	return m.Repository.SlugExists(slug, excludeID)
}

func (m *mockErrorRepo) CountStatsByUser(userID uint) (int64, int64, int64, error) {
	if m.failCountStats {
		return 0, 0, 0, errors.New("stats err")
	}
	return m.Repository.CountStatsByUser(userID)
}

func (m *mockErrorRepo) FindAllUsers() ([]database.User, error) {
	if m.failFindAllUsers {
		return nil, errors.New("find all users err")
	}
	return m.Repository.FindAllUsers()
}

func (m *mockErrorRepo) FindAllPosts() ([]database.Post, error) {
	if m.failFindAllPosts {
		return nil, errors.New("find all posts err")
	}
	return m.Repository.FindAllPosts()
}

func TestHelpers(t *testing.T) {
	assert.Equal(t, 1, CalculateReadingTime(""))
	longText := strings.Repeat("kata ", 450)
	assert.Equal(t, 3, CalculateReadingTime(longText))

	assert.Equal(t, "artikel", Slugify("   !!!   "))
	assert.Equal(t, "belajar-go-gin", Slugify("Belajar Go & Gin!"))

	assert.Equal(t, "Teknologi", NormalizeCategory("teknologi"))
	assert.Equal(t, "Umum", NormalizeCategory("KategoriLain"))

	assert.Equal(t, "Custom Excerpt", BuildExcerpt("Custom Excerpt", "Content"))
	longContent := strings.Repeat("abcdefghij ", 20)
	exc := BuildExcerpt("", longContent)
	assert.True(t, strings.HasSuffix(exc, "..."))
	assert.Equal(t, "Short", BuildExcerpt("", "Short"))
}

func TestPostFeature_FullLifecycle(t *testing.T) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		uid := uint(2) // default member
		if c.GetHeader("X-User-ID") == "1" {
			uid = 1
		}
		role := "member"
		if c.GetHeader("X-Role") == "admin" {
			role = "admin"
		}
		c.Set("user_id", uid)
		c.Set("role", role)
		c.Next()
	})

	r.GET("/api/public/categories", h.GetPublicCategories)
	r.GET("/api/public/posts", h.ListPublicPosts)
	r.GET("/api/public/posts/:slug", h.GetPublicPostBySlug)
	r.GET("/api/posts", h.ListAuthorPosts)
	r.POST("/api/posts", h.CreatePost)
	r.GET("/api/posts/stats", h.GetAuthorStats)
	r.GET("/api/posts/:id", h.GetAuthorPostByID)
	r.PUT("/api/posts/:id", h.UpdatePost)
	r.PATCH("/api/posts/:id/toggle-publish", h.TogglePublish)
	r.DELETE("/api/posts/:id", h.DeletePost)
	r.GET("/api/admin/users", h.AdminListUsers)
	r.GET("/api/admin/posts", h.AdminListPosts)
	r.GET("/api/admin/users/:id/posts", h.AdminListUserPosts)

	// 1. GetCategories
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/categories", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. CreatePost invalid JSON & empty fields
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString("{bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"","content":""}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 3. CreatePost published (ID 1)
	b1, _ := json.Marshal(CreatePostRequest{
		Title:          "Belajar Golang Gin",
		Category:       "Teknologi",
		Excerpt:        "Ringkasan Go",
		Content:        "Konten lengkap Go Gin.",
		CoverImagePath: "https://example.com/cover.png",
		IsPublished:    true,
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBuffer(b1)))
	assert.Equal(t, http.StatusCreated, w.Code)

	// 4. CreatePost duplicate title -> generates unique slug "belajar-golang-gin-2" (ID 2, draft)
	b2, _ := json.Marshal(CreatePostRequest{
		Title:       "Belajar Golang Gin",
		Category:    "Pendidikan",
		Content:     "Konten kedua.",
		IsPublished: false,
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBuffer(b2)))
	assert.Equal(t, http.StatusCreated, w.Code)

	// 5. Public posts listing (all, search q, search param, category filter)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts?q=Golang&category=Teknologi", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts?search=Golang&category=Semua", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. GetPublicPostBySlug success & not found
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts/belajar-golang-gin", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts/tidak-ada", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 7. ListAuthorPosts & GetAuthorStats
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/stats", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 8. GetAuthorPostByID: invalid ID, not found, forbidden, success
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/999", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	reqOtherUser := httptest.NewRequest(http.MethodGet, "/api/posts/1", nil)
	reqOtherUser.Header.Set("X-User-ID", "1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, reqOtherUser)
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 9. UpdatePost: invalid ID, invalid JSON, not found, forbidden, empty input, success
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBufferString("{bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/999", bytes.NewBufferString(`{"title":"T","content":"C"}`)))
	assert.Equal(t, http.StatusNotFound, w.Code)

	reqUpdForbidden := httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBufferString(`{"title":"T","content":"C"}`))
	reqUpdForbidden.Header.Set("X-User-ID", "1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, reqUpdForbidden)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// Empty fields on UpdatePost keep existing values (200 OK)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBufferString(`{"title":"","content":""}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	fFalse := false
	updReq1, _ := json.Marshal(UpdatePostRequest{
		Title:          "Judul Baru Diperbarui",
		Category:       "Bisnis",
		Excerpt:        "Ringkasan baru",
		Content:        "Isi diperbarui",
		CoverImagePath: "https://example.com/new.png",
		IsPublished:    &fFalse,
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBuffer(updReq1)))
	assert.Equal(t, http.StatusOK, w.Code)

	fTrue := true
	updReq2, _ := json.Marshal(UpdatePostRequest{
		Title:       "Judul Baru Diperbarui",
		Category:    "Bisnis",
		Content:     "Isi diperbarui lagi",
		IsPublished: &fTrue,
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBuffer(updReq2)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 10. TogglePublish: invalid ID, not found, forbidden, success (published -> draft -> published)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/abc/toggle-publish", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/999/toggle-publish", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	reqTogForbidden := httptest.NewRequest(http.MethodPatch, "/api/posts/1/toggle-publish", nil)
	reqTogForbidden.Header.Set("X-User-ID", "1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, reqTogForbidden)
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/1/toggle-publish", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/1/toggle-publish", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 11. Admin endpoints
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/posts", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users/abc/posts", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users/2/posts", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 12. DeletePost: invalid ID, not found, forbidden, success
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/posts/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/posts/999", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	reqDelForbidden := httptest.NewRequest(http.MethodDelete, "/api/posts/1", nil)
	reqDelForbidden.Header.Set("X-User-ID", "1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, reqDelForbidden)
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/posts/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostRepositoryAndService_ErrorBranches(t *testing.T) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)
	baseRepo := NewRepository(db)

	// Create 1 post for update/toggle/delete error tests
	p := &database.Post{
		UserID:  2,
		Title:   "Initial",
		Slug:    "initial",
		Content: "Initial content",
	}
	assert.NoError(t, baseRepo.Create(p))

	mRepo := &mockErrorRepo{Repository: baseRepo}
	svc := NewService(mRepo)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("role", "admin")
		c.Next()
	})
	r.GET("/api/public/posts", h.ListPublicPosts)
	r.GET("/api/posts", h.ListAuthorPosts)
	r.POST("/api/posts", h.CreatePost)
	r.GET("/api/posts/stats", h.GetAuthorStats)
	r.PUT("/api/posts/:id", h.UpdatePost)
	r.PATCH("/api/posts/:id/toggle-publish", h.TogglePublish)
	r.DELETE("/api/posts/:id", h.DeletePost)
	r.GET("/api/admin/users", h.AdminListUsers)
	r.GET("/api/admin/posts", h.AdminListPosts)
	r.GET("/api/admin/users/:id/posts", h.AdminListUserPosts)

	// 1. FindPublished error
	mRepo.failFindPublished = true
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failFindPublished = false

	// 2. FindByUser error (ListAuthorPosts & AdminListUserPosts)
	mRepo.failFindByUser = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users/2/posts", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failFindByUser = false

	// 3. SlugExists error on CreatePost & UpdatePost
	mRepo.failSlugExists = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"New","content":"Body"}`)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBufferString(`{"title":"Changed","content":"Body"}`)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failSlugExists = false

	// 4. Create error on CreatePost
	mRepo.failCreate = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"New","content":"Body"}`)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failCreate = false

	// 5. Update error on UpdatePost & TogglePublish
	mRepo.failUpdate = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBufferString(`{"title":"Initial","content":"Body"}`)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/1/toggle-publish", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failUpdate = false

	// 6. Delete error on DeletePost
	mRepo.failDelete = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/posts/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failDelete = false

	// 7. CountStats error
	mRepo.failCountStats = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/stats", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failCountStats = false

	// 8. FindAllUsers error
	mRepo.failFindAllUsers = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failFindAllUsers = false

	// 9. FindAllPosts error
	mRepo.failFindAllPosts = true
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/posts", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mRepo.failFindAllPosts = false

	// 10. Test underlying GORM repository errors when DB is closed
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	_, err = baseRepo.FindPublished("", "")
	assert.Error(t, err)
	_, err = baseRepo.FindByUser(1)
	assert.Error(t, err)
	_, err = baseRepo.SlugExists("slug", 1)
	assert.Error(t, err)
	_, _, _, err = baseRepo.CountStatsByUser(1)
	assert.Error(t, err)
	_, err = baseRepo.FindAllUsers()
	assert.Error(t, err)
	_, err = baseRepo.FindAllPosts()
	assert.Error(t, err)

	// Test second query error in CountStatsByUser
	db2, _ := database.ConnectTestDB()
	repo2 := NewRepository(db2)
	queryCount := 0
	_ = db2.Callback().Query().Before("gorm:query").Register("fail_second_query", func(d *gorm.DB) {
		queryCount++
		if queryCount == 2 {
			d.AddError(errors.New("second query failed"))
		}
	})
	_, _, _, err = repo2.CountStatsByUser(1)
	assert.Error(t, err)
}
