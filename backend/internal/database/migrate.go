package database

import (
	"fmt"
	"log"

	"github.com/ispcms/backend/internal/config"
	"github.com/ispcms/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// PrepareSchema runs destructive/structural fixes BEFORE AutoMigrate so that
// AutoMigrate sees a clean slate when creating new indexes.
// Safe to call on every startup — all operations are idempotent.
func PrepareSchema(db *gorm.DB) {
	if !db.Migrator().HasTable("internet_accounts") {
		return // fresh install — nothing to clean
	}

	// 1. Remove nil-UUID and orphaned rows (old sync bug).
	db.Exec(`
		DELETE FROM internet_accounts
		WHERE router_id = '00000000-0000-0000-0000-000000000000'
		   OR router_id NOT IN (SELECT id FROM routers)
	`)

	// 2. Deduplicate by (router_id, username) — keep the most recently updated row.
	//    Required before we can add the new unique index.
	db.Exec(`
		DELETE FROM internet_accounts
		WHERE id IN (
			SELECT id FROM (
				SELECT id,
				       ROW_NUMBER() OVER (
				           PARTITION BY router_id, username
				           ORDER BY updated_at DESC, id DESC
				       ) AS rn
				FROM internet_accounts
			) ranked
			WHERE rn > 1
		)
	`)

	// 3. Drop the old unique index so AutoMigrate can create the new one cleanly.
	db.Exec(`DROP INDEX IF EXISTS idx_ia_router_secret`)

	// 4. Remove phantom routers and their cascade-created dependents.
	//
	//    Phantom routers are created by GORM's association cascade when a model
	//    is saved with zero-value BelongsTo struct fields (e.g. billRepo.Update,
	//    paymentRepo.Create). They are identifiable because:
	//      - name = '' AND ip_address = '' (Go zero-value strings, admins always set these)
	//      - no sync_logs referencing them (they were never actually synced)
	//
	//    Clean in dependency order to avoid FK violations.
	if db.Migrator().HasTable("sync_logs") {
		// 4a. Delete payment_records that belong to bills linked to phantom internet_accounts.
		if db.Migrator().HasTable("payment_records") {
			db.Exec(`
				DELETE FROM payment_records
				WHERE internet_account_id IN (
					SELECT id FROM internet_accounts
					WHERE router_id IN (
						SELECT id FROM routers WHERE deleted_at IS NULL AND name = '' AND ip_address = ''
					)
				)
			`)
		}
		// 4b. Delete bill_generation_logs, monthly_bills for phantom accounts.
		if db.Migrator().HasTable("monthly_bills") {
			db.Exec(`
				DELETE FROM monthly_bills
				WHERE internet_account_id IN (
					SELECT id FROM internet_accounts
					WHERE router_id IN (
						SELECT id FROM routers WHERE deleted_at IS NULL AND name = '' AND ip_address = ''
					)
				)
			`)
		}
		// 4c. Delete customer_subscriptions for phantom accounts.
		if db.Migrator().HasTable("customer_subscriptions") {
			db.Exec(`
				DELETE FROM customer_subscriptions
				WHERE internet_account_id IN (
					SELECT id FROM internet_accounts
					WHERE router_id IN (
						SELECT id FROM routers WHERE deleted_at IS NULL AND name = '' AND ip_address = ''
					)
				)
			`)
		}
		// 4d. Delete phantom internet_accounts (reference phantom routers).
		db.Exec(`
			DELETE FROM internet_accounts
			WHERE router_id IN (
				SELECT id FROM routers WHERE deleted_at IS NULL AND name = '' AND ip_address = ''
			)
		`)
		// 4e. Delete phantom routers (blank name AND blank IP — set by cascade from zero-value structs).
		db.Exec(`
			DELETE FROM routers
			WHERE deleted_at IS NULL AND name = '' AND ip_address = ''
		`)
		// 4f. Legacy cleanup: routers with no sync_logs AND no internet_accounts
		//     (created by older cascade bug path before the above was comprehensive).
		db.Exec(`
			DELETE FROM routers
			WHERE deleted_at IS NULL
			  AND id NOT IN (SELECT DISTINCT router_id FROM sync_logs)
			  AND id NOT IN (SELECT DISTINCT router_id FROM internet_accounts WHERE router_id IS NOT NULL)
		`)
		log.Println("PrepareSchema: cleaned phantom routers and cascade-created dependents")
	}
}

func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")
	return db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.User{},
		&models.UserRole{},
		&models.Router{},
		&models.PPPoESecret{},
		&models.PPPoEActiveSession{},
		&models.SyncLog{},
		&models.ActivityLog{},
		&models.InternetAccount{},
		// Billing module
		&models.Package{},
		&models.ProfileMapping{},
		&models.CustomerSubscription{},
		&models.MonthlyBill{},
		&models.BillGenerationLog{},
		&models.Notification{},
		&models.PaymentRecord{},
		// Expense module
		&models.ExpenseCategory{},
		&models.Expense{},
		// OLT / Network module
		&models.SNMPProfile{},
		&models.OLT{},
		&models.PONPort{},
		&models.ONU{},
		&models.OLTSyncLog{},
	)
}

