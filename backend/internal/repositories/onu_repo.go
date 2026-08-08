package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type ONUFilter struct {
	OLTID     *uuid.UUID
	PONPortID *uuid.UUID
	Status    string
	Search    string // MAC, serial, onuID
	Unlinked  bool   // true = internet_account_id IS NULL
	Page      int
	PageSize  int
}

type ONURepository interface {
	List(f ONUFilter) ([]models.ONU, int64, error)
	FindByID(id uuid.UUID) (*models.ONU, error)
	FindByOLTAndSlot(oltID uuid.UUID, portIndex, onuSlot int) (*models.ONU, error)
	Upsert(o *models.ONU) (bool, error) // returns isNew
	ArchiveMissing(oltID uuid.UUID, activeKeys [][2]int) (int64, error)
	LinkToAccount(id uuid.UUID, accountID *uuid.UUID) error
	CountByOLTAndPort(oltID, portID uuid.UUID) (int64, error)
}

type onuRepository struct{ db *gorm.DB }

func NewONURepository(db *gorm.DB) ONURepository {
	return &onuRepository{db: db}
}

func (r *onuRepository) List(f ONUFilter) ([]models.ONU, int64, error) {
	q := r.db.Where("archived_at IS NULL").
		Preload("OLT").
		Preload("PONPort").
		Preload("InternetAccount")

	if f.OLTID != nil {
		q = q.Where("olt_id = ?", f.OLTID)
	}
	if f.PONPortID != nil {
		q = q.Where("pon_port_id = ?", f.PONPortID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Unlinked {
		q = q.Where("internet_account_id IS NULL")
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("mac_address ILIKE ? OR serial_number ILIKE ? OR onu_id ILIKE ?", like, like, like)
	}

	var total int64
	q.Model(&models.ONU{}).Count(&total)

	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var onus []models.ONU
	err := q.Order("olt_id, port_index, onu_slot").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&onus).Error
	return onus, total, err
}

func (r *onuRepository) FindByID(id uuid.UUID) (*models.ONU, error) {
	var o models.ONU
	err := r.db.Preload("OLT").Preload("PONPort").Preload("InternetAccount").
		First(&o, "id = ?", id).Error
	return &o, err
}

func (r *onuRepository) FindByOLTAndSlot(oltID uuid.UUID, portIndex, onuSlot int) (*models.ONU, error) {
	var o models.ONU
	err := r.db.Where("olt_id = ? AND port_index = ? AND onu_slot = ?", oltID, portIndex, onuSlot).
		First(&o).Error
	return &o, err
}

func (r *onuRepository) Upsert(o *models.ONU) (bool, error) {
	existing, err := r.FindByOLTAndSlot(o.OLTID, o.PortIndex, o.OnuSlot)
	if err != nil {
		// new ONU
		if o.ID == uuid.Nil {
			o.ID = uuid.New()
		}
		return true, r.db.Create(o).Error
	}
	// update existing
	existing.OnuID = o.OnuID
	existing.MACAddress = o.MACAddress
	existing.SerialNumber = o.SerialNumber
	existing.Vendor = o.Vendor
	existing.Model = o.Model
	existing.Status = o.Status
	existing.RegStatus = o.RegStatus
	existing.RXPower = o.RXPower
	existing.TXPower = o.TXPower
	existing.Distance = o.Distance
	existing.ArchivedAt = nil
	if o.Status == "online" {
		now := o.UpdatedAt
		existing.LastOnlineAt = &now
	}
	return false, r.db.Save(existing).Error
}

func (r *onuRepository) ArchiveMissing(oltID uuid.UUID, activeKeys [][2]int) (int64, error) {
	if len(activeKeys) == 0 {
		result := r.db.Model(&models.ONU{}).
			Where("olt_id = ? AND archived_at IS NULL", oltID).
			Update("archived_at", "NOW()")
		return result.RowsAffected, result.Error
	}
	// Build (port_index, onu_slot) pairs
	type pair struct{ P, S int }
	seen := make([]pair, 0, len(activeKeys))
	for _, k := range activeKeys {
		seen = append(seen, pair{k[0], k[1]})
	}
	// Archive any ONU whose (port_index, onu_slot) is NOT in the active set
	var ids []uuid.UUID
	r.db.Model(&models.ONU{}).
		Where("olt_id = ? AND archived_at IS NULL", oltID).
		Pluck("id", &ids)

	var toArchive []uuid.UUID
	for _, id := range ids {
		var o models.ONU
		r.db.Select("port_index, onu_slot").First(&o, "id = ?", id)
		found := false
		for _, s := range seen {
			if s.P == o.PortIndex && s.S == o.OnuSlot {
				found = true
				break
			}
		}
		if !found {
			toArchive = append(toArchive, id)
		}
	}
	if len(toArchive) == 0 {
		return 0, nil
	}
	result := r.db.Model(&models.ONU{}).Where("id IN ?", toArchive).Update("archived_at", "NOW()")
	return result.RowsAffected, result.Error
}

func (r *onuRepository) LinkToAccount(id uuid.UUID, accountID *uuid.UUID) error {
	return r.db.Model(&models.ONU{}).Where("id = ?", id).Update("internet_account_id", accountID).Error
}

func (r *onuRepository) CountByOLTAndPort(oltID, portID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.ONU{}).
		Where("olt_id = ? AND pon_port_id = ? AND archived_at IS NULL", oltID, portID).
		Count(&count).Error
	return count, err
}
