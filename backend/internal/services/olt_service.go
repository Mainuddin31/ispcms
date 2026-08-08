package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/pkg/utils"
)

// ── SNMP Profile Service ──────────────────────────────────────────────────────

type CreateSNMPProfileInput struct {
	Name        string
	Vendor      string
	Technology  string
	OIDMap      map[string]string
	Description string
}

type SNMPProfileService interface {
	List() ([]models.SNMPProfile, error)
	Get(id uuid.UUID) (*models.SNMPProfile, error)
	Create(p *models.SNMPProfile) error
	CreateFromInput(input *CreateSNMPProfileInput) (*models.SNMPProfile, error)
	UpdateFromInput(id uuid.UUID, input *CreateSNMPProfileInput) (*models.SNMPProfile, error)
	Update(id uuid.UUID, p *models.SNMPProfile) error
	Delete(id uuid.UUID) error
}

type snmpProfileService struct{ repo repositories.SNMPProfileRepository }

func NewSNMPProfileService(repo repositories.SNMPProfileRepository) SNMPProfileService {
	return &snmpProfileService{repo: repo}
}
func (s *snmpProfileService) List() ([]models.SNMPProfile, error)              { return s.repo.List() }
func (s *snmpProfileService) Get(id uuid.UUID) (*models.SNMPProfile, error)    { return s.repo.FindByID(id) }
func (s *snmpProfileService) Create(p *models.SNMPProfile) error               { return s.repo.Create(p) }
func (s *snmpProfileService) Update(id uuid.UUID, p *models.SNMPProfile) error {
	p.ID = id
	return s.repo.Update(p)
}
func (s *snmpProfileService) Delete(id uuid.UUID) error { return s.repo.Delete(id) }

func (s *snmpProfileService) CreateFromInput(input *CreateSNMPProfileInput) (*models.SNMPProfile, error) {
	if input.Name == "" || input.Vendor == "" {
		return nil, fmt.Errorf("name and vendor are required")
	}
	p := &models.SNMPProfile{
		Name:        input.Name,
		Vendor:      input.Vendor,
		Technology:  input.Technology,
		OIDMap:      models.OIDMap(input.OIDMap),
		Description: input.Description,
	}
	return p, s.repo.Create(p)
}

func (s *snmpProfileService) UpdateFromInput(id uuid.UUID, input *CreateSNMPProfileInput) (*models.SNMPProfile, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	p.Name = input.Name
	p.Vendor = input.Vendor
	p.Technology = input.Technology
	p.OIDMap = models.OIDMap(input.OIDMap)
	p.Description = input.Description
	return p, s.repo.Update(p)
}

// ── OLT Service ───────────────────────────────────────────────────────────────

type CreateOLTInput struct {
	Name           string
	Vendor         string
	Model          string
	SNMPProfileID  uuid.UUID
	ManagementIP   string
	SNMPVersion    string
	SNMPPort       int
	Timeout        int
	Retries        int
	Community      string
	V3Username     string
	V3AuthProtocol string
	V3AuthPassword string
	V3PrivProtocol string
	V3PrivPassword string
	POP            string
	Rack           string
	Cabinet        string
	Description    string
	Status         string
	SyncInterval   int
}

type OLTService interface {
	List(f repositories.OLTFilter) ([]models.OLT, error)
	Get(id uuid.UUID) (*models.OLT, error)
	Create(input CreateOLTInput, jwtSecret string) (*models.OLT, error)
	Update(id uuid.UUID, input CreateOLTInput, jwtSecret string) (*models.OLT, error)
	Delete(id uuid.UUID, deletedBy uuid.UUID) error
	Stats() (*models.OLTStats, error)
	GetSyncLogs(oltID uuid.UUID, limit int) ([]models.OLTSyncLog, error)
	RecentSyncLogs(limit int) ([]models.OLTSyncLog, error)
	ListPONPorts(oltID uuid.UUID) ([]models.PONPort, error)
}

type oltService struct {
	oltRepo     repositories.OLTRepository
	portRepo    repositories.PONPortRepository
	syncLogRepo repositories.OLTSyncLogRepository
}

func NewOLTService(
	oltRepo repositories.OLTRepository,
	portRepo repositories.PONPortRepository,
	syncLogRepo repositories.OLTSyncLogRepository,
) OLTService {
	return &oltService{oltRepo: oltRepo, portRepo: portRepo, syncLogRepo: syncLogRepo}
}

func (s *oltService) List(f repositories.OLTFilter) ([]models.OLT, error) { return s.oltRepo.List(f) }
func (s *oltService) Get(id uuid.UUID) (*models.OLT, error)               { return s.oltRepo.FindByID(id) }
func (s *oltService) Stats() (*models.OLTStats, error)                    { return s.oltRepo.Stats() }

