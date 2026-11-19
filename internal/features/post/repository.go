package post

import (
	"strings"

	"app/internal/database"

	"gorm.io/gorm"
)

type Repository interface {
	FindPublished(query, category string) ([]database.Post, error)
	FindPublishedBySlug(slug string) (*database.Post, error)
	FindByUser(userID uint) ([]database.Post, error)
	FindByID(id uint) (*database.Post, error)
	Create(post *database.Post) error
	Update(post *database.Post) error
	Delete(post *database.Post) error
	SlugExists(slug string, excludeID uint) (bool, error)
	CountStatsByUser(userID uint) (total int64, published int64, draft int64, err error)
	FindAllUsers() ([]database.User, error)
	FindAllPosts() ([]database.Post, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) FindPublished(query, category string) ([]database.Post, error) {
	var posts []database.Post
	tx := r.db.Preload("User").Where("is_published = ?", true)

	q := strings.TrimSpace(query)
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", like, like, like)
	}

	cat := strings.TrimSpace(category)
	if cat != "" && !strings.EqualFold(cat, "Semua") {
		tx = tx.Where("LOWER(category) = ?", strings.ToLower(cat))
	}

	if err := tx.Order("published_at DESC, id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *gormRepository) FindPublishedBySlug(slug string) (*database.Post, error) {
	var p database.Post
	if err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) FindByUser(userID uint) ([]database.Post, error) {
	var posts []database.Post
	if err := r.db.Preload("User").Where("user_id = ?", userID).Order("created_at DESC, id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *gormRepository) FindByID(id uint) (*database.Post, error) {
	var p database.Post
	if err := r.db.Preload("User").First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) Create(p *database.Post) error {
	return r.db.Create(p).Error
}

func (r *gormRepository) Update(p *database.Post) error {
	return r.db.Save(p).Error
}

func (r *gormRepository) Delete(p *database.Post) error {
	return r.db.Delete(p).Error
}

func (r *gormRepository) SlugExists(slug string, excludeID uint) (bool, error) {
	var count int64
	tx := r.db.Model(&database.Post{}).Where("slug = ?", slug)
	if excludeID > 0 {
		tx = tx.Where("id != ?", excludeID)
	}
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *gormRepository) CountStatsByUser(userID uint) (int64, int64, int64, error) {
	var total, published int64
	if err := r.db.Model(&database.Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, 0, 0, err
	}
	if err := r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", userID, true).Count(&published).Error; err != nil {
		return 0, 0, 0, err
	}
	return total, published, total - published, nil
}

func (r *gormRepository) FindAllUsers() ([]database.User, error) {
	var users []database.User
	if err := r.db.Preload("Posts").Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *gormRepository) FindAllPosts() ([]database.Post, error) {
	var posts []database.Post
	if err := r.db.Preload("User").Order("created_at DESC, id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}
