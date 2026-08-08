package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/config"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/pkg/mikrotik"
	"github.com/ispcms/backend/pkg/utils"
	"gorm.io/gorm"
)

// SyncSummary is the aggregate result returned to the caller after syncing all routers.
type SyncSummary struct {
	RoutersProcessed int               `json:"routers_processed"`
	RoutersSucceeded int               `json:"routers_succeeded"`
	RoutersFailed    int               `json:"routers_failed"`
	TotalSecrets     int               `json:"total_secrets"`
	NewAccounts      int               `json:"new_accounts"`
	UpdatedAccounts  int               `json:"updated_accounts"`
	ArchivedAccounts int               `json:"archived_accounts"`
	OnlineUsers      int               `json:"online_users"`
	OfflineUsers     int               `json:"offline_users"`
	DurationMs       int64             `json:"duration_ms"`
	Logs             []*models.SyncLog `json:"logs"`
	Errors           []string          `json:"errors"`
}

type SyncService interface {
	SyncRouter(routerID uuid.UUID) (*models.SyncLog, error)
	SyncAllRouters() (*SyncSummary, []error)
	GetSyncLogs(routerID *uuid.UUID, limit int) ([]models.SyncLog, error)
	GetLastSyncLog(routerID uuid.UUID) (*models.SyncLog, error)
}

type syncService struct {
	routerRepo          repositories.RouterRepository
	pppoeRepo           repositories.PPPoERepository
	internetAccountRepo repositories.InternetAccountRepository
	profileMappingRepo  repositories.ProfileMappingRepository
	billingSvc          BillingService
	notifSvc            NotificationService
	activitySvc         ActivityService
	db                  *gorm.DB
	cfg                 *config.Config
}

func NewSyncService(
	routerRepo repositories.RouterRepository,
	pppoeRepo repositories.PPPoERepository,
	internetAccountRepo repositories.InternetAccountRepository,
	profileMappingRepo repositories.ProfileMappingRepository,
	billingSvc BillingService,
	notifSvc NotificationService,
	activitySvc ActivityService,
	db *gorm.DB,
	cfg *config.Config,
) SyncService {
	return &syncService{
		routerRepo:          routerRepo,
		pppoeRepo:           pppoeRepo,
		internetAccountRepo: internetAccountRepo,
		profileMappingRepo:  profileMappingRepo,
		billingSvc:          billingSvc,
		notifSvc:            notifSvc,
		activitySvc:         activitySvc,
		db:                  db,
		cfg:                 cfg,
	}
}