func (s *oltService) Create(input CreateOLTInput, jwtSecret string) (*models.OLT, error) {
	if input.Name == "" || input.ManagementIP == "" {
		return nil, fmt.Errorf("name and management_ip are required")
	}
	if input.SNMPProfileID == uuid.Nil {
		return nil, fmt.Errorf("snmp_profile_id is required")
	}
	olt := &models.OLT{
		Name:           input.Name,
		Vendor:         input.Vendor,
		Model:          input.Model,
		SNMPProfileID:  input.SNMPProfileID,
		ManagementIP:   input.ManagementIP,
		SNMPVersion:    input.SNMPVersion,
		SNMPPort:       input.SNMPPort,
		Timeout:        input.Timeout,
		Retries:        input.Retries,
		Community:      input.Community,
		V3Username:     input.V3Username,
		V3AuthProtocol: input.V3AuthProtocol,
		V3PrivProtocol: input.V3PrivProtocol,
		POP:            input.POP,
		Rack:           input.Rack,
		Cabinet:        input.Cabinet,
		Description:    input.Description,
		Status:         input.Status,
		SyncInterval:   input.SyncInterval,
	}
	if olt.Status == "" {
		olt.Status = "active"
	}
	if input.V3AuthPassword != "" {
		enc, err := utils.Encrypt(input.V3AuthPassword, jwtSecret)
		if err != nil {
			return nil, err
		}
		olt.V3AuthPassword = enc
	}
	if input.V3PrivPassword != "" {
		enc, err := utils.Encrypt(input.V3PrivPassword, jwtSecret)
		if err != nil {
			return nil, err
		}
		olt.V3PrivPassword = enc
	}
	return olt, s.oltRepo.Create(olt)
}

func (s *oltService) Update(id uuid.UUID, input CreateOLTInput, jwtSecret string) (*models.OLT, error) {
	olt, err := s.oltRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	olt.Name = input.Name
	olt.Vendor = input.Vendor
	olt.Model = input.Model
	olt.SNMPProfileID = input.SNMPProfileID
	olt.ManagementIP = input.ManagementIP
	olt.SNMPVersion = input.SNMPVersion
	olt.SNMPPort = input.SNMPPort
	olt.Timeout = input.Timeout
	olt.Retries = input.Retries
	olt.Community = input.Community
	olt.V3Username = input.V3Username
	olt.V3AuthProtocol = input.V3AuthProtocol
	olt.V3PrivProtocol = input.V3PrivProtocol
	olt.POP = input.POP
	olt.Rack = input.Rack
	olt.Cabinet = input.Cabinet
	olt.Description = input.Description
	olt.Status = input.Status
	olt.SyncInterval = input.SyncInterval

	if input.V3AuthPassword != "" && input.V3AuthPassword != "***" {
		enc, _ := utils.Encrypt(input.V3AuthPassword, jwtSecret)
		olt.V3AuthPassword = enc
	}
	if input.V3PrivPassword != "" && input.V3PrivPassword != "***" {
		enc, _ := utils.Encrypt(input.V3PrivPassword, jwtSecret)
		olt.V3PrivPassword = enc
	}
	return olt, s.oltRepo.Update(olt)
}

func (s *oltService) Delete(id uuid.UUID, deletedBy uuid.UUID) error {
	return s.oltRepo.SoftDelete(id, deletedBy)
}

func (s *oltService) GetSyncLogs(oltID uuid.UUID, limit int) ([]models.OLTSyncLog, error) {
	return s.syncLogRepo.ListByOLT(oltID, limit)
}

func (s *oltService) RecentSyncLogs(limit int) ([]models.OLTSyncLog, error) {
	return s.syncLogRepo.ListRecent(limit)
}

func (s *oltService) ListPONPorts(oltID uuid.UUID) ([]models.PONPort, error) {
	return s.portRepo.ListByOLT(oltID)
}

// ── ONU Service ───────────────────────────────────────────────────────────────

type ONUService interface {
	List(f repositories.ONUFilter) ([]models.ONU, int64, error)
	Get(id uuid.UUID) (*models.ONU, error)
	Link(id uuid.UUID, accountID *uuid.UUID) error
}

type onuService struct{ repo repositories.ONURepository }

func NewONUService(repo repositories.ONURepository) ONUService {
	return &onuService{repo: repo}
}

func (s *onuService) List(f repositories.ONUFilter) ([]models.ONU, int64, error) {
	return s.repo.List(f)
}
func (s *onuService) Get(id uuid.UUID) (*models.ONU, error)           { return s.repo.FindByID(id) }
func (s *onuService) Link(id uuid.UUID, accountID *uuid.UUID) error { return s.repo.LinkToAccount(id, accountID) }

// ── OLT Background Scheduler ──────────────────────────────────────────────────

// OLTScheduler periodically syncs OLTs that have a non-zero SyncInterval.
type OLTScheduler struct {
	oltRepo  repositories.OLTRepository
	syncSvc  OLTSyncService
	stopCh   chan struct{}
}

func NewOLTScheduler(oltRepo repositories.OLTRepository, syncSvc OLTSyncService) *OLTScheduler {
	return &OLTScheduler{oltRepo: oltRepo, syncSvc: syncSvc, stopCh: make(chan struct{})}
}

func (sc *OLTScheduler) Start() {
	go sc.run()
}

func (sc *OLTScheduler) Stop() {
	close(sc.stopCh)
}

func (sc *OLTScheduler) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-sc.stopCh:
			return
		case <-ticker.C:
			sc.tick()
		}
	}
}

func (sc *OLTScheduler) tick() {
	olts, err := sc.oltRepo.FindActiveWithInterval()
	if err != nil {
		return
	}
	now := time.Now()
	for _, olt := range olts {
		if olt.SyncInterval <= 0 {
			continue
		}
		due := olt.LastSyncAt == nil ||
			now.Sub(*olt.LastSyncAt) >= time.Duration(olt.SyncInterval)*time.Minute
		if due {
			go func(id uuid.UUID) {
				_, _ = sc.syncSvc.Sync(id)
			}(olt.ID)
		}
	}
}
