package post

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"app/internal/database"
)

var (
	ErrInvalidPostInput = errors.New("judul dan konten artikel wajib diisi")
	ErrPostNotFound     = errors.New("artikel tidak ditemukan")
	ErrForbidden        = errors.New("anda tidak memiliki akses ke artikel ini")
)

var nonAlnumRegex = regexp.MustCompile(`[^a-z0-9]+`)

type Service interface {
	GetCategories() []string
	ListPublicPosts(query, category string) ([]PostResponse, error)
	GetPublicPostBySlug(slug string) (*PostResponse, error)
	ListAuthorPosts(userID uint) ([]PostResponse, error)
	GetAuthorPostByID(userID, postID uint, isAdmin bool) (*PostResponse, error)
	CreatePost(userID uint, req CreatePostRequest) (*PostResponse, error)
	UpdatePost(userID, postID uint, isAdmin bool, req UpdatePostRequest) (*PostResponse, error)
	TogglePublish(userID, postID uint, isAdmin bool) (*PostResponse, error)
	DeletePost(userID, postID uint, isAdmin bool) error
	GetAuthorStats(userID uint) (*PostStatsResponse, error)
	AdminListUsers() ([]AdminUserSummary, error)
	AdminListAllPosts() ([]PostResponse, error)
	AdminListUserPosts(targetUserID uint) ([]PostResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func CalculateReadingTime(content string) int {
	words := strings.Fields(strings.TrimSpace(content))
	if len(words) == 0 {
		return 1
	}
	return int(math.Ceil(float64(len(words)) / 200.0))
}

func Slugify(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))
	slug := nonAlnumRegex.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "artikel"
	}
	return slug
}

func NormalizeCategory(cat string) string {
	trimmed := strings.TrimSpace(cat)
	for _, valid := range ValidCategories {
		if strings.EqualFold(trimmed, valid) {
			return valid
		}
	}
	return "Umum"
}

func BuildExcerpt(excerpt, content string) string {
	trimmed := strings.TrimSpace(excerpt)
	if trimmed != "" {
		return trimmed
	}
	plain := strings.Join(strings.Fields(content), " ")
	if len(plain) > 160 {
		return plain[:160] + "..."
	}
	return plain
}

func toPostResponse(p *database.Post) PostResponse {
	rt := CalculateReadingTime(p.Content)
	res := PostResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		AuthorName:     "Penulis",
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        BuildExcerpt(p.Excerpt, p.Content),
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    rt,
		ReadingMinutes: rt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if p.User != nil && p.User.ID != 0 {
		res.AuthorName = p.User.Username
		res.Author = &AuthorInfo{
			ID:       p.User.ID,
			Username: p.User.Username,
			Email:    p.User.Email,
			Role:     p.User.Role,
		}
	}
	return res
}

func toPublicPostResponse(p *database.Post) PostResponse {
	res := toPostResponse(p)
	res.UserID = 0
	return res
}

func (s *service) generateUniqueSlug(title string, excludeID uint) (string, error) {
	base := Slugify(title)
	slug := base
	counter := 2
	for {
		exists, err := s.repo.SlugExists(slug, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, counter)
		counter++
	}
}

func (s *service) GetCategories() []string {
	result := make([]string, len(ValidCategories))
	copy(result, ValidCategories)
	return result
}

func (s *service) ListPublicPosts(query, category string) ([]PostResponse, error) {
	posts, err := s.repo.FindPublished(query, category)
	if err != nil {
		return nil, err
	}
	res := make([]PostResponse, 0, len(posts))
	for i := range posts {
		res = append(res, toPublicPostResponse(&posts[i]))
	}
	return res, nil
}

func (s *service) GetPublicPostBySlug(slug string) (*PostResponse, error) {
	p, err := s.repo.FindPublishedBySlug(slug)
	if err != nil {
		return nil, ErrPostNotFound
	}
	resp := toPublicPostResponse(p)
	return &resp, nil
}

func (s *service) ListAuthorPosts(userID uint) ([]PostResponse, error) {
	posts, err := s.repo.FindByUser(userID)
	if err != nil {
		return nil, err
	}
	res := make([]PostResponse, 0, len(posts))
	for i := range posts {
		res = append(res, toPostResponse(&posts[i]))
	}
	return res, nil
}

