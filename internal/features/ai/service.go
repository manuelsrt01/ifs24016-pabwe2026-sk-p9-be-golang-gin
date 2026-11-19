package ai

import (
	"errors"
	"strings"

	"app/internal/features/post"
)

var (
	ErrTopicRequired = errors.New("field topic artikel wajib diisi")
)

type Service interface {
	GenerateBlog(req GenerateBlogRequest) (*GeneratedBlogResponse, error)
}

type service struct {
	searchEngine SearchEngine
	aiClient     AIClient
}

func NewService(searchEngine SearchEngine, aiClient AIClient) Service {
	return &service{
		searchEngine: searchEngine,
		aiClient:     aiClient,
	}
}

func (s *service) GenerateBlog(req GenerateBlogRequest) (*GeneratedBlogResponse, error) {
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return nil, ErrTopicRequired
	}

	category := post.NormalizeCategory(req.Category)

	includeSearch := true
	if req.IncludeWebSearch != nil {
		includeSearch = *req.IncludeWebSearch
	}

	sources := []SearchSource{}
	if includeSearch && s.searchEngine != nil {
		found, err := s.searchEngine.Search(topic)
		if err == nil && len(found) > 0 {
			sources = found
		}
	}

	res, err := s.aiClient.GenerateArticle(topic, category, sources)
	if err != nil {
		return nil, err
	}

	res.Category = post.NormalizeCategory(res.Category)
	res.Sources = sources
	res.SearchSources = sources
	return res, nil
}
