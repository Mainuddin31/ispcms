package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── OIDMap ────────────────────────────────────────────────────────────────────
// Stores vendor-specific OID mappings as a JSON column.
// Standard keys used by the sync engine:
//
//	sys_name, sys_descr
//	pon_port_table      – base OID for PON port table walk
//	onu_mac             – base OID for ONU MAC walk
//	onu_status          – base OID for ONU operational status walk
//	onu_rx_power        – base OID for RX optical power walk
//	onu_tx_power        – base OID for TX optical power walk
//	onu_distance        – base OID for distance walk
//	onu_serial          – base OID for serial number walk
//	onu_model           – base OID for ONU model/type walk
//	index_port_pos      – 0-based position in the OID suffix that holds the port number (default "0")
//	index_onu_pos       – 0-based position in the OID suffix that holds the ONU slot  (default "1")
//	power_divisor       – divisor to convert raw integer to dBm (e.g. "10" means raw/10)
//	distance_unit       – "m" or "cm"
type OIDMap map[string]string

func (m OIDMap) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

func (m *OIDMap) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("OIDMap: cannot scan type %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// ── SNMPProfile ───────────────────────────────────────────────────────────────

type SNMPProfile struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Vendor      string    `gorm:"type:varchar(100);not null" json:"vendor"`
	Technology  string    `gorm:"type:varchar(20);not null" json:"technology"` // EPON | GPON
	OIDMap      OIDMap    `gorm:"type:text;not null" json:"oid_map"`
	Description string    `gorm:"type:text" json:"description"`
	IsDefault   bool      `gorm:"default:false" json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *SNMPProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// ── OLT ───────────────────────────────────────────────────────────────────────

type OLT struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name string    `gorm:"type:varchar(100);not null" json:"name"`

	// Hardware info
	Vendor string `gorm:"type:varchar(100)" json:"vendor"`
	Model  string `gorm:"type:varchar(100)" json:"model"`

	// SNMP Profile
	SNMPProfileID uuid.UUID    `gorm:"type:uuid;not null" json:"snmp_profile_id"`
	SNMPProfile   *SNMPProfile `gorm:"foreignKey:SNMPProfileID" json:"snmp_profile,omitempty"`

	// Network
	ManagementIP string `gorm:"type:varchar(45);not null" json:"management_ip"`

	// SNMP v2c / v3 common
	SNMPVersion string `gorm:"type:varchar(10);default:'v2c'" json:"snmp_version"` // v2c | v3
	SNMPPort    int    `gorm:"default:161" json:"snmp_port"`
	Timeout     int    `gorm:"default:5" json:"timeout"`   // seconds
	Retries     int    `gorm:"default:3" json:"retries"`

	// v2c
	Community string `gorm:"type:varchar(100)" json:"community"`

	// v3
	V3Username     string `gorm:"type:varchar(100)" json:"v3_username"`
	V3AuthProtocol string `gorm:"type:varchar(10)" json:"v3_auth_protocol"` // MD5 | SHA
	V3AuthPassword string `gorm:"type:text" json:"v3_auth_password"`         // encrypted
	V3PrivProtocol string `gorm:"type:varchar(10)" json:"v3_priv_protocol"` // DES | AES
	V3PrivPassword string `gorm:"type:text" json:"v3_priv_password"`         // encrypted

	// Location
	POP         string `gorm:"type:varchar(100)" json:"pop"`
	Rack        string `gorm:"type:varchar(50)" json:"rack"`
	Cabinet     string `gorm:"type:varchar(50)" json:"cabinet"`
	Description string `gorm:"type:text" json:"description"`

	// Status
	Status string `gorm:"type:varchar(20);default:'active'" json:"status"` // active | maintenance | offline | disabled

	// CLI access (Telnet/SSH) — used when SNMP FDB cannot resolve ONU-level MACs
	CLIProtocol      string `gorm:"type:varchar(10)" json:"cli_protocol"`       // "telnet" | "ssh" | "" (disabled)
	CLIPort          int    `gorm:"default:0" json:"cli_port"`                  // 0 = use protocol default (23/22)
	CLIUsername      string `gorm:"type:varchar(100)" json:"cli_username"`
	CLIPassword      string `gorm:"type:text" json:"cli_password"`               // encrypted
	CLIEnablePassword string `gorm:"type:text" json:"cli_enable_password"`       // encrypted; empty = same as CLIPassword

	// Sync scheduling
	SyncInterval int        `gorm:"default:0" json:"sync_interval"` // minutes; 0 = manual
	LastSyncAt   *time.Time `json:"last_sync_at"`

	// Soft delete
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	DeletedByID *uuid.UUID `gorm:"column:deleted_by;type:uuid" json:"deleted_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	PONPorts   []PONPort    `gorm:"foreignKey:OLTID" json:"pon_ports,omitempty"`
	SyncLogs   []OLTSyncLog `gorm:"foreignKey:OLTID" json:"sync_logs,omitempty"`
}

