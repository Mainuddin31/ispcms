package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/config"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/pkg/mikrotik"
	"github.com/ispcms/backend/pkg/utils"
)

type RouterService interface {
	List(page, pageSize int, search string) ([]models.Router, int64, error)
	GetByID(id uuid.UUID) (*models.Router, error)
	Create(req CreateRouterRequest) (*models.Router, error)
	Update(id uuid.UUID, req UpdateRouterRequest) (*models.Router, error)
	Delete(id uuid.UUID) error
	TestConnection(id uuid.UUID) error
	TestConnectionRaw(ip string, port int, username, password string) error
}

type CreateRouterRequest struct {
	Name        string `json:"name" validate:"required"`
	IPAddress   string `json:"ip_address" validate:"required"`
	APIPort     int    `json:"api_port"`
	Username    string `json:"username" validate:"required"`
	Password    string `json:"password" validate:"required"`
	Location    string `json:"location"`
	POPName     string `json:"pop_name"`
	Description string `json:"description"`
}

type UpdateRouterRequest struct {
	Name        string `json:"name"`
	IPAddress   string `json:"ip_address"`
	APIPort     int    `json:"api_port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Location    string `json:"location"`
	POPName     string `json:"pop_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type routerService struct {
	routerRepo repositories.RouterRepository
	cfg        *config.Config
}

func NewRouterService(routerRepo repositories.RouterRepository, cfg *config.Config) RouterService {
	return &routerService{routerRepo: routerRepo, cfg: cfg}
}

func (s *routerService) List(page, pageSize int, search string) ([]models.Router, int64, error) {
	return s.routerRepo.FindAll(page, pageSize, search)
}

func (s *routerService) GetByID(id uuid.UUID) (*models.Router, error) {
	return s.routerRepo.FindByID(id)
}

func (s *routerService) Create(req CreateRouterRequest) (*models.Router, error) {
	port := req.APIPort
	if port == 0 {
		port = 8728
	}

	encPass, err := utils.Encrypt(req.Password, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("failed to encrypt credentials")
	}

	router := &models.Router{
		Name:             req.Name,
		IPAddress:        req.IPAddress,
		APIPort:          port,
		Username:         req.Username,
		Password:         encPass,
		Location:         req.Location,
		POPName:          req.POPName,
		Description:      req.Description,
		Status:           "active",
		ConnectionStatus: "disconnected",
	}
	if err := s.routerRepo.Create(router); err != nil {
		return nil, err
	}
	return router, nil
}

func (s *routerService) Update(id uuid.UUID, req UpdateRouterRequest) (*models.Router, error) {
	router, err := s.routerRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("router not found")
	}
	if req.Name != "" {
		router.Name = req.Name
	}
	if req.IPAddress != "" {
		router.IPAddress = req.IPAddress
	}
	if req.APIPort != 0 {
		router.APIPort = req.APIPort
	}
	if req.Username != "" {
		router.Username = req.Username
	}
	if req.Password != "" {
		encPass, err := utils.Encrypt(req.Password, s.cfg.JWTSecret)
		if err != nil {
			return nil, err
		}
		router.Password = encPass
	}
	if req.Location != "" {
		router.Location = req.Location
	}
	if req.POPName != "" {
		router.POPName = req.POPName
	}
	if req.Description != "" {
		router.Description = req.Description
	}
	if req.Status != "" {
		router.Status = req.Status
	}
	if err := s.routerRepo.Update(router); err != nil {
		return nil, err
	}
	return router, nil
}

func (s *routerService) Delete(id uuid.UUID) error {
	return s.routerRepo.Delete(id)
}

func (s *routerService) TestConnection(id uuid.UUID) error {
	router, err := s.routerRepo.FindByID(id)
	if err != nil {
		return errors.New("router not found")
	}
	pass, err := utils.Decrypt(router.Password, s.cfg.JWTSecret)
	if err != nil {
		return errors.New("failed to decrypt credentials")
	}
	if err := mikrotik.TestConnection(router.IPAddress, router.APIPort, router.Username, pass); err != nil {
		_ = s.routerRepo.UpdateConnectionStatus(id, "disconnected")
		return err
	}
	_ = s.routerRepo.UpdateConnectionStatus(id, "connected")
	return nil
}

func (s *routerService) TestConnectionRaw(ip string, port int, username, password string) error {
	return mikrotik.TestConnection(ip, port, username, password)
}
