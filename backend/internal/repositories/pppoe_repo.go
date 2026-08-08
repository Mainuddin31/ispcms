package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type PPPoERepository interface {
	UpsertSecrets(routerID uuid.UUID, secrets []models.PPPoESecret) (created, updated int, err error)
	MarkDeletedSecrets(routerID uuid.UUID, activeRouterOSIDs []string) (int, error)
	FindSecrets(routerID *uuid.UUID, page, pageSize int, search string, disabled *bool) ([]models.PPPoESecret, int64, error)
	FindSecretByID(id uuid.UUID) (*models.PPPoESecret, error)
	UpsertSessions(routerID uuid.UUID, sessions []models.PPPoEActiveSession) error
	ClearSessions(routerID uuid.UUID) error
	FindSessions(routerID *uuid.UUID) ([]models.PPPoEActiveSession, error)
	CountSecrets() (total, active, disabled int64, err error)
	CountSessions() (int64, error)
}

type pppoeRepository struct{ db *gorm.DB }

func NewPPPoERepository(db *gorm.DB) PPPoERepository { return &pppoeRepository{db: db} }

func (r *pppoeRepository) UpsertSecrets(routerID uuid.UUID, secrets []models.PPPoESecret) (int, int, error) {
	created, updated := 0, 0
	for _, s := range secrets {
		var existing models.PPPoESecret
		err := r.db.Where("router_id = ? AND routeros_id = ?", routerID, s.RouterOSID).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			s.RouterID = routerID
			if e := r.db.Create(&s).Error; e != nil {
				return created, updated, e
			}
			created++
		} else if err == nil {
			r.db.Model(&existing).Updates(map[string]interface{}{
				"username":       s.Username,
				"password":       s.Password,
				"profile":        s.Profile,
				"service":        s.Service,
				"local_address":  s.LocalAddress,
				"remote_address": s.RemoteAddress,
				"caller_id":      s.CallerID,
				"disabled":       s.Disabled,
				"comment":        s.Comment,
				"sync_time":      s.SyncTime,
			})
			updated++
		} else {
			return created, updated, err
		}
	}
	return created, updated, nil
}

func (r *pppoeRepository) MarkDeletedSecrets(routerID uuid.UUID, activeIDs []string) (int, error) {
	if len(activeIDs) == 0 {
		return 0, nil
	}
	result := r.db.Model(&models.PPPoESecret{}).
		Where("router_id = ? AND routeros_id NOT IN ?", routerID, activeIDs).
		Update("deleted_at", time.Now())
	return int(result.RowsAffected), result.Error
}

func (r *pppoeRepository) FindSecrets(routerID *uuid.UUID, page, pageSize int, search string, disabled *bool) ([]models.PPPoESecret, int64, error) {
	var secrets []models.PPPoESecret
	var total int64
	q := r.db.Model(&models.PPPoESecret{}).Preload("Router")
	if routerID != nil {
		q = q.Where("router_id = ?", routerID)
	}
	if disabled != nil {
		q = q.Where("disabled = ?", disabled)
	}
	if search != "" {
		q = q.Where("username ILIKE ? OR comment ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Find(&secrets).Error
	return secrets, total, err
}

func (r *pppoeRepository) FindSecretByID(id uuid.UUID) (*models.PPPoESecret, error) {
	var s models.PPPoESecret
	err := r.db.Preload("Router").First(&s, "id = ?", id).Error
	return &s, err
}

func (r *pppoeRepository) UpsertSessions(routerID uuid.UUID, sessions []models.PPPoEActiveSession) error {
	if len(sessions) == 0 {
		return nil
	}
	// ClearSessions already deleted all rows for this router, so plain insert is safe.
	return r.db.Create(&sessions).Error
}

func (r *pppoeRepository) ClearSessions(routerID uuid.UUID) error {
	return r.db.Where("router_id = ?", routerID).Delete(&models.PPPoEActiveSession{}).Error
}

func (r *pppoeRepository) FindSessions(routerID *uuid.UUID) ([]models.PPPoEActiveSession, error) {
	var sessions []models.PPPoEActiveSession
	q := r.db.Model(&models.PPPoEActiveSession{})
	if routerID != nil {
		q = q.Where("router_id = ?", routerID)
	}
	err := q.Find(&sessions).Error
	return sessions, err
}

func (r *pppoeRepository) CountSecrets() (total, active, disabled int64, err error) {
	r.db.Model(&models.PPPoESecret{}).Count(&total)
	r.db.Model(&models.PPPoESecret{}).Where("disabled = ?", false).Count(&active)
	r.db.Model(&models.PPPoESecret{}).Where("disabled = ?", true).Count(&disabled)
	return
}

func (r *pppoeRepository) CountSessions() (int64, error) {
	var count int64
	err := r.db.Model(&models.PPPoEActiveSession{}).Count(&count).Error
	return count, err
}