func (o *OLT) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// ── PONPort ───────────────────────────────────────────────────────────────────

type PONPort struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OLTID     uuid.UUID `gorm:"type:uuid;not null;index" json:"olt_id"`
	OLT       *OLT      `gorm:"foreignKey:OLTID" json:"olt,omitempty"`

	PortIndex int    `gorm:"not null" json:"port_index"` // raw SNMP index
	PortName  string `gorm:"type:varchar(20);not null" json:"port_name"` // e.g. "1/1"
	ONUCount  int    `gorm:"default:0" json:"onu_count"`
	MaxONUs   int    `gorm:"default:64" json:"max_onus"`
	Status    string `gorm:"type:varchar(20);default:'active'" json:"status"`

	ArchivedAt *time.Time `gorm:"index" json:"archived_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	ONUs []ONU `gorm:"foreignKey:PONPortID" json:"onus,omitempty"`
}

func (p *PONPort) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// ── ONU ───────────────────────────────────────────────────────────────────────

type ONU struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OLTID     uuid.UUID `gorm:"type:uuid;not null;index" json:"olt_id"`
	OLT       *OLT      `gorm:"foreignKey:OLTID" json:"olt,omitempty"`
	PONPortID uuid.UUID `gorm:"type:uuid;not null;index" json:"pon_port_id"`
	PONPort   *PONPort  `gorm:"foreignKey:PONPortID" json:"pon_port,omitempty"`

	// Identifiers — unique per OLT + port + slot
	PortIndex int    `gorm:"not null" json:"port_index"` // SNMP port number
	OnuSlot   int    `gorm:"not null" json:"onu_slot"`   // SNMP ONU slot within port
	OnuID     string `gorm:"type:varchar(20);not null" json:"onu_id"` // e.g. "1/1/3"

	// Hardware
	MACAddress   string `gorm:"type:varchar(17);index" json:"mac_address"`
	SerialNumber string `gorm:"type:varchar(50);index" json:"serial_number"`
	Vendor       string `gorm:"type:varchar(100)" json:"vendor"`
	Model        string `gorm:"type:varchar(100)" json:"model"`

	// Status
	Status    string `gorm:"type:varchar(20);default:'offline'" json:"status"`     // online | offline
	RegStatus string `gorm:"type:varchar(20);default:'registered'" json:"reg_status"` // registered | deregistered

	// Optics
	Distance *float64 `gorm:"type:decimal(10,2)" json:"distance"` // meters
	RXPower  *float64 `gorm:"type:decimal(8,2)" json:"rx_power"`  // dBm
	TXPower  *float64 `gorm:"type:decimal(8,2)" json:"tx_power"`  // dBm

	// Timestamps
	LastOnlineAt *time.Time `json:"last_online_at"`
	ArchivedAt   *time.Time `gorm:"index" json:"archived_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Link to PPPoE Internet Account
	InternetAccountID *uuid.UUID      `gorm:"type:uuid;index" json:"internet_account_id,omitempty"`
	InternetAccount   *InternetAccount `gorm:"foreignKey:InternetAccountID" json:"internet_account,omitempty"`
}

func (o *ONU) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// ── OLTSyncLog ────────────────────────────────────────────────────────────────

type OLTSyncLog struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	OLTID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"olt_id"`
	OLT         *OLT       `gorm:"foreignKey:OLTID" json:"olt,omitempty"`
	Status      string     `gorm:"type:varchar(20);default:'running'" json:"status"` // running | success | failed
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	DurationMs  int64      `json:"duration_ms"`

	PortsDiscovered int `json:"ports_discovered"`
	ONUsDiscovered  int `json:"onus_discovered"`
	NewONUs         int `json:"new_onus"`
	UpdatedONUs     int `json:"updated_onus"`
	ArchivedONUs    int `json:"archived_onus"`
	LinkedONUs      int `json:"linked_onus"` // ONUs auto-linked to internet accounts during this sync
	ErrorMessage    string `gorm:"type:text" json:"error_message,omitempty"`
}

func (l *OLTSyncLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// ── OLTStats (computed, not persisted) ───────────────────────────────────────

type OLTStats struct {
	TotalOLTs       int64   `json:"total_olts"`
	ActiveOLTs      int64   `json:"active_olts"`
	TotalPONPorts   int64   `json:"total_pon_ports"`
	TotalONUs       int64   `json:"total_onus"`
	OnlineONUs      int64   `json:"online_onus"`
	OfflineONUs     int64   `json:"offline_onus"`
	UnassignedONUs  int64   `json:"unassigned_onus"`
	PortUtilization float64 `json:"port_utilization_pct"`
}
