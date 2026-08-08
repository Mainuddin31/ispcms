package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
)

type RoleService interface {
	List() ([]models.Role, error)
	GetByID(id uuid.UUID) (*models.Role, error)
	Create(req CreateRoleRequest) (*models.Role, error)
	Update(id uuid.UUID, req UpdateRoleRequest) (*models.Role, error)
	Delete(id uuid.UUID) error
	ListPermissions() ([]models.Permission, error)
	AssignPermission(roleID, permID uuid.UUID) error
	RemovePermission(roleID, permID uuid.UUID) error
	SetPermissions(roleID uuid.UUID, permIDs []uuid.UUID) error
}

type CreateRoleRequest struct {
	Name        string `json:"name" validate:"required"`
	DisplayName string `json:"display_name" validate:"required"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

type roleService struct {
	roleRepo repositories.RoleRepository
}

func NewRoleService(roleRepo repositories.RoleRepository) RoleService {
	return &roleService{roleRepo: roleRepo}
}

func (s *roleService) List() ([]models.Role, error) {
	return s.roleRepo.FindAll()
}

func (s *roleService) GetByID(id uuid.UUID) (*models.Role, error) {
	return s.roleRepo.FindByID(id)
}

func (s *roleService) Create(req CreateRoleRequest) (*models.Role, error) {
	if _, err := s.roleRepo.FindByName(req.Name); err == nil {
		return nil, errors.New("role name already exists")
	}
	role := &models.Role{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
	}
	if err := s.roleRepo.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) Update(id uuid.UUID, req UpdateRoleRequest) (*models.Role, error) {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("role not found")
	}
	if req.DisplayName != "" {
		role.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if err := s.roleRepo.Update(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) Delete(id uuid.UUID) error {
	return s.roleRepo.Delete(id)
}

func (s *roleService) ListPermissions() ([]models.Permission, error) {
	return s.roleRepo.FindAllPermissions()
}

func (s *roleService) AssignPermission(roleID, permID uuid.UUID) error {
	return s.roleRepo.AssignPermission(roleID, permID)
}

func (s *roleService) RemovePermission(roleID, permID uuid.UUID) error {
	return s.roleRepo.RemovePermission(roleID, permID)
}

func (s *roleService) SetPermissions(roleID uuid.UUID, permIDs []uuid.UUID) error {
	role, err := s.roleRepo.FindByID(roleID)
	if err != nil {
		return errors.New("role not found")
	}
	// Remove all existing permissions
	for _, p := range role.Permissions {
		_ = s.roleRepo.RemovePermission(roleID, p.ID)
	}
	// Assign new ones
	for _, pid := range permIDs {
		if err := s.roleRepo.AssignPermission(roleID, pid); err != nil {
			return err
		}
	}
	return nil
}
