package auth

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("username, email, dan password tidak valid (password minimal 6 karakter dan email harus valid)")
	ErrUsernameTaken      = errors.New("username sudah terdaftar")
	ErrEmailTaken         = errors.New("email sudah terdaftar")
	ErrInvalidCredentials = errors.New("username atau password salah")
	ErrUserNotFound       = errors.New("pengguna tidak ditemukan")
)

var bcryptHash = func(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

var signToken = func(token *jwt.Token, key []byte) (string, error) {
	return token.SignedString(key)
}

type Service interface {
	Register(req RegisterRequest) (*LoginResponse, error)
	Login(req LoginRequest) (*LoginResponse, error)
	GetProfile(userID uint) (*UserResponse, error)
	UpdateProfile(userID uint, req UpdateProfileRequest) (*UserResponse, error)
}

type service struct {
	repo      Repository
	secretKey string
}

func NewService(repo Repository, secretKey string) Service {
	return &service{repo: repo, secretKey: secretKey}
}

func isValidEmail(email string) bool {
	if !strings.Contains(email, "@") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func toUserResponse(u *database.User) *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (s *service) buildLoginResponse(user *database.User) (*LoginResponse, error) {
	claims := middleware.UserClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := signToken(token, []byte(s.secretKey))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: signed,
		User:  *toUserResponse(user),
	}, nil
}

func (s *service) Register(req RegisterRequest) (*LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)

	if username == "" || email == "" || len(password) < 6 || !isValidEmail(email) {
		return nil, ErrInvalidInput
	}

	if existing, _ := s.repo.FindByUsername(username); existing != nil {
		return nil, ErrUsernameTaken
	}

	if existing, _ := s.repo.FindByEmail(email); existing != nil {
		return nil, ErrEmailTaken
	}

	hashed, err := bcryptHash(password)
	if err != nil {
		return nil, err
	}

	role := "member"
	if strings.EqualFold(strings.TrimSpace(req.Role), "admin") {
		role = "admin"
	}

	user := &database.User{
		Username: username,
		Email:    email,
		Password: string(hashed),
		Role:     role,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return s.buildLoginResponse(user)
}

func (s *service) Login(req LoginRequest) (*LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	password := req.Password

	if username == "" || password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.repo.FindByUsername(username)
	if err != nil {
		user, err = s.repo.FindByEmail(username)
		if err != nil {
			return nil, ErrInvalidCredentials
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.buildLoginResponse(user)
}

func (s *service) GetProfile(userID uint) (*UserResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return toUserResponse(user), nil
}

func (s *service) UpdateProfile(userID uint, req UpdateProfileRequest) (*UserResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || !isValidEmail(email) || (password != "" && len(password) < 6) {
		return nil, ErrInvalidInput
	}

	if username != "" && username != user.Username {
		if existing, _ := s.repo.FindByUsername(username); existing != nil && existing.ID != userID {
			return nil, ErrUsernameTaken
		}
		user.Username = username
	}

	if email != user.Email {
		if existing, _ := s.repo.FindByEmail(email); existing != nil && existing.ID != userID {
			return nil, ErrEmailTaken
		}
		user.Email = email
	}

	if password != "" {
		hashed, err := bcryptHash(password)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashed)
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	return toUserResponse(user), nil
}