func Seed(db *gorm.DB, cfg *config.Config) error {
	log.Println("Seeding database...")

	// Default roles
	roles := []models.Role{
		{Name: "super_admin", DisplayName: "Super Admin", Description: "Full system access"},
		{Name: "admin", DisplayName: "Admin", Description: "Administrative access"},
		{Name: "billing_officer", DisplayName: "Billing Officer", Description: "Billing and subscription management"},
		{Name: "operator", DisplayName: "Operator", Description: "Operational access"},
		{Name: "viewer", DisplayName: "Viewer", Description: "Read-only access"},
	}
	for i := range roles {
		db.FirstOrCreate(&roles[i], models.Role{Name: roles[i].Name})
	}

	// Default permissions
	// accounts  = internet account management (/internet-accounts)
	// pppoe     = raw PPPoE data from routers (/pppoe/secrets, /pppoe/sessions)
	// network   = OLTs, PON ports, ONUs, SNMP profiles
	modules := []string{"users", "roles", "routers", "pppoe", "accounts", "dashboard", "billing", "packages", "subscriptions", "notifications", "expenses", "network", "reports"}
	actions := []string{"view", "create", "update", "delete"}
	for _, mod := range modules {
		for _, action := range actions {
			perm := models.Permission{
				Module: mod,
				Action: action,
				Name:   fmt.Sprintf("%s.%s", mod, action),
			}
			db.FirstOrCreate(&perm, models.Permission{Name: perm.Name})
		}
	}

	// Super Admin gets all permissions
	var superAdminRole models.Role
	db.First(&superAdminRole, "name = ?", "super_admin")
	var allPerms []models.Permission
	db.Find(&allPerms)
	for _, p := range allPerms {
		rp := models.RolePermission{RoleID: superAdminRole.ID, PermissionID: p.ID}
		db.FirstOrCreate(&rp, rp)
	}

	// Admin gets all except roles.delete (cannot remove built-in roles)
	var adminRole models.Role
	db.First(&adminRole, "name = ?", "admin")
	for _, p := range allPerms {
		if p.Module == "roles" && p.Action == "delete" {
			continue
		}
		rp := models.RolePermission{RoleID: adminRole.ID, PermissionID: p.ID}
		db.FirstOrCreate(&rp, rp)
	}

	// Billing Officer: view+create+update on billing/packages/subscriptions/notifications, view everywhere
	// No delete on billing records or packages — destructive changes require admin.
	var billingOfficerRole models.Role
	db.First(&billingOfficerRole, "name = ?", "billing_officer")
	for _, p := range allPerms {
		allowed := false
		if (p.Module == "billing" || p.Module == "packages" || p.Module == "subscriptions" || p.Module == "notifications") &&
			(p.Action == "view" || p.Action == "create" || p.Action == "update") {
			allowed = true
		}
		if p.Action == "view" {
			allowed = true
		}
		if allowed {
			rp := models.RolePermission{RoleID: billingOfficerRole.ID, PermissionID: p.ID}
			db.FirstOrCreate(&rp, rp)
		}
	}

	// Operator: manage routers, PPPoE, internet accounts, and network devices.
	// Can view packages (to assist customers). No delete on any module.
	var operatorRole models.Role
	db.First(&operatorRole, "name = ?", "operator")
	for _, p := range allPerms {
		allowed := false
		// Full manage (no delete) on routers, pppoe, accounts, network
		if (p.Module == "routers" || p.Module == "pppoe" || p.Module == "accounts" || p.Module == "network") &&
			(p.Action == "view" || p.Action == "create" || p.Action == "update") {
			allowed = true
		}
		// View-only on dashboard and packages
		if (p.Module == "dashboard" || p.Module == "packages") && p.Action == "view" {
			allowed = true
		}
		if allowed {
			rp := models.RolePermission{RoleID: operatorRole.ID, PermissionID: p.ID}
			db.FirstOrCreate(&rp, rp)
		}
	}

	// Viewer: view only
	var viewerRole models.Role
	db.First(&viewerRole, "name = ?", "viewer")
	for _, p := range allPerms {
		if p.Action == "view" {
			rp := models.RolePermission{RoleID: viewerRole.ID, PermissionID: p.ID}
			db.FirstOrCreate(&rp, rp)
		}
	}

	// Super Admin user
	var existingUser models.User
	if db.First(&existingUser, "username = ?", cfg.SuperAdminUsername).Error != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SuperAdminPassword), 12)
		if err != nil {
			return err
		}
		user := models.User{
			FullName: "Super Administrator",
			Username: cfg.SuperAdminUsername,
			Email:    cfg.SuperAdminEmail,
			Password: string(hash),
			Status:   "active",
		}
		if err := db.Create(&user).Error; err != nil {
			return err
		}
		db.Create(&models.UserRole{UserID: user.ID, RoleID: superAdminRole.ID})
		log.Printf("Created Super Admin: %s / %s", cfg.SuperAdminUsername, cfg.SuperAdminPassword)
	}

	// Default expense categories
	defaultCategories := []string{
		"Office Rent", "Staff Salary", "Electricity", "Internet Upstream",
		"Fiber Maintenance", "ONU Purchase", "Router Purchase", "Switch Purchase",
		"Cable Purchase", "Fuel", "Generator", "Vehicle", "Office Supplies",
		"Marketing", "Software Subscription", "Bank Charge", "Miscellaneous",
	}
	for _, name := range defaultCategories {
		var cat models.ExpenseCategory
		if db.First(&cat, "name = ?", name).Error != nil {
			db.Create(&models.ExpenseCategory{Name: name, Status: "active"})
		}
	}

	// Billing officer gets view/create/update on expenses
	for _, p := range allPerms {
		if p.Module == "expenses" && (p.Action == "view" || p.Action == "create" || p.Action == "update") {
			rp := models.RolePermission{RoleID: billingOfficerRole.ID, PermissionID: p.ID}
			db.FirstOrCreate(&rp, rp)
		}
	}

	// Seed default SNMP profiles
	seedSNMPProfiles(db)

	return nil
}

