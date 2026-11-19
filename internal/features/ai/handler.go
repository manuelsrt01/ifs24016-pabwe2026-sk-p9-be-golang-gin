package ai

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

// GenerateBlog godoc
// @Summary Menghasilkan draf artikel utuh menggunakan AI dan Search Engine
// @Tags AI
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body GenerateBlogRequest true "Parameter Topik AI Generator"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 502 {object} response.APIResponse
// @Router /api/posts/ai-generate [post]
func (h *Handler) GenerateBlog(c *gin.Context) {
	var req GenerateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	res, err := h.service.GenerateBlog(req)
	if err != nil {
		if errors.Is(err, ErrTopicRequired) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusBadGateway, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Draf artikel AI berhasil dibuat", res)
}
