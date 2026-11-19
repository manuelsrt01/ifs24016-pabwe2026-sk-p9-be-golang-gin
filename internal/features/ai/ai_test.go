package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read body error")
}

type ioCloser struct {
	errReader
}

func (ioCloser) Close() error { return nil }

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func makeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestSearchClient(t *testing.T) {
	// 1. Empty query or endpoint
	cEmpty := NewSearchClient("", "")
	_, err := cEmpty.Search("   ")
	assert.Error(t, err)
	_, err = cEmpty.Search("golang")
	assert.Error(t, err)

	// 2. Invalid URL format
	cBadURL := NewSearchClient("://bad-url", "key")
	_, err = cBadURL.Search("golang")
	assert.Error(t, err)

	// 3. Test various responses via in-memory RoundTripper
	mode := "results"
	client := NewSearchClient("https://search.delcom.org/api", "secret-token").(*httpSearchClient)
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch mode {
		case "network_err":
			return nil, errors.New("network unreachable")
		case "results":
			return makeResponse(http.StatusOK, `{"results":[{"title":"Go 1.27","url":"https://go.dev","snippet":"Release notes"},{"title":"","url":"","snippet":""}]}`), nil
		case "sources":
			return makeResponse(http.StatusOK, `{"sources":[{"title":"Src","url":"https://src.dev","snippet":"Info"}]}`), nil
		case "data_list":
			return makeResponse(http.StatusOK, `{"data":[{"title":"DataItem","url":"https://data.dev","snippet":"Info"}]}`), nil
		case "data_nested_results":
			return makeResponse(http.StatusOK, `{"data":{"results":[{"title":"NestedR","url":"https://nr.dev","snippet":"Info"}]}}`), nil
		case "data_nested_sources":
			return makeResponse(http.StatusOK, `{"data":{"sources":[{"title":"NestedS","url":"https://ns.dev","snippet":"Info"}]}}`), nil
		case "empty_json":
			return makeResponse(http.StatusOK, `{"status":"ok"}`), nil
		case "invalid_json":
			return makeResponse(http.StatusOK, `not-json`), nil
		case "500":
			return makeResponse(http.StatusInternalServerError, ``), nil
		case "405_then_ok":
			if r.Method == http.MethodGet {
				return makeResponse(http.StatusMethodNotAllowed, ``), nil
			}
			return makeResponse(http.StatusOK, `{"results":[{"title":"FromPost","url":"https://post.dev","snippet":"Post snippet"}]}`), nil
		case "405_then_err":
			if r.Method == http.MethodGet {
				return makeResponse(http.StatusMethodNotAllowed, ``), nil
			}
			return nil, errors.New("post network error")
		default:
			return makeResponse(http.StatusOK, `{}`), nil
		}
	})

	mode = "network_err"
	_, err = client.Search("golang")
	assert.Error(t, err)

	mode = "results"
	res, err := client.Search("golang")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "Go 1.27", res[0].Title)

	mode = "sources"
	res, err = client.Search("golang")
	assert.NoError(t, err)
	assert.Len(t, res, 1)

	mode = "data_list"
	res, err = client.Search("golang")
	assert.NoError(t, err)
	assert.Len(t, res, 1)

	mode = "data_nested_results"
	res, err = client.Search("golang")
	assert.NoError(t, err)
	assert.Len(t, res, 1)

	mode = "data_nested_sources"
	res, err = client.Search("golang")
	assert.NoError(t, err)
	assert.Len(t, res, 1)

	mode = "empty_json"
	res, err = client.Search("golang")
	assert.NoError(t, err)
	assert.Empty(t, res)

	mode = "invalid_json"
	_, err = client.Search("golang")
	assert.Error(t, err)

	mode = "500"
	_, err = client.Search("golang")
	assert.Error(t, err)

	mode = "405_then_ok"
	res, err = client.Search("golang")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "FromPost", res[0].Title)

	mode = "405_then_err"
	_, err = client.Search("golang")
	assert.Error(t, err)

	// 4.5. POST NewRequest error on 405 fallback (control char in endpoint on POST)
	clientBadPost := NewSearchClient("https://search.delcom.org/api", "secret-token").(*httpSearchClient)
	clientBadPost.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		clientBadPost.endpoint = "://bad-post-url"
		return makeResponse(http.StatusMethodNotAllowed, ``), nil
	})
	_, err = clientBadPost.Search("golang")
	assert.Error(t, err)

	// Test parseSearchResponse read error directly
	_, err = parseSearchResponse(&http.Response{
		StatusCode: http.StatusOK,
		Body:       ioCloser{errReader{}},
	})
	assert.Error(t, err)
}

