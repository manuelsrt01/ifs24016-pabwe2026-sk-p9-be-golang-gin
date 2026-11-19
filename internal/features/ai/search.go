package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchEngine interface {
	Search(query string) ([]SearchSource, error)
}

type httpSearchClient struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewSearchClient(endpoint, apiKey string) SearchEngine {
	return &httpSearchClient{
		endpoint: strings.TrimSpace(endpoint),
		apiKey:   strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *httpSearchClient) Search(query string) ([]SearchSource, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, errors.New("topik pencarian tidak boleh kosong")
	}
	if s.endpoint == "" {
		return nil, errors.New("SEARCH_ENGINE_ENDPOINT belum dikonfigurasi")
	}

	reqURL := fmt.Sprintf("%s?q=%s", s.endpoint, url.QueryEscape(q))
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If GET returns 405 Method Not Allowed, fallback to POST with JSON body
	if resp.StatusCode == http.StatusMethodNotAllowed {
		payload, _ := json.Marshal(map[string]string{"query": q, "q": q})
		postReq, reqErr := http.NewRequest(http.MethodPost, s.endpoint, bytes.NewBuffer(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		postReq.Header.Set("Content-Type", "application/json")
		postReq.Header.Set("Accept", "application/json")
		if s.apiKey != "" {
			postReq.Header.Set("Authorization", "Bearer "+s.apiKey)
		}
		postResp, postErr := s.httpClient.Do(postReq)
		if postErr != nil {
			return nil, postErr
		}
		defer postResp.Body.Close()
		return parseSearchResponse(postResp)
	}

	return parseSearchResponse(resp)
}

func parseSearchResponse(resp *http.Response) ([]SearchSource, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search engine gagal dengan status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Results []SearchSource `json:"results"`
		Sources []SearchSource `json:"sources"`
		Data    any            `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, errors.New("format respons search engine tidak valid")
	}

	if len(raw.Results) > 0 {
		return filterSources(raw.Results), nil
	}
	if len(raw.Sources) > 0 {
		return filterSources(raw.Sources), nil
	}

	if raw.Data != nil {
		dataBytes, _ := json.Marshal(raw.Data)
		var directList []SearchSource
		if err := json.Unmarshal(dataBytes, &directList); err == nil && len(directList) > 0 {
			return filterSources(directList), nil
		}
		var nested struct {
			Results []SearchSource `json:"results"`
			Sources []SearchSource `json:"sources"`
		}
		if err := json.Unmarshal(dataBytes, &nested); err == nil {
			if len(nested.Results) > 0 {
				return filterSources(nested.Results), nil
			}
			if len(nested.Sources) > 0 {
				return filterSources(nested.Sources), nil
			}
		}
	}

	return []SearchSource{}, nil
}

func filterSources(items []SearchSource) []SearchSource {
	out := make([]SearchSource, 0, len(items))
	for _, it := range items {
		t := strings.TrimSpace(it.Title)
		u := strings.TrimSpace(it.URL)
		sn := strings.TrimSpace(it.Snippet)
		if t == "" && u == "" && sn == "" {
			continue
		}
		out = append(out, SearchSource{
			Title:   t,
			URL:     u,
			Snippet: sn,
		})
	}
	return out
}
