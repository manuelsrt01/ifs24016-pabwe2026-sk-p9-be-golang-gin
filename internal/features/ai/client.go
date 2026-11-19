package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AIClient interface {
	GenerateArticle(topic, category string, sources []SearchSource) (*GeneratedBlogResponse, error)
}

type openAIClient struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewAIClient(endpoint, apiKey, model string) AIClient {
	return &openAIClient{
		endpoint: strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		apiKey:   strings.TrimSpace(apiKey),
		model:    strings.TrimSpace(model),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *openAIClient) GenerateArticle(topic, category string, sources []SearchSource) (*GeneratedBlogResponse, error) {
	if c.apiKey == "" || c.apiKey == "key" {
		return nil, errors.New("OPENAI_KEY belum dikonfigurasi pada environment")
	}

	var contextBuilder strings.Builder
	if len(sources) > 0 {
		contextBuilder.WriteString("\n\nKonteks dan Referensi dari Search Engine:\n")
		for i, src := range sources {
			contextBuilder.WriteString(fmt.Sprintf("%d. [%s](%s) - %s\n", i+1, src.Title, src.URL, src.Snippet))
		}
	}

	systemPrompt := `Anda adalah penulis blog profesional berbahasa Indonesia untuk platform Delcom Blog.
Balas HANYA dengan satu objek JSON valid tanpa markdown code fence tambahan, dengan struktur:
{
  "title": "Judul Artikel Menarik",
  "category": "Teknologi|Pendidikan|Gaya Hidup|Bisnis|Opini|Umum",
  "excerpt": "Ringkasan 1-2 kalimat pemikat pembaca.",
  "content": "Isi artikel lengkap dalam format Markdown dengan subjudul ##, daftar poin, dan sitasi sumber referensi jika tersedia."
}`

	userPrompt := fmt.Sprintf("Tuliskan artikel blog mendalam tentang topik: %q dengan kategori: %q.%s", topic, category, contextBuilder.String())

	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.7,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("layanan AI mengembalikan status %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBytes, &completion); err != nil || len(completion.Choices) == 0 {
		return nil, errors.New("respons dari layanan AI tidak valid")
	}

	rawContent := cleanJSONBlock(completion.Choices[0].Message.Content)

	var parsed struct {
		Title    string `json:"title"`
		Category string `json:"category"`
		Excerpt  string `json:"excerpt"`
		Content  string `json:"content"`
	}

	if err := json.Unmarshal([]byte(rawContent), &parsed); err != nil {
		return nil, errors.New("gagal memproses JSON artikel dari AI")
	}

	if strings.TrimSpace(parsed.Title) == "" || strings.TrimSpace(parsed.Content) == "" {
		return nil, errors.New("hasil AI tidak memiliki judul atau konten")
	}

	cat := strings.TrimSpace(parsed.Category)
	if cat == "" {
		cat = category
	}
	if cat == "" {
		cat = "Umum"
	}

	if sources == nil {
		sources = []SearchSource{}
	}

	return &GeneratedBlogResponse{
		Title:         strings.TrimSpace(parsed.Title),
		Category:      cat,
		Excerpt:       strings.TrimSpace(parsed.Excerpt),
		Content:       strings.TrimSpace(parsed.Content),
		Sources:       sources,
		SearchSources: sources,
	}, nil
}

func cleanJSONBlock(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}
