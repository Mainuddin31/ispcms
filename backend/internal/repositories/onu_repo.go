package repositories

import (
	"strings"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

// MACTableEntry is one entry from the OLT bridge MAC address table.
// MAC is the customer CPE MAC address (any format); PortIdx and ONUSlot
// identify which ONU the customer device is reachable through.
type MACTableEntry struct {
	MAC     string // customer CPE MAC (any format — normalized internally)
	PortIdx int    // EPON port index (1-based)
	ONUSlot int    // ONU slot within the port
}

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
	// AutoLinkByMAC matches unlinked ONUs to internet accounts by comparing
	// the ONU MAC address against the PPPoE caller_id (normalized hex-only).
	// When oltID is non-nil only ONUs on that OLT are considered.
	// Returns the number of ONUs that were newly linked.
	AutoLinkByMAC(oltID *uuid.UUID) (int64, error)
	// LinkFromMACTable links ONUs using data from the OLT bridge MAC address table.
	// Each entry maps a customer CPE MAC (caller_id) to the ONU it is reachable
	// through (identified by portIdx + onuSlot). Only unlinked ONUs are updated.
	// Returns the number of new links created.
	LinkFromMACTable(oltID uuid.UUID, entries []MACTableEntry) (int64, error)
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

// normalizeMACHex strips all non-hex characters and lowercases a MAC address,
// returning a 12-character bare hex string (e.g. "1c61b462ed3d").
func normalizeMACHex(mac string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(mac) {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// LinkFromMACTable links ONUs to internet accounts using data from the OLT
// bridge MAC address table. Each entry contains a customer CPE MAC address
// (what PPPoE caller_id stores) and the ONU (portIdx, onuSlot) the customer
// device was learned on.
//
// For each entry:
//  1. Find the internet account whose caller_id normalizes to the same hex.
//  2. Find the ONU by (oltID, portIdx, onuSlot).
//  3. If the ONU has no existing link, set internet_account_id.
func (r *onuRepository) LinkFromMACTable(oltID uuid.UUID, entries []MACTableEntry) (int64, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	var linked int64
	for _, e := range entries {
		normalized := normalizeMACHex(e.MAC)
		if len(normalized) != 12 {
			continue // invalid MAC
		}

		// Find internet account by caller_id
		// Scan into string first — GORM Raw().Scan() cannot convert a UUID
		// string directly into [16]byte (uuid.UUID).
		var accountIDStr string
		if err := r.db.Raw(`
			SELECT id FROM internet_accounts
			WHERE LOWER(REGEXP_REPLACE(caller_id, '[^0-9a-fA-F]', '', 'g')) = ?
			  AND caller_id <> ''
			  AND archived_at IS NULL
			LIMIT 1`, normalized).Scan(&accountIDStr).Error; err != nil || accountIDStr == "" {
			continue
		}
		accountID, err := uuid.Parse(accountIDStr)
		if err != nil || accountID == uuid.Nil {
			continue
		}

		// Link ONU if not already linked
		result := r.db.Model(&models.ONU{}).
			Where("olt_id = ? AND port_index = ? AND onu_slot = ? AND internet_account_id IS NULL AND archived_at IS NULL",
				oltID, e.PortIdx, e.ONUSlot).
			Updates(map[string]interface{}{
				"internet_account_id": accountID,
				"updated_at":          "NOW()",
			})
		linked += result.RowsAffected
	}
	return linked, nil
}

// AutoLinkByMAC links unlinked ONUs to internet accounts by matching their
// MAC address against the PPPoE caller_id.
//
// Both sides are normalized to bare lowercase hex before comparison so that
// formats like "aa:bb:cc:dd:ee:ff", "aa-bb-cc-dd-ee-ff", and "aabb.ccdd.eeff"
// all match each other.
//
// Only ONUs with no existing link (internet_account_id IS NULL) are updated;
// manually assigned links are never overwritten.
func (r *onuRepository) AutoLinkByMAC(oltID *uuid.UUID) (int64, error) {
	oltFilter := ""
	var args []interface{}
	if oltID != nil {
		oltFilter = "AND onus.olt_id = ?"
		args = append(args, *oltID)
	}

	sql := `
		UPDATE onus
		SET internet_account_id = ia.id,
		    updated_at = NOW()
		FROM internet_accounts ia
		WHERE onus.archived_at IS NULL
		  AND onus.internet_account_id IS NULL
		  AND ia.caller_id <> ''
		  AND ia.archived_at IS NULL
		  AND LOWER(REGEXP_REPLACE(onus.mac_address, '[^0-9a-fA-F]', '', 'g'))
		      = LOWER(REGEXP_REPLACE(ia.caller_id, '[^0-9a-fA-F]', '', 'g'))
		  ` + oltFilter

	result := r.db.Exec(sql, args...)
	return result.RowsAffected, result.Error
}
