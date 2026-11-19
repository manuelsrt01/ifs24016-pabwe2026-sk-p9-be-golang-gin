package auth

import (
	"errors"
	"net/http"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register godoc
// @Summary Registrasi akun penulis baru
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Data Registrasi"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Router /api/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	res, err := h.service.Register(req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrUsernameTaken) || errors.Is(err, ErrEmailTaken) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal mendaftarkan pengguna")
		return
	}

	response.Success(c, http.StatusCreated, "Registrasi berhasil", res)
}

// Login godoc
// @Summary Autentikasi kredensial pengguna dan penerbitan token JWT
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Kredensial Login"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Router /api/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	res, err := h.service.Login(req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrInvalidCredentials) {
			response.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memproses login")
		return
	}

	response.Success(c, http.StatusOK, "Login berhasil", res)
}

// Logout godoc
// @Summary Logout sesi pengguna
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.APIResponse
// @Router /api/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

// GetProfile godoc
// @Summary Mengambil data profil penulis yang sedang login
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Router /api/auth/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.service.GetProfile(userID)
	if err != nil {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Profil berhasil diambil", user)
}

// UpdateProfile godoc
// @Summary Memperbarui informasi profil penulis yang sedang login
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body UpdateProfileRequest true "Data Profil"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Router /api/auth/profile [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	user, err := h.service.UpdateProfile(userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrUsernameTaken) || errors.Is(err, ErrEmailTaken) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memperbarui profil")
		return
	}

	response.Success(c, http.StatusOK, "Profil berhasil diperbarui", user)
}
