package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/config"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	jwtpkg "github.com/ispcms/backend/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountLocked      = errors.New("account is locked due to too many failed attempts")
	ErrAccountDisabled    = errors.New("account is disabled")
)

type AuthService interface {
	Login(username, password string) (*jwtpkg.TokenPair, *models.User, error)
	RefreshToken(refreshToken string) (*jwtpkg.TokenPair, error)
	ChangePassword(userID uuid.UUID, oldPass, newPass string) error
	ResetPassword(userID uuid.UUID, newPass string) error
}

type authService struct {
	userRepo repositories.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo repositories.UserRepository, cfg *config.Config) AuthService {
	return &authService{userRepo: userRepo, cfg: cfg}
}

func (s *authService) Login(username, password string) (*jwtpkg.TokenPair, *models.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		user, err = s.userRepo.FindByEmail(username)
		if err != nil {
			return nil, nil, ErrInvalidCredentials
		}
	}
	if user.Status == "disabled" {
		return nil, nil, ErrAccountDisabled
	}
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, nil, ErrAccountLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.userRepo.IncrementFailedAttempts(user.ID)
		if user.FailedAttempts+1 >= s.cfg.MaxLoginAttempts {
			s.userRepo.LockUser(user.ID, s.cfg.LockDurationMinutes)
		}
		return nil, nil, ErrInvalidCredentials
	}

	s.userRepo.ResetFailedAttempts(user.ID)
	s.userRepo.UpdateLastLogin(user.ID)

	roleNames := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleNames = append(roleNames, r.Name)
	}

	tokens, err := jwtpkg.GenerateTokenPair(s.cfg.JWTSecret, user.ID, user.Username, roleNames)
	if err != nil {
		return nil, nil, fmt.Errorf("generating tokens: %w", err)
	}
	return tokens, user, nil
}

func (s *authService) RefreshToken(refreshToken string) (*jwtpkg.TokenPair, error) {
	claims, err := jwtpkg.ValidateToken(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	roleNames := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleNames = append(roleNames, r.Name)
	}
	return jwtpkg.GenerateTokenPair(s.cfg.JWTSecret, user.ID, user.Username, roleNames)
}

func (s *authService) ChangePassword(userID uuid.UUID, oldPass, newPass string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPass)); err != nil {
		return errors.New("current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), s.cfg.BcryptCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return s.userRepo.Update(user)
}

func (s *authService) ResetPassword(userID uuid.UUID, newPass string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), s.cfg.BcryptCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return s.userRepo.Update(user)
}
