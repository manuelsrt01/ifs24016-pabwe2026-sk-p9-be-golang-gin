package post

import "time"

var ValidCategories = []string{
	"Teknologi",
	"Pendidikan",
	"Gaya Hidup",
	"Bisnis",
	"Opini",
	"Umum",
}

type CreatePostRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

type UpdatePostRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published,omitempty"`
}

type AuthorInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type PostResponse struct {
	ID             uint        `json:"id"`
	UserID         uint        `json:"user_id"`
	Title          string      `json:"title"`
	Slug           string      `json:"slug"`
	Category       string      `json:"category"`
	Excerpt        string      `json:"excerpt"`
	Content        string      `json:"content"`
	CoverImagePath string      `json:"cover_image_path"`
	IsPublished    bool        `json:"is_published"`
	PublishedAt    *time.Time  `json:"published_at"`
	ReadingTime    int         `json:"reading_time"`
	ReadingMinutes int         `json:"reading_minutes"`
	Author         *AuthorInfo `json:"author,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type PostStatsResponse struct {
	Total          int64 `json:"total"`
	Published      int64 `json:"published"`
	Draft          int64 `json:"draft"`
	TotalArtikel   int64 `json:"total_artikel"`
	TotalTerbit    int64 `json:"total_terbit"`
	TotalSelesai   int64 `json:"total_selesai"`
	BelumSelesai   int64 `json:"belum_selesai"`
}

type AdminUserSummary struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	TotalPosts     int64     `json:"total_posts"`
	PublishedPosts int64     `json:"published_posts"`
	DraftPosts     int64     `json:"draft_posts"`
	TotalArtikel   int64     `json:"total_artikel"`
	TotalTerbit    int64     `json:"total_terbit"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
