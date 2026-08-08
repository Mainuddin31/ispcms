package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InternetAccountFilter struct {
	RouterID        *uuid.UUID
	Profile         string
	Search          string
	IsOnline        *bool
	Disabled        *bool
	Archived        *bool
	// Prefixes restricts results to accounts whose username starts with one of
	// these values. nil = unrestricted (admin/super_admin). empty slice = no results.
	Prefixes        []string
	PrefixRestricted bool // true when prefixes came from a role (even if empty)
}

type InternetAccountRepository interface {
	BulkUpsert(accounts []models.InternetAccount) (created, updated int, err error)
	MarkArchived(routerID uuid.UUID, activeMikrotikIDs []string) (int, error)
	ClearSessionInfo(routerID uuid.UUID) error
	UpdateSessionInfo(routerID uuid.UUID, username string, currentIP, sessionID, uptime string, connectedSince *time.Time) error
	List(filter InternetAccountFilter, page, pageSize int) ([]models.InternetAccount, int64, error)
	GetByID(id uuid.UUID) (*models.InternetAccount, error)
	CountStats(prefixes []string, prefixRestricted bool) (total, enabled, disabled, online, offline, archived int64, err error)
	ListProfiles() ([]string, error)
	// DeleteOrphaned removes rows whose router_id is the nil UUID or references a
	// router that no longer exists in the routers table. Returns count deleted.
	DeleteOrphaned() (int, error)
}

type internetAccountRepository struct{ db *gorm.DB }

func NewInternetAccountRepository(db *gorm.DB) InternetAccountRepository {
	return &internetAccountRepository{db: db}
}

// BulkUpsert inserts or updates internet accounts using (router_id, username) as the
// conflict key. This guarantees idempotency: running sync N times produces the same
// result. Returns (created, updated, error).
func (r *internetAccountRepository) BulkUpsert(accounts []models.InternetAccount) (int, int, error) {
	if len(accounts) == 0 {
		return 0, 0, nil
	}

	usernames := make([]string, len(accounts))
	for i, a := range accounts {
		usernames[i] = a.Username
	}
	routerID := accounts[0].RouterID

	// Count how many rows already exist — those will be updated, the rest inserted.
	var existingCount int64
	r.db.Model(&models.InternetAccount{}).
		Where("router_id = ? AND username IN ?", routerID, usernames).
		Count(&existingCount)

	now := time.Now()
	for i := range accounts {
		accounts[i].LastSyncAt = &now
		accounts[i].SyncStatus = "synced"
		accounts[i].ArchivedAt = nil // un-archive if it was previously missing
	}

	err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "router_id"},
			{Name: "username"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"mikrotik_secret_id", "password", "service", "profile",
			"local_address", "remote_address", "caller_id",
			"comment", "disabled", "sync_status", "last_sync_at",
			"archived_at", "updated_at",
		}),
	}).Create(&accounts).Error
	if err != nil {
		return 0, 0, err
	}

	updated := int(existingCount)
	created := len(accounts) - updated
	if created < 0 {
		created = 0
	}
	return created, updated, nil
}

// MarkArchived archives accounts whose username is no longer present on the router.
// Uses the active username list (not mikrotik_secret_id) to match the new unique key.
func (r *internetAccountRepository) MarkArchived(routerID uuid.UUID, activeUsernames []string) (int, error) {
	now := time.Now()
	q := r.db.Model(&models.InternetAccount{}).
		Where("router_id = ? AND archived_at IS NULL", routerID)
	if len(activeUsernames) > 0 {
		q = q.Where("username NOT IN ?", activeUsernames)
	}
	result := q.Updates(map[string]interface{}{
		"archived_at": now,
		"sync_status": "archived",
		"is_online":   false,
		"updated_at":  now,
	})
	return int(result.RowsAffected), result.Error
}

func (r *internetAccountRepository) ClearSessionInfo(routerID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.InternetAccount{}).
		Where("router_id = ? AND is_online = ?", routerID, true).
		Updates(map[string]interface{}{
			"is_online":       false,
			"current_ip":      "",
			"session_id":      "",
			"uptime":          "",
			"connected_since": nil,
			"updated_at":      now,
		}).Error
}

func (r *internetAccountRepository) UpdateSessionInfo(
	routerID uuid.UUID,
	username string,
	currentIP, sessionID, uptime string,
	connectedSince *time.Time,
) error {
	now := time.Now()
	return r.db.Model(&models.InternetAccount{}).
		Where("router_id = ? AND username = ? AND archived_at IS NULL", routerID, username).
		Updates(map[string]interface{}{
			"is_online":       true,
			"current_ip":      currentIP,
			"session_id":      sessionID,
			"uptime":          uptime,
			"connected_since": connectedSince,
			"last_seen":       now,
			"updated_at":      now,
		}).Error
}