func TestAIClient(t *testing.T) {
	// 1. Missing API key
	cNoKey := NewAIClient("https://api.example.com", "", "model")
	_, err := cNoKey.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)

	// 2. Bad endpoint URL
	cBadURL := NewAIClient("://bad", "valid-key", "model")
	_, err = cBadURL.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)

	// 3. Mock AI transport modes
	mode := "ok"
	aiC := NewAIClient("https://ai.delcom.org/api/v1", "real-key", "qwen3.8-flash").(*openAIClient)
	aiC.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch mode {
		case "network_err":
			return nil, errors.New("ai unreachable")
		case "ok":
			return makeResponse(http.StatusOK, `{"choices":[{"message":{"content":"`+"```json\\n"+`{\"title\":\"Judul AI\",\"category\":\"\",\"excerpt\":\"Ringkasan\",\"content\":\"## Subjudul\\nIsi artikel\"}`+"\\n```"+`"}}]}`), nil
		case "500":
			return makeResponse(http.StatusInternalServerError, ``), nil
		case "bad_outer_json":
			return makeResponse(http.StatusOK, `not-json`), nil
		case "bad_inner_json":
			return makeResponse(http.StatusOK, `{"choices":[{"message":{"content":"not-json"}}]}`), nil
		case "empty_fields":
			return makeResponse(http.StatusOK, `{"choices":[{"message":{"content":"{\"title\":\"\",\"content\":\"\"}"}}]}`), nil
		case "read_err":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioCloser{errReader{}},
			}, nil
		default:
			return makeResponse(http.StatusOK, `{}`), nil
		}
	})

	mode = "network_err"
	_, err = aiC.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)

	mode = "ok"
	res, err := aiC.GenerateArticle("Topik", "", nil)
	assert.NoError(t, err)
	assert.Equal(t, "Judul AI", res.Title)
	assert.Equal(t, "Umum", res.Category)

	// Also test fallback when AI returns empty category AND caller passed non-empty category
	resCat, err := aiC.GenerateArticle("Topik", "Bisnis", nil)
	assert.NoError(t, err)
	assert.Equal(t, "Bisnis", resCat.Category)

	res, err = aiC.GenerateArticle("Topik", "Teknologi", []SearchSource{{Title: "Ref", URL: "https://ref", Snippet: "Snip"}})
	assert.NoError(t, err)
	assert.Equal(t, "Teknologi", res.Category)
	assert.Len(t, res.Sources, 1)

	mode = "500"
	_, err = aiC.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)

	mode = "bad_outer_json"
	_, err = aiC.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)

	mode = "bad_inner_json"
	_, err = aiC.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)

	mode = "empty_fields"
	_, err = aiC.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)

	mode = "read_err"
	_, err = aiC.GenerateArticle("Topik", "Teknologi", nil)
	assert.Error(t, err)
}

func TestAIServiceAndHandler(t *testing.T) {
	sc := NewSearchClient("https://search.delcom.org", "valid-api-key").(*httpSearchClient)
	sc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusOK, `{"results":[{"title":"Ref 1","url":"https://ref1.org","snippet":"Snippet 1"}]}`), nil
	})

	aiMode := "ok"
	ac := NewAIClient("https://ai.delcom.org", "valid-api-key", "model").(*openAIClient)
	ac.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if aiMode == "fail" {
			return makeResponse(http.StatusBadGateway, ``), nil
		}
		return makeResponse(http.StatusOK, `{"choices":[{"message":{"content":"{\"title\":\"Artikel AI\",\"category\":\"Teknologi\",\"excerpt\":\"Excerpt AI\",\"content\":\"Konten AI\"}"}}]}`), nil
	})

	svc := NewService(sc, ac)
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/posts/ai-generate", h.GenerateBlog)

	// 1. Invalid JSON
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBufferString("{bad")))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 2. Empty topic
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBufferString(`{"topic":"   "}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 3. Success with web search enabled (default)
	bReq, _ := json.Marshal(GenerateBlogRequest{
		Topic:    "Masa Depan AI",
		Category: "Teknologi",
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBuffer(bReq)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Success with web search explicitly false
	fFalse := false
	bReqNoSearch, _ := json.Marshal(GenerateBlogRequest{
		Topic:            "Masa Depan AI",
		Category:         "Teknologi",
		IncludeWebSearch: &fFalse,
	})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBuffer(bReqNoSearch)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. AI client error -> 500 Internal Server Error
	aiMode = "fail"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBuffer(bReq)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
