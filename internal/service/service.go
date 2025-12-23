package service

import (
	"errors"
	"strings"
)

// User is a small example domain entity used for unit testing in PZ-15.
type User struct {
	ID    int64
	Email string
}

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidEmail = errors.New("invalid email")
	ErrAlreadyExists = errors.New("already exists")
)

// UserRepo is a dependency that must be stubbed in unit tests.
type UserRepo interface {
	ByEmail(email string) (User, error)
	Create(email string) (User, error)
}

// Service contains business logic independent from transport/database.
type Service struct {
	repo UserRepo
}

func New(repo UserRepo) *Service {
	return &Service{repo: repo}
}

// FindIDByEmail validates email and returns the user id.
func (s *Service) FindIDByEmail(email string) (int64, error) {
	email = strings.TrimSpace(email)
	if !isEmailLike(email) {
		return 0, ErrInvalidEmail
	}
	u, err := s.repo.ByEmail(email)
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}

// Register creates a new user if it does not exist.
func (s *Service) Register(email string) (User, error) {
	email = strings.TrimSpace(email)
	if !isEmailLike(email) {
		return User{}, ErrInvalidEmail
	}
	// Check existing
	if _, err := s.repo.ByEmail(email); err == nil {
		return User{}, ErrAlreadyExists
	} else if !errors.Is(err, ErrNotFound) {
		return User{}, err
	}
	// Create new
	return s.repo.Create(email)
}

func isEmailLike(s string) bool {
	if len(s) < 3 {
		return false
	}
	// Simplified check for a coursework.
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	return true
}