func (r *internetAccountRepository) List(filter InternetAccountFilter, page, pageSize int) ([]models.InternetAccount, int64, error) {
	var accounts []models.InternetAccount
	var total int64

	q := r.db.Model(&models.InternetAccount{}).Preload("Router").Preload("ONU").Preload("ONU.OLT")

	if filter.Archived != nil && *filter.Archived {
		q = q.Where("archived_at IS NOT NULL")
	} else {
		q = q.Where("archived_at IS NULL")
	}

	if filter.RouterID != nil {
		q = q.Where("router_id = ?", filter.RouterID)
	}
	if filter.Profile != "" {
		q = q.Where("profile = ?", filter.Profile)
	}
	if filter.IsOnline != nil {
		q = q.Where("is_online = ?", *filter.IsOnline)
	}
	if filter.Disabled != nil {
		q = q.Where("disabled = ?", *filter.Disabled)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("username ILIKE ? OR comment ILIKE ? OR caller_id ILIKE ?", like, like, like)
	}

	// Prefix-based scoping: restricted users only see accounts whose username
	// starts with one of their role's allowed prefixes.
	if filter.PrefixRestricted {
		if len(filter.Prefixes) == 0 {
			// No prefixes configured → no accounts visible.
			return []models.InternetAccount{}, 0, nil
		}
		// Build: username ILIKE 'p1%' OR username ILIKE 'p2%' …
		sub := r.db.Where("username ILIKE ?", filter.Prefixes[0]+"%")
		for _, p := range filter.Prefixes[1:] {
			sub = sub.Or("username ILIKE ?", p+"%")
		}
		q = q.Where(sub)
	}

	q.Count(&total)
	err := q.Order("username ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&accounts).Error
	return accounts, total, err
}

func (r *internetAccountRepository) GetByID(id uuid.UUID) (*models.InternetAccount, error) {
	var a models.InternetAccount
	err := r.db.Preload("Router").Preload("ONU").Preload("ONU.OLT").First(&a, "id = ?", id).Error
	return &a, err
}

func (r *internetAccountRepository) CountStats(prefixes []string, prefixRestricted bool) (total, enabled, disabled, online, offline, archived int64, err error) {
	applyPrefix := func(q *gorm.DB) *gorm.DB {
		if !prefixRestricted || len(prefixes) == 0 {
			return q
		}
		sub := r.db.Where("username ILIKE ?", prefixes[0]+"%")
		for _, p := range prefixes[1:] {
			sub = sub.Or("username ILIKE ?", p+"%")
		}
		return q.Where(sub)
	}
	if prefixRestricted && len(prefixes) == 0 {
		return 0, 0, 0, 0, 0, 0, nil
	}
	applyPrefix(r.db.Model(&models.InternetAccount{}).Where("archived_at IS NULL")).Count(&total)
	applyPrefix(r.db.Model(&models.InternetAccount{}).Where("archived_at IS NULL AND disabled = ?", false)).Count(&enabled)
	applyPrefix(r.db.Model(&models.InternetAccount{}).Where("archived_at IS NULL AND disabled = ?", true)).Count(&disabled)
	applyPrefix(r.db.Model(&models.InternetAccount{}).Where("archived_at IS NULL AND is_online = ?", true)).Count(&online)
	applyPrefix(r.db.Model(&models.InternetAccount{}).Where("archived_at IS NULL AND is_online = ?", false)).Count(&offline)
	applyPrefix(r.db.Model(&models.InternetAccount{}).Where("archived_at IS NOT NULL")).Count(&archived)
	return
}

func (r *internetAccountRepository) DeleteOrphaned() (int, error) {
	result := r.db.Exec(`
		DELETE FROM internet_accounts
		WHERE router_id = '00000000-0000-0000-0000-000000000000'
		   OR router_id NOT IN (SELECT id FROM routers)
	`)
	return int(result.RowsAffected), result.Error
}

func (r *internetAccountRepository) ListProfiles() ([]string, error) {
	var profiles []string
	err := r.db.Model(&models.InternetAccount{}).
		Where("archived_at IS NULL AND profile != ''").
		Distinct("profile").
		Order("profile ASC").
		Pluck("profile", &profiles).Error
	return profiles, err
}