func seedSNMPProfiles(db *gorm.DB) {
	profiles := []models.SNMPProfile{
		{
			Name:       "BDCOM_EPON",
			Vendor:     "BDCOM",
			Technology: "EPON",
			Description: "BDCOM EPON OLT (P3310, P3600 series)",
			OIDMap: models.OIDMap{
				"sys_name":        "1.3.6.1.2.1.1.5.0",
				"sys_descr":       "1.3.6.1.2.1.1.1.0",
				"onu_mac":         "1.3.6.1.4.1.3320.101.10.1.1.1",
				"onu_status":      "1.3.6.1.4.1.3320.101.10.1.1.9",
				"onu_rx_power":    "1.3.6.1.4.1.3320.101.10.1.1.30",
				"onu_tx_power":    "1.3.6.1.4.1.3320.101.10.1.1.31",
				"onu_distance":    "1.3.6.1.4.1.3320.101.10.1.1.28",
				"onu_serial":      "1.3.6.1.4.1.3320.101.10.1.1.3",
				"onu_model":       "1.3.6.1.4.1.3320.101.10.1.1.4",
				"index_port_pos":  "0",
				"index_onu_pos":   "1",
				"power_divisor":   "10",
				"distance_unit":   "m",
			},
		},
		{
			Name:       "VSOL_EPON",
			Vendor:     "VSOL",
			Technology: "EPON",
			Description: "VSOL EPON OLT (V1600, V1700 series)",
			OIDMap: models.OIDMap{
				"sys_name":       "1.3.6.1.2.1.1.5.0",
				"sys_descr":      "1.3.6.1.2.1.1.1.0",
				"onu_mac":        "1.3.6.1.4.1.37950.1.1.3.10.1.1",
				"onu_status":     "1.3.6.1.4.1.37950.1.1.3.10.1.2",
				"onu_rx_power":   "1.3.6.1.4.1.37950.1.1.3.10.1.5",
				"onu_tx_power":   "1.3.6.1.4.1.37950.1.1.3.10.1.6",
				"onu_distance":   "1.3.6.1.4.1.37950.1.1.3.10.1.7",
				"onu_serial":     "1.3.6.1.4.1.37950.1.1.3.10.1.3",
				"index_port_pos": "0",
				"index_onu_pos":  "1",
				"power_divisor":  "10",
				"distance_unit":  "m",
			},
		},
		{
			Name:       "VSOL_EPON_V2",
			Vendor:     "VSOL",
			Technology: "EPON",
			// Uses the 1.1.5 MIB subtree. ONU port/slot are encoded in the
			// interface name column (e.g. "EPON0/3:14" → port 3, slot 14).
			Description: "VSOL EPON OLT V2 (1.1.5 MIB tree — VS-S1600-16T4GE-L and similar)",
			OIDMap: models.OIDMap{
				"sys_name":   "1.3.6.1.2.1.1.5.0",
				"sys_descr":  "1.3.6.1.2.1.1.1.0",
				"onu_mac":    "1.3.6.1.4.1.37950.1.1.5.10.3.2.1.3",
				"onu_status": "1.3.6.1.4.1.37950.1.1.5.10.3.2.1.4",
				// onu_ifname: flat ONU index → "EPON0/P:S" — used to determine
				// portIdx (P) and onuSlot (S) instead of the MAC table index.
				"onu_ifname": "1.3.6.1.4.1.37950.1.1.5.10.3.2.1.5",
			},
		},
		{
			Name:       "CDATA_EPON",
			Vendor:     "C-Data",
			Technology: "EPON",
			Description: "C-Data EPON OLT (FD1104S, FD1204S series)",
			OIDMap: models.OIDMap{
				"sys_name":       "1.3.6.1.2.1.1.5.0",
				"sys_descr":      "1.3.6.1.2.1.1.1.0",
				"onu_mac":        "1.3.6.1.4.1.34592.1.2.5.1.1",
				"onu_status":     "1.3.6.1.4.1.34592.1.2.5.1.2",
				"onu_rx_power":   "1.3.6.1.4.1.34592.1.2.5.1.8",
				"onu_tx_power":   "1.3.6.1.4.1.34592.1.2.5.1.9",
				"onu_distance":   "1.3.6.1.4.1.34592.1.2.5.1.10",
				"onu_serial":     "1.3.6.1.4.1.34592.1.2.5.1.3",
				"index_port_pos": "0",
				"index_onu_pos":  "1",
				"power_divisor":  "10",
				"distance_unit":  "m",
			},
		},
		{
			Name:       "HUAWEI_GPON",
			Vendor:     "Huawei",
			Technology: "GPON",
			Description: "Huawei GPON OLT (MA5800, MA5608T series)",
			OIDMap: models.OIDMap{
				"sys_name":       "1.3.6.1.2.1.1.5.0",
				"sys_descr":      "1.3.6.1.2.1.1.1.0",
				"onu_mac":        "1.3.6.1.4.1.2011.6.128.1.1.2.51.1.2",
				"onu_status":     "1.3.6.1.4.1.2011.6.128.1.1.2.51.1.7",
				"onu_rx_power":   "1.3.6.1.4.1.2011.6.128.1.1.2.51.1.8",
				"onu_tx_power":   "1.3.6.1.4.1.2011.6.128.1.1.2.51.1.9",
				"onu_distance":   "1.3.6.1.4.1.2011.6.128.1.1.2.51.1.10",
				"onu_serial":     "1.3.6.1.4.1.2011.6.128.1.1.2.51.1.3",
				"index_port_pos": "2",
				"index_onu_pos":  "3",
				"power_divisor":  "10",
				"distance_unit":  "m",
			},
		},
		{
			Name:       "ZTE_GPON",
			Vendor:     "ZTE",
			Technology: "GPON",
			Description: "ZTE GPON OLT (C600, C300 series)",
			OIDMap: models.OIDMap{
				"sys_name":       "1.3.6.1.2.1.1.5.0",
				"sys_descr":      "1.3.6.1.2.1.1.1.0",
				"onu_mac":        "1.3.6.1.4.1.3902.1082.500.5.1.1.1.1",
				"onu_status":     "1.3.6.1.4.1.3902.1082.500.5.1.1.1.2",
				"onu_rx_power":   "1.3.6.1.4.1.3902.1082.500.5.1.1.1.6",
				"onu_tx_power":   "1.3.6.1.4.1.3902.1082.500.5.1.1.1.7",
				"onu_serial":     "1.3.6.1.4.1.3902.1082.500.5.1.1.1.3",
				"index_port_pos": "1",
				"index_onu_pos":  "2",
				"power_divisor":  "10",
				"distance_unit":  "m",
			},
		},
		{
			Name:        "RICHERLINK_EPON",
			Vendor:      "Richerlink",
			Technology:  "EPON",
			Description: "Richerlink EPON OLT (RL8004EL, older firmware). Per-ONU RX/TX power via separate optical table.",
			OIDMap: models.OIDMap{
				"sys_name":             "1.3.6.1.2.1.1.5.0",
				"sys_descr":            "1.3.6.1.2.1.1.1.0",
				"onu_mac":              "1.3.6.1.4.1.34168.2.3.5.1.1.3",
				"onu_status":           "1.3.6.1.4.1.34168.2.3.5.1.1.9",
				"onu_model":            "1.3.6.1.4.1.34168.2.3.5.1.1.25",
				"onu_distance":         "1.3.6.1.4.1.34168.2.3.5.1.1.28",
				"onu_rx_power":         "1.3.6.1.4.1.34168.2.3.5.6.1.6",
				"onu_tx_power":         "1.3.6.1.4.1.34168.2.3.5.6.1.5",
				"index_packed":         "true",
				"status_online_string": "configuration ok",
				"power_divisor":        "1",
				"distance_unit":        "m",
			},
		},
		{
			Name:        "RICHERLINK_EPON_V2",
			Vendor:      "Richerlink",
			Technology:  "EPON",
			Description: "Richerlink EPON OLT (RL8004EL, firmware V1.0.0.32715+). Uses GETNEXT walk — this firmware does not respond to GETBULK. Per-ONU optical power not available.",
			OIDMap: models.OIDMap{
				"sys_name":             "1.3.6.1.2.1.1.5.0",
				"sys_descr":            "1.3.6.1.2.1.1.1.0",
				"onu_mac":              "1.3.6.1.4.1.34168.2.3.5.1.1.3",
				"onu_status":           "1.3.6.1.4.1.34168.2.3.5.1.1.9",
				"index_packed":         "true",
				"status_online_string": "configuration ok",
				"power_divisor":        "1",
				"use_getnext":          "true",
			},
		},
	}

	for i := range profiles {
		var existing models.SNMPProfile
		if db.First(&existing, "name = ?", profiles[i].Name).Error != nil {
			db.Create(&profiles[i])
			log.Printf("Seeded SNMP profile: %s", profiles[i].Name)
		} else {
			// Always update seeded profiles so OID map and description stay in sync
			db.Model(&existing).Updates(map[string]interface{}{
				"description": profiles[i].Description,
				"oid_map":     profiles[i].OIDMap,
				"vendor":      profiles[i].Vendor,
				"technology":  profiles[i].Technology,
			})
			log.Printf("Updated SNMP profile: %s", profiles[i].Name)
		}
	}

	// Fix up any VSOL profiles using the 1.1.5.12.1.25.1 OID tree that
	// are missing onu_rx_power (e.g. user-created profiles from the UI).
	fixupVSOL512Profiles(db)
}

// fixupVSOL512Profiles adds onu_rx_power / power_divisor to any SNMP profile
// whose onu_mac OID is under the 1.3.6.1.4.1.37950.1.1.5.12.1.25.1 tree but
// is missing a rx_power entry. This covers user-created VSOL V1600D profiles.
func fixupVSOL512Profiles(db *gorm.DB) {
	const macPrefix = "1.3.6.1.4.1.37950.1.1.5.12.1.25.1.5"
	const rxOID = "1.3.6.1.4.1.37950.1.1.5.12.1.25.1.12"

	var profiles []models.SNMPProfile
	if err := db.Find(&profiles).Error; err != nil {
		return
	}
	for _, p := range profiles {
		mac, hasMac := p.OIDMap["onu_mac"]
		_, hasRx := p.OIDMap["onu_rx_power"]
		if hasMac && mac == macPrefix && !hasRx {
			p.OIDMap["onu_rx_power"] = rxOID
			p.OIDMap["power_divisor"] = "-10"
			if err := db.Save(&p).Error; err == nil {
				log.Printf("Patched VSOL profile '%s' with onu_rx_power OID", p.Name)
			}
		}
	}
}