func (s *service) GetAuthorPostByID(userID, postID uint, isAdmin bool) (*PostResponse, error) {
	p, err := s.repo.FindByID(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if p.UserID != userID && !isAdmin {
		return nil, ErrForbidden
	}
	resp := toPostResponse(p)
	return &resp, nil
}

func (s *service) CreatePost(userID uint, req CreatePostRequest) (*PostResponse, error) {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return nil, ErrInvalidPostInput
	}

	slug, err := s.generateUniqueSlug(title, 0)
	if err != nil {
		return nil, err
	}

	var publishedAt *time.Time
	if req.IsPublished {
		now := time.Now()
		publishedAt = &now
	}

	p := &database.Post{
		UserID:         userID,
		Title:          title,
		Slug:           slug,
		Category:       NormalizeCategory(req.Category),
		Excerpt:        BuildExcerpt(req.Excerpt, content),
		Content:        content,
		CoverImagePath: strings.TrimSpace(req.CoverImagePath),
		IsPublished:    req.IsPublished,
		PublishedAt:    publishedAt,
	}

	if err := s.repo.Create(p); err != nil {
		return nil, err
	}

	created, err := s.repo.FindByID(p.ID)
	if err == nil {
		p = created
	}

	resp := toPostResponse(p)
	return &resp, nil
}

func (s *service) UpdatePost(userID, postID uint, isAdmin bool, req UpdatePostRequest) (*PostResponse, error) {
	p, err := s.repo.FindByID(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if p.UserID != userID && !isAdmin {
		return nil, ErrForbidden
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = p.Title
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = p.Content
	}

	if title != p.Title {
		newSlug, err := s.generateUniqueSlug(title, p.ID)
		if err != nil {
			return nil, err
		}
		p.Slug = newSlug
	}

	p.Title = title
	if strings.TrimSpace(req.Category) != "" {
		p.Category = NormalizeCategory(req.Category)
	}
	if strings.TrimSpace(req.Excerpt) != "" {
		p.Excerpt = strings.TrimSpace(req.Excerpt)
	} else {
		p.Excerpt = BuildExcerpt(p.Excerpt, content)
	}
	p.Content = content
	if req.CoverImagePath != "" {
		p.CoverImagePath = strings.TrimSpace(req.CoverImagePath)
	}

	if req.IsPublished != nil {
		p.IsPublished = *req.IsPublished
		if p.IsPublished && p.PublishedAt == nil {
			now := time.Now()
			p.PublishedAt = &now
		} else if !p.IsPublished {
			p.PublishedAt = nil
		}
	}

	if err := s.repo.Update(p); err != nil {
		return nil, err
	}

	resp := toPostResponse(p)
	return &resp, nil
}

func (s *service) TogglePublish(userID, postID uint, isAdmin bool) (*PostResponse, error) {
	p, err := s.repo.FindByID(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if p.UserID != userID && !isAdmin {
		return nil, ErrForbidden
	}

	p.IsPublished = !p.IsPublished
	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	} else {
		p.PublishedAt = nil
	}

	if err := s.repo.Update(p); err != nil {
		return nil, err
	}

	resp := toPostResponse(p)
	return &resp, nil
}

func (s *service) DeletePost(userID, postID uint, isAdmin bool) error {
	p, err := s.repo.FindByID(postID)
	if err != nil {
		return ErrPostNotFound
	}
	if p.UserID != userID && !isAdmin {
		return ErrForbidden
	}
	return s.repo.Delete(p)
}

func (s *service) GetAuthorStats(userID uint) (*PostStatsResponse, error) {
	total, published, draft, err := s.repo.CountStatsByUser(userID)
	if err != nil {
		return nil, err
	}
	return &PostStatsResponse{
		Total:         total,
		Published:     published,
		Draft:         draft,
		TotalArticles: total,
		TotalArtikel:  total,
		TotalTerbit:   published,
		TotalSelesai:  published,
		BelumSelesai:  draft,
	}, nil
}

func (s *service) AdminListUsers() ([]AdminUserSummary, error) {
	users, err := s.repo.FindAllUsers()
	if err != nil {
		return nil, err
	}
	res := make([]AdminUserSummary, 0, len(users))
	for _, u := range users {
		var total, pub int64
		total = int64(len(u.Posts))
		for _, p := range u.Posts {
			if p.IsPublished {
				pub++
			}
		}
		res = append(res, AdminUserSummary{
			ID:             u.ID,
			Username:       u.Username,
			Email:          u.Email,
			Role:           u.Role,
			TotalPosts:     total,
			PublishedPosts: pub,
			DraftPosts:     total - pub,
			TotalArticles:  total,
			TotalPublished: pub,
			TotalArtikel:   total,
			TotalTerbit:    pub,
			CreatedAt:      u.CreatedAt,
			UpdatedAt:      u.UpdatedAt,
		})
	}
	return res, nil
}

func (s *service) AdminListAllPosts() ([]PostResponse, error) {
	posts, err := s.repo.FindAllPosts()
	if err != nil {
		return nil, err
	}
	res := make([]PostResponse, 0, len(posts))
	for i := range posts {
		res = append(res, toPostResponse(&posts[i]))
	}
	return res, nil
}

func (s *service) AdminListUserPosts(targetUserID uint) ([]PostResponse, error) {
	return s.ListAuthorPosts(targetUserID)
}
