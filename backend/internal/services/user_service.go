package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/config"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	List(page, pageSize int, search string) ([]models.User, int64, error)
	GetByID(id uuid.UUID) (*models.User, error)
	Create(req CreateUserRequest, createdBy uuid.UUID) (*models.User, error)
	Update(id uuid.UUID, req UpdateUserRequest) (*models.User, error)
	Delete(id uuid.UUID) error
	AssignRole(userID, roleID uuid.UUID) error
	RemoveRole(userID, roleID uuid.UUID) error
	SetStatus(id uuid.UUID, status string) error
}

type CreateUserRequest struct {
	FullName string    `json:"full_name" validate:"required"`
	Username string    `json:"username" validate:"required"`
	Email    string    `json:"email" validate:"required,email"`
	Phone    string    `json:"phone"`
	Password string    `json:"password" validate:"required,min=8"`
	RoleID   uuid.UUID `json:"role_id"`
}

type UpdateUserRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
}

type userService struct {
	userRepo repositories.UserRepository
	cfg      *config.Config
}

func NewUserService(userRepo repositories.UserRepository, cfg *config.Config) UserService {
	return &userService{userRepo: userRepo, cfg: cfg}
}

func (s *userService) List(page, pageSize int, search string) ([]models.User, int64, error) {
	return s.userRepo.FindAll(page, pageSize, search)
}

func (s *userService) GetByID(id uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *userService) Create(req CreateUserRequest, createdBy uuid.UUID) (*models.User, error) {
	// Check username uniqueness
	if _, err := s.userRepo.FindByUsername(req.Username); err == nil {
		return nil, errors.New("username already taken")
	}
	if _, err := s.userRepo.FindByEmail(req.Email); err == nil {
		return nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BcryptCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		FullName:  req.FullName,
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  string(hash),
		Status:    "active",
		CreatedBy: &createdBy,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	if req.RoleID != uuid.Nil {
		_ = s.userRepo.AssignRole(user.ID, req.RoleID)
	}
	return s.userRepo.FindByID(user.ID)
}

func (s *userService) Update(id uuid.UUID, req UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) Delete(id uuid.UUID) error {
	return s.userRepo.Delete(id)
}

func (s *userService) AssignRole(userID, roleID uuid.UUID) error {
	return s.userRepo.AssignRole(userID, roleID)
}

func (s *userService) RemoveRole(userID, roleID uuid.UUID) error {
	return s.userRepo.RemoveRole(userID, roleID)
}

func (s *userService) SetStatus(id uuid.UUID, status string) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user not found")
	}
	if status != "active" && status != "disabled" {
		return errors.New("invalid status")
	}
	user.Status = status
	return s.userRepo.Update(user)
}