func (s *syncService) SyncRouter(routerID uuid.UUID) (*models.SyncLog, error) {
	router, err := s.routerRepo.FindByID(routerID)
	if err != nil {
		return nil, fmt.Errorf("router not found: %w", err)
	}

	log := &models.SyncLog{
		RouterID:  routerID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := s.db.Create(log).Error; err != nil {
		return nil, err
	}

	startTime := time.Now()
	syncErr := s.doSync(router, log)
	log.Duration = time.Since(startTime).Milliseconds()

	now := time.Now()
	log.CompletedAt = &now
	if syncErr != nil {
		log.Status = "failed"
		log.ErrorMessage = syncErr.Error()
		_ = s.routerRepo.UpdateConnectionStatus(routerID, "disconnected")
	} else {
		log.Status = "success"
		_ = s.routerRepo.UpdateConnectionStatus(routerID, "connected")
		_ = s.routerRepo.UpdateSyncTime(routerID)
		if s.activitySvc != nil {
			desc := fmt.Sprintf("Router: %s | Secrets: %d | New: %d | Updated: %d | Duration: %dms",
				router.Name, log.SecretsTotal, log.NewAccounts, log.UpdatedAccounts, log.Duration)
			s.activitySvc.Log(nil, "sync", "sync_completed", "Router Sync Completed", desc, "sync_log", log.ID.String())
		}
	}
	s.db.Omit("Router").Save(log)
	return log, syncErr
}

func (s *syncService) SyncAllRouters() (*SyncSummary, []error) {
	var routers []models.Router
	s.db.Where("status = ?", "active").Find(&routers)

	summary := &SyncSummary{}
	var errs []error
	start := time.Now()

	for _, r := range routers {
		summary.RoutersProcessed++
		log, err := s.SyncRouter(r.ID)
		if log != nil {
			summary.Logs = append(summary.Logs, log)
			summary.TotalSecrets += log.SecretsTotal
			summary.NewAccounts += log.NewAccounts
			summary.UpdatedAccounts += log.UpdatedAccounts
			summary.ArchivedAccounts += log.ArchivedAccounts
			summary.OnlineUsers += log.OnlineCount
			summary.OfflineUsers += log.OfflineCount
		}
		if err != nil {
			summary.RoutersFailed++
			errs = append(errs, fmt.Errorf("router %s: %w", r.Name, err))
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %s", r.Name, err.Error()))
		} else {
			summary.RoutersSucceeded++
		}
	}
	summary.DurationMs = time.Since(start).Milliseconds()
	return summary, errs
}

func (s *syncService) doSync(router *models.Router, log *models.SyncLog) error {
	pass, err := utils.Decrypt(router.Password, s.cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("decrypting credentials: %w", err)
	}

	client, err := mikrotik.NewClient(router.IPAddress, router.APIPort, router.Username, pass)
	if err != nil {
		return fmt.Errorf("connecting to router: %w", err)
	}
	defer client.Close()

	// ── Step 1: Fetch PPPoE Secrets ───────────────────────────────────────────
	rawSecrets, err := client.GetPPPoESecrets()
	if err != nil {
		return fmt.Errorf("reading PPPoE secrets: %w", err)
	}

	now := time.Now()

	// Build internet_account rows and track active usernames for archive detection.
	internetAccounts := make([]models.InternetAccount, 0, len(rawSecrets))
	activeUsernames := make([]string, 0, len(rawSecrets))
	activeMikrotikIDs := make([]string, 0, len(rawSecrets))

	// Build legacy pppoe_secrets rows in parallel.
	legacySecrets := make([]models.PPPoESecret, 0, len(rawSecrets))

	for _, rs := range rawSecrets {
		activeUsernames = append(activeUsernames, rs.Username)
		activeMikrotikIDs = append(activeMikrotikIDs, rs.RouterOSID)

		legacySecrets = append(legacySecrets, models.PPPoESecret{
			RouterID:      router.ID,
			RouterOSID:    rs.RouterOSID,
			Username:      rs.Username,
			Password:      rs.Password,
			Profile:       rs.Profile,
			Service:       rs.Service,
			LocalAddress:  rs.LocalAddress,
			RemoteAddress: rs.RemoteAddress,
			CallerID:      rs.CallerID,
			Disabled:      rs.Disabled,
			Comment:       rs.Comment,
			SyncTime:      now,
		})

		internetAccounts = append(internetAccounts, models.InternetAccount{
			RouterID:         router.ID,
			MikrotikSecretID: rs.RouterOSID,
			Username:         rs.Username,
			Password:         rs.Password,
			Service:          rs.Service,
			Profile:          rs.Profile,
			LocalAddress:     rs.LocalAddress,
			RemoteAddress:    rs.RemoteAddress,
			CallerID:         rs.CallerID,
			Comment:          rs.Comment,
			Disabled:         rs.Disabled,
		})
	}

	log.SecretsTotal = len(rawSecrets)

	// ── Step 2: Upsert into internet_accounts ─────────────────────────────────
	// Conflict key: (router_id, username) — guarantees no duplicates.
	iaCreated, iaUpdated, err := s.internetAccountRepo.BulkUpsert(internetAccounts)
	if err != nil {
		return fmt.Errorf("upserting internet accounts: %w", err)
	}
	log.NewAccounts = iaCreated
	log.UpdatedAccounts = iaUpdated
	log.SecretsCreated = iaCreated // legacy field
	log.SecretsUpdated = iaUpdated // legacy field

	// ── Step 3: Archive accounts no longer on the router ──────────────────────
	iaArchived, err := s.internetAccountRepo.MarkArchived(router.ID, activeUsernames)
	if err != nil {
		return fmt.Errorf("archiving missing internet accounts: %w", err)
	}
	log.ArchivedAccounts = iaArchived
	log.SecretsDeleted = iaArchived // legacy field

	// ── Step 4: Upsert into legacy pppoe_secrets table ────────────────────────
	_, _, err = s.pppoeRepo.UpsertSecrets(router.ID, legacySecrets)
	if err != nil {
		return fmt.Errorf("upserting pppoe secrets: %w", err)
	}
	_, err = s.pppoeRepo.MarkDeletedSecrets(router.ID, activeMikrotikIDs)
	if err != nil {
		return fmt.Errorf("marking deleted pppoe secrets: %w", err)
	}

	// ── Step 5: Profile mapping & subscription assignment ─────────────────────
	// Reload accounts with real UUIDs after upsert.
	var dbAccounts []models.InternetAccount
	s.db.Where("router_id = ? AND archived_at IS NULL", router.ID).Find(&dbAccounts)
	accountByUsername := make(map[string]*models.InternetAccount, len(dbAccounts))
	for i := range dbAccounts {
		accountByUsername[dbAccounts[i].Username] = &dbAccounts[i]
	}

	allMappings, _ := s.profileMappingRepo.ListAll()
	mappingByProfile := make(map[string]*models.ProfileMapping, len(allMappings))
	for i := range allMappings {
		mappingByProfile[allMappings[i].MikrotikProfile] = &allMappings[i]
	}

	notifiedProfiles := map[string]bool{}
	for _, acc := range internetAccounts {
		dbAcc, found := accountByUsername[acc.Username]
		if !found {
			continue
		}
		profile := acc.Profile
		mappingStatus := "ok"
		if profile == "" {
			mappingStatus = "no_profile"
		} else if mapping, ok := mappingByProfile[profile]; ok {
			if s.billingSvc != nil {
				_, _ = s.billingSvc.AssignSubscriptionFromSync(dbAcc.ID, &mapping.Package)
			}
		} else {
			mappingStatus = "mapping_required"
			if !notifiedProfiles[profile] && s.notifSvc != nil {
				_ = s.notifSvc.NotifyPackageMappingMissing(profile)
				notifiedProfiles[profile] = true
			}
		}
		s.db.Model(&models.InternetAccount{}).
			Where("id = ?", dbAcc.ID).
			Update("package_mapping_status", mappingStatus)
	}

	// ── Step 6: Fetch and apply active sessions ────────────────────────────────
	rawSessions, err := client.GetActiveSessions()
	if err != nil {
		return fmt.Errorf("reading active sessions: %w", err)
	}

	// Reset all session fields for this router first.
	_ = s.internetAccountRepo.ClearSessionInfo(router.ID)
	_ = s.pppoeRepo.ClearSessions(router.ID)

	legacySessions := make([]models.PPPoEActiveSession, 0, len(rawSessions))
	for _, rs := range rawSessions {
		legacySessions = append(legacySessions, models.PPPoEActiveSession{
			RouterID:   router.ID,
			RouterOSID: rs.RouterOSID,
			Username:   rs.Username,
			CurrentIP:  rs.CurrentIP,
			Uptime:     rs.Uptime,
			SessionID:  rs.SessionID,
			SyncTime:   now,
		})
		_ = s.internetAccountRepo.UpdateSessionInfo(
			router.ID,
			rs.Username,
			rs.CurrentIP,
			rs.SessionID,
			rs.Uptime,
			rs.ConnectedSince,
		)
	}
	if err := s.pppoeRepo.UpsertSessions(router.ID, legacySessions); err != nil {
		return fmt.Errorf("upserting pppoe sessions: %w", err)
	}

	log.SessionsTotal = len(rawSessions)

	// ── Step 7: Count online / offline for this router ────────────────────────
	var onlineCount, offlineCount int64
	s.db.Model(&models.InternetAccount{}).
		Where("router_id = ? AND archived_at IS NULL AND is_online = ?", router.ID, true).
		Count(&onlineCount)
	s.db.Model(&models.InternetAccount{}).
		Where("router_id = ? AND archived_at IS NULL AND is_online = ?", router.ID, false).
		Count(&offlineCount)
	log.OnlineCount = int(onlineCount)
	log.OfflineCount = int(offlineCount)

	return nil
}

func (s *syncService) GetSyncLogs(routerID *uuid.UUID, limit int) ([]models.SyncLog, error) {
	var logs []models.SyncLog
	q := s.db.Model(&models.SyncLog{}).Preload("Router").Order("started_at DESC")
	if routerID != nil {
		q = q.Where("router_id = ?", routerID)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&logs).Error
	return logs, err
}

func (s *syncService) GetLastSyncLog(routerID uuid.UUID) (*models.SyncLog, error) {
	var log models.SyncLog
	err := s.db.Where("router_id = ?", routerID).Order("started_at DESC").First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}
