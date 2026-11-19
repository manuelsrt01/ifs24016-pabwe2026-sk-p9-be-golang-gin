package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, http.StatusOK, "Operation successful", gin.H{"id": 1})

	assert.Equal(t, http.StatusOK, w.Code)
	var body APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "success", body.Status)
	assert.Equal(t, "Operation successful", body.Message)
	assert.NotNil(t, body.Data)
}

func TestFail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, http.StatusBadRequest, "Invalid input")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "fail", body.Status)
	assert.Equal(t, "Invalid input", body.Message)
	assert.Nil(t, body.Data)
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusInternalServerError, "Database error")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, "Database error", body.Message)
}

func TestAbortFail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	AbortFail(c, http.StatusUnauthorized, "Unauthorized")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted())
	var body APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "fail", body.Status)
	assert.Equal(t, "Unauthorized", body.Message)
}

func TestAbortError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	AbortError(c, http.StatusServiceUnavailable, "Service unavailable")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, c.IsAborted())
	var body APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, "Service unavailable", body.Message)
}
