package post

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func parseUintParam(c *gin.Context, param string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return uint(id), true
}

func isAdminCtx(c *gin.Context) bool {
	return strings.EqualFold(c.GetString("role"), "admin")
}

// GetPublicCategories godoc
// @Summary Mengambil daftar seluruh kategori artikel blog
// @Tags Public
// @Produce json
// @Success 200 {object} response.APIResponse
// @Router /api/public/categories [get]
func (h *Handler) GetPublicCategories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Daftar kategori berhasil diambil", h.service.GetCategories())
}

// ListPublicPosts godoc
// @Summary Mengambil daftar seluruh artikel terbit dengan pencarian dan filter kategori
// @Tags Public
// @Produce json
// @Param q query string false "Kata kunci pencarian"
// @Param category query string false "Filter kategori"
// @Success 200 {object} response.APIResponse
// @Router /api/public/posts [get]
func (h *Handler) ListPublicPosts(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		q = c.Query("search")
	}
	category := c.Query("category")

	posts, err := h.service.ListPublicPosts(q, category)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar artikel publik")
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel publik berhasil diambil", posts)
}

// GetPublicPostBySlug godoc
// @Summary Mengambil detail artikel publik terbit berdasarkan slug
// @Tags Public
// @Produce json
// @Param slug path string true "Slug Artikel"
// @Success 200 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Router /api/public/posts/{slug} [get]
func (h *Handler) GetPublicPostBySlug(c *gin.Context) {
	slug := c.Param("slug")
	p, err := h.service.GetPublicPostBySlug(slug)
	if err != nil {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", p)
}

// ListAuthorPosts godoc
// @Summary Mengambil seluruh daftar artikel milik penulis yang sedang login
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.APIResponse
// @Router /api/posts [get]
func (h *Handler) ListAuthorPosts(c *gin.Context) {
	userID := c.GetUint("user_id")
	posts, err := h.service.ListAuthorPosts(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar artikel penulis")
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel penulis berhasil diambil", posts)
}

// CreatePost godoc
// @Summary Membuat artikel baru
// @Tags Posts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePostRequest true "Data Artikel Baru"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Router /api/posts [post]
func (h *Handler) CreatePost(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	p, err := h.service.CreatePost(userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidPostInput) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal menyimpan artikel")
		return
	}

	response.Success(c, http.StatusCreated, "Artikel berhasil dibuat", p)
}

// GetAuthorStats godoc
// @Summary Mengambil ringkasan 3 kartu statistik penulis
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.APIResponse
// @Router /api/posts/stats [get]
func (h *Handler) GetAuthorStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	stats, err := h.service.GetAuthorStats(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil statistik artikel")
		return
	}
	response.Success(c, http.StatusOK, "Statistik artikel berhasil diambil", stats)
}

// GetAuthorPostByID godoc
// @Summary Mengambil detail satu artikel milik penulis berdasarkan ID
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Artikel"
// @Success 200 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Router /api/posts/{id} [get]
func (h *Handler) GetAuthorPostByID(c *gin.Context) {
	postID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")

	p, err := h.service.GetAuthorPostByID(userID, postID, isAdminCtx(c))
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", p)
}

// UpdatePost godoc
// @Summary Memperbarui artikel milik penulis
// @Tags Posts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Artikel"
// @Param request body UpdatePostRequest true "Data Pembaruan Artikel"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Router /api/posts/{id} [put]
func (h *Handler) UpdatePost(c *gin.Context) {
	postID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	userID := c.GetUint("user_id")
	p, err := h.service.UpdatePost(userID, postID, isAdminCtx(c), req)
	if err != nil {
		if errors.Is(err, ErrInvalidPostInput) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memperbarui artikel")
		return
	}

	response.Success(c, http.StatusOK, "Artikel berhasil diperbarui", p)
}

// TogglePublish godoc
// @Summary Mengubah status publikasi artikel secara instan (Draf <-> Terbit)
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Artikel"
// @Success 200 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Router /api/posts/{id}/toggle-publish [patch]
func (h *Handler) TogglePublish(c *gin.Context) {
	postID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")

	p, err := h.service.TogglePublish(userID, postID, isAdminCtx(c))
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal mengubah status publikasi")
		return
	}

	response.Success(c, http.StatusOK, "Status publikasi artikel berhasil diubah", p)
}

// DeletePost godoc
// @Summary Menghapus artikel milik penulis
// @Tags Posts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Artikel"
// @Success 200 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Router /api/posts/{id} [delete]
func (h *Handler) DeletePost(c *gin.Context) {
	postID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")

	err := h.service.DeletePost(userID, postID, isAdminCtx(c))
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal menghapus artikel")
		return
	}

	response.Success(c, http.StatusOK, "Artikel berhasil dihapus", nil)
}

// AdminListUsers godoc
// @Summary Mengambil data seluruh pengguna sistem beserta total artikel yang ditulis
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.APIResponse
// @Router /api/admin/users [get]
func (h *Handler) AdminListUsers(c *gin.Context) {
	users, err := h.service.AdminListUsers()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar pengguna")
		return
	}
	response.Success(c, http.StatusOK, "Daftar seluruh pengguna berhasil diambil", users)
}

// AdminListPosts godoc
// @Summary Monitoring arsip seluruh artikel dari semua penulis dalam sistem
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.APIResponse
// @Router /api/admin/posts [get]
func (h *Handler) AdminListPosts(c *gin.Context) {
	posts, err := h.service.AdminListAllPosts()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil arsip seluruh artikel")
		return
	}
	response.Success(c, http.StatusOK, "Arsip seluruh artikel berhasil diambil", posts)
}

// AdminListUserPosts godoc
// @Summary Mengambil seluruh artikel yang ditulis oleh pengguna tertentu
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pengguna"
// @Success 200 {object} response.APIResponse
// @Router /api/admin/users/{id}/posts [get]
func (h *Handler) AdminListUserPosts(c *gin.Context) {
	targetUserID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	posts, err := h.service.AdminListUserPosts(targetUserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil artikel pengguna")
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel pengguna berhasil diambil", posts)
}
