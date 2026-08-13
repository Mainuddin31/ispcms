package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/pkg/snmp"
	"github.com/ispcms/backend/pkg/telnet"
	"github.com/ispcms/backend/pkg/utils"
)

// OLTSyncService synchronises a single OLT via SNMP.
type OLTSyncService interface {
	Sync(oltID uuid.UUID) (*models.OLTSyncLog, error)
	TestConnection(oltID uuid.UUID) error
	TestConnectionRaw(ip, community string, port int, version string) error
	// SNMPProbe walks the given OID on the OLT and returns up to limit entries
	// as raw key→value strings. Useful for testing OIDs before adding to a profile.
	SNMPProbe(oltID uuid.UUID, oid string, limit int) ([]SNMPProbeEntry, error)
}

// SNMPProbeEntry is one raw result from an SNMP walk.
type SNMPProbeEntry struct {
	OIDSuffix string `json:"oid_suffix"`
	Value     string `json:"value"`
}

type oltSyncService struct {
	oltRepo      repositories.OLTRepository
	portRepo     repositories.PONPortRepository
	onuRepo      repositories.ONURepository
	syncLogRepo  repositories.OLTSyncLogRepository
	activitySvc  ActivityService
	jwtSecret    string // used to decrypt stored passwords
}

func NewOLTSyncService(
	oltRepo repositories.OLTRepository,
	portRepo repositories.PONPortRepository,
	onuRepo repositories.ONURepository,
	syncLogRepo repositories.OLTSyncLogRepository,
	activitySvc ActivityService,
	jwtSecret string,
) OLTSyncService {
	return &oltSyncService{
		oltRepo:     oltRepo,
		portRepo:    portRepo,
		onuRepo:     onuRepo,
		syncLogRepo: syncLogRepo,
		activitySvc: activitySvc,
		jwtSecret:   jwtSecret,
	}
}

// Sync performs a full SNMP synchronisation of the given OLT.
func (s *oltSyncService) Sync(oltID uuid.UUID) (*models.OLTSyncLog, error) {
	olt, err := s.oltRepo.FindByID(oltID)
	if err != nil {
		return nil, fmt.Errorf("OLT not found: %w", err)
	}
	if olt.SNMPProfile == nil {
		return nil, fmt.Errorf("OLT has no SNMP profile")
	}

	log := &models.OLTSyncLog{
		OLTID:     oltID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	_ = s.syncLogRepo.Create(log)

	start := time.Now()
	syncErr := s.doSync(olt, log)
	log.DurationMs = time.Since(start).Milliseconds()
	now := time.Now()
	log.CompletedAt = &now

	if syncErr != nil {
		log.Status = "failed"
		log.ErrorMessage = syncErr.Error()
	} else {
		log.Status = "success"
		_ = s.oltRepo.UpdateLastSync(oltID, now)
		if s.activitySvc != nil {
			desc := fmt.Sprintf("OLT: %s | Ports: %d | ONUs: %d | New: %d | Updated: %d | Auto-linked: %d",
				olt.Name, log.PortsDiscovered, log.ONUsDiscovered, log.NewONUs, log.UpdatedONUs, log.LinkedONUs)
			s.activitySvc.Log(nil, "network", "olt_sync_completed", "OLT Sync Completed", desc, "olt_sync_log", log.ID.String())
		}
	}
	_ = s.syncLogRepo.Update(log)
	return log, syncErr
}

func (s *oltSyncService) doSync(olt *models.OLT, syncLog *models.OLTSyncLog) error {
	// Overall sync timeout: 90 seconds. Prevents hangs on unresponsive OLTs
	// (e.g. Richerlink devices that stop responding mid-BulkWalk).
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client, err := s.buildClient(olt)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close()
	client.SetContext(ctx)
	if olt.SNMPProfile != nil && olt.SNMPProfile.OIDMap["use_getnext"] == "true" {
		client.SetUseGetNext(true)
	}

	log := syncLog

	profile := olt.SNMPProfile
	oids := profile.OIDMap

	// ── 1. PON Port discovery ─────────────────────────────────────────────────
	portBaseOID, hasPortOID := oids["pon_port_table"]
	activePortIndexes := []int{}
	portIndexToID := map[int]uuid.UUID{}

	if hasPortOID && portBaseOID != "" {
		portWalk, err := client.Walk(portBaseOID)
		if err == nil && len(portWalk) > 0 {
			for suffix := range portWalk {
				parts := snmp.ParseIndex(suffix)
				if len(parts) < 1 {
					continue
				}
				portIdx := parts[0]
				activePortIndexes = append(activePortIndexes, portIdx)
			}
		}
	}

	// If no port walk defined or yielded nothing, we'll discover ports from ONU walk
	// We'll handle port creation lazily during ONU processing.

	// ── 2. ONU MAC walk ──────────────────────────────────────────────────────
	macBaseOID, hasMacOID := oids["onu_mac"]
	if !hasMacOID || macBaseOID == "" {
		return fmt.Errorf("SNMP profile missing onu_mac OID")
	}

	macWalk, err := client.Walk(macBaseOID)
	if err != nil {
		return fmt.Errorf("ONU MAC walk failed: %w", err)
	}

	// Read index positions from profile (defaults: port=0, onu=1)
	portPos := profileInt(oids, "index_port_pos", 0)
	onuPos := profileInt(oids, "index_onu_pos", 1)
	powerDiv := profileFloat(oids, "power_divisor", 10.0)
	indexPacked := oids["index_packed"] == "true"     // single int: port<<16|slot<<8|onu
	statusOnlineStr := oids["status_online_string"]   // string match for online status
	rxNegate := oids["rx_power_negate"] == "true"

	// ── 3. Walk optional OIDs ─────────────────────────────────────────────────
	// onu_ifname: VSOL V2-style OID where each entry is "EPON0/P:S".
	// Port and slot are parsed from this string; the index suffix is a flat integer.
	ifnameWalk := walkOpt(client, oids, "onu_ifname")
	statusWalk := walkOpt(client, oids, "onu_status")
	rxWalk := walkOpt(client, oids, "onu_rx_power")
	txWalk := walkOpt(client, oids, "onu_tx_power")
	distWalk := walkOpt(client, oids, "onu_distance")
	serialWalk := walkOpt(client, oids, "onu_serial")
	modelWalk := walkOpt(client, oids, "onu_model")

	// ── 4. Process each ONU ──────────────────────────────────────────────────
	log.ONUsDiscovered = len(macWalk)

	activeKeys := make([][2]int, 0, len(macWalk))

	for suffix, macRaw := range macWalk {
		var portIdx, onuSlot int

		if len(ifnameWalk) > 0 {
			// VSOL V2-style: interface name "EPON0/P:S" encodes port and slot.
			// The index suffix is a flat integer; skip non-PON entries (CPU, GE1…).
			ifnameRaw, ok := ifnameWalk[suffix]
			if !ok {
				continue
			}
			p, s, err := parseEPONIfname(fmt.Sprintf("%v", ifnameRaw))
			if err != nil {
				continue // non-PON entry — skip
			}
			portIdx = p
			onuSlot = s
		} else if indexPacked {
			// Richerlink-style: single integer encodes port<<16 | slot<<8 | onu
			parts := snmp.ParseIndex(suffix)
			if len(parts) != 1 {
				continue
			}
			packed := parts[0]
			portIdx = (packed >> 16) & 0xFF
			onuSlot = (packed >> 8) & 0xFF
		} else {
			parts := snmp.ParseIndex(suffix)
			if len(parts) <= onuPos || len(parts) <= portPos {
				continue
			}
			portIdx = parts[portPos]
			onuSlot = parts[onuPos]
		}

		activeKeys = append(activeKeys, [2]int{portIdx, onuSlot})

		// Ensure PON port exists
		portID := s.ensurePONPort(olt.ID, portIdx, portIndexToID)
		if portID == uuid.Nil {
			continue
		}

		onuID := fmt.Sprintf("%d/%d/%d", 1, portIdx, onuSlot)
		macStr := normalizeMac(fmt.Sprintf("%v", macRaw))

		// Status: support both integer (1=online) and string comparison
		oStatus := "offline"
		if sv, ok := statusWalk[suffix]; ok {
			if statusOnlineStr != "" {
				if str, ok2 := sv.(string); ok2 && strings.Contains(str, statusOnlineStr) {
					oStatus = "online"
				}
			} else {
				if n, ok2 := sv.(int64); ok2 && n == 1 {
					oStatus = "online"
				}
			}
		}

		rxPow := extractPower(rxWalk, suffix, powerDiv, rxNegate)
		txPow := extractPower(txWalk, suffix, powerDiv, false)

		var dist *float64
		if dv, ok := distWalk[suffix]; ok {
			if n, ok2 := dv.(int64); ok2 {
				unit := oids["distance_unit"]
				f := float64(n)
				if unit == "cm" {
					f = f / 100.0
				}
				dist = &f
			}
		}

		serial := ""
		if sv, ok := serialWalk[suffix]; ok {
			serial = fmt.Sprintf("%v", sv)
		}
		modelStr := ""
		if mv, ok := modelWalk[suffix]; ok {
			modelStr = fmt.Sprintf("%v", mv)
		}

		now := time.Now()
		var lastOnlineAt *time.Time
		if oStatus == "online" {
			lastOnlineAt = &now
		}
		onu := &models.ONU{
			OLTID:        olt.ID,
			PONPortID:    portID,
			PortIndex:    portIdx,
			OnuSlot:      onuSlot,
			OnuID:        onuID,
			MACAddress:   macStr,
			SerialNumber: serial,
			Model:        modelStr,
			Status:       oStatus,
			RegStatus:    "registered",
			RXPower:      rxPow,
			TXPower:      txPow,
			Distance:     dist,
			LastOnlineAt: lastOnlineAt,
			UpdatedAt:    now,
		}

		isNew, err := s.onuRepo.Upsert(onu)
		if err == nil {
			if isNew {
				log.NewONUs++
			} else {
				log.UpdatedONUs++
			}
		}
	}

	// ── 5. Archive missing ONUs ───────────────────────────────────────────────
	archived, _ := s.onuRepo.ArchiveMissing(olt.ID, activeKeys)
	log.ArchivedONUs = int(archived)

	// ── 5b. Auto-link ONUs to internet accounts by ONU MAC ───────────────────
	// Fast path: matches onus.mac_address against internet_accounts.caller_id
	// using a single SQL UPDATE. Works when the ONU hardware MAC == caller_id.
	linked, _ := s.onuRepo.AutoLinkByMAC(&olt.ID)
	syncLog.LinkedONUs = int(linked)

	// ── 5c. Auto-link via bridge MAC table (FDB or CLI) ──────────────────────
	// Preferred: CLI scrape (Telnet/SSH) — gives exact ONU-level MACs on OLTs
	// whose SNMP FDB only exposes port-level entries (e.g. Richerlink).
	// Fallback: SNMP FDB walk — used when CLI is not configured (e.g. VSOL).
	var macEntries []repositories.MACTableEntry
	if olt.CLIProtocol != "" && olt.CLIUsername != "" {
		macEntries = s.walkCLIMACTable(olt)
	}
	if len(macEntries) == 0 {
		macEntries = s.walkFDBTable(client, oids)
	}
	if len(macEntries) > 0 {
		fdbLinked, _ := s.onuRepo.LinkFromMACTable(olt.ID, macEntries)
		syncLog.LinkedONUs += int(fdbLinked)
	}

	// ── 6. Update port counts ─────────────────────────────────────────────────
	for portIdx, portID := range portIndexToID {
		count, _ := s.onuRepo.CountByOLTAndPort(olt.ID, portID)
		_ = s.portRepo.UpdateONUCount(portID, int(count))
		_ = portIdx
	}

	// Archive missing ports
	if len(activePortIndexes) > 0 {
		_, _ = s.portRepo.ArchiveMissing(olt.ID, activePortIndexes)
	}

	// Port count from discovered port IDs
	log.PortsDiscovered = len(portIndexToID)

	return nil
}

// ensurePONPort creates the PON port in DB if it doesn't already exist and
// returns its UUID. portIndexToID is a local cache updated in place.
func (s *oltSyncService) ensurePONPort(oltID uuid.UUID, portIdx int, portIndexToID map[int]uuid.UUID) uuid.UUID {
	if id, ok := portIndexToID[portIdx]; ok {
		return id
	}
	port := &models.PONPort{
		OLTID:     oltID,
		PortIndex: portIdx,
		PortName:  fmt.Sprintf("1/%d", portIdx),
		Status:    "active",
		MaxONUs:   64,
	}
	if err := s.portRepo.Upsert(port); err != nil {
		return uuid.Nil
	}
	// fetch ID
	p, err := s.portRepo.FindByOLTAndIndex(oltID, portIdx)
	if err != nil {
		return uuid.Nil
	}
	portIndexToID[portIdx] = p.ID
	return p.ID
}

// TestConnection opens a transient SNMP session and verifies connectivity.
func (s *oltSyncService) TestConnection(oltID uuid.UUID) error {
	olt, err := s.oltRepo.FindByID(oltID)
	if err != nil {
		return fmt.Errorf("OLT not found")
	}
	client, err := s.buildClient(olt)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.TestConnection()
}

func (s *oltSyncService) TestConnectionRaw(ip, community string, port int, version string) error {
	cfg := snmp.Config{
		Host:      ip,
		Port:      uint16(port),
		Version:   version,
		Community: community,
		Timeout:   5 * time.Second,
		Retries:   1,
	}
	client, err := snmp.New(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.TestConnection()
}

// SNMPProbe walks oid on the given OLT and returns up to limit raw entries.
func (s *oltSyncService) SNMPProbe(oltID uuid.UUID, oid string, limit int) ([]SNMPProbeEntry, error) {
	olt, err := s.oltRepo.FindByID(oltID)
	if err != nil {
		return nil, fmt.Errorf("OLT not found")
	}
	client, err := s.buildClient(olt)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer client.Close()

	if olt.SNMPProfile != nil && olt.SNMPProfile.OIDMap["use_getnext"] == "true" {
		client.SetUseGetNext(true)
	}

	walkResult, err := client.Walk(oid)
	if err != nil {
		return nil, fmt.Errorf("walk failed: %w", err)
	}

	if limit <= 0 || limit > 200 {
		limit = 20
	}

	entries := make([]SNMPProbeEntry, 0, limit)
	for suffix, val := range walkResult {
		entries = append(entries, SNMPProbeEntry{
			OIDSuffix: suffix,
			Value:     fmt.Sprintf("%v", val),
		})
		if len(entries) >= limit {
			break
		}
	}
	return entries, nil
}

// buildClient creates an SNMP client from the OLT configuration.
func (s *oltSyncService) buildClient(olt *models.OLT) (*snmp.Client, error) {
	port := uint16(olt.SNMPPort)
	if port == 0 {
		port = 161
	}
	timeout := time.Duration(olt.Timeout) * time.Second
	if timeout == 0 {
		timeout = 3 * time.Second // default 3s per PDU (was 5s)
	}
	retries := olt.Retries
	if retries == 0 {
		retries = 2 // default 2 retries (was 3) — combined with 90s ctx timeout
	}

	cfg := snmp.Config{
		Host:    olt.ManagementIP,
		Port:    port,
		Version: olt.SNMPVersion,
		Timeout: timeout,
		Retries: retries,
	}

	switch olt.SNMPVersion {
	case "v3":
		cfg.V3Username = olt.V3Username
		cfg.V3AuthProtocol = olt.V3AuthProtocol
		cfg.V3PrivProtocol = olt.V3PrivProtocol
		if olt.V3AuthPassword != "" {
			p, _ := utils.Decrypt(olt.V3AuthPassword, s.jwtSecret)
			cfg.V3AuthPassword = p
		}
		if olt.V3PrivPassword != "" {
			p, _ := utils.Decrypt(olt.V3PrivPassword, s.jwtSecret)
			cfg.V3PrivPassword = p
		}
	default:
		cfg.Community = olt.Community
	}

	return snmp.New(cfg)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func walkOpt(client *snmp.Client, oids models.OIDMap, key string) map[string]interface{} {
	base, ok := oids[key]
	if !ok || base == "" {
		return map[string]interface{}{}
	}
	result, err := client.Walk(base)
	if err != nil {
		log.Printf("[SNMP] walkOpt: walk failed for key=%s oid=%s: %v", key, base, err)
		return map[string]interface{}{}
	}
	log.Printf("[SNMP] walkOpt: key=%s oid=%s returned %d entries", key, base, len(result))
	return result
}

// toSignedPower converts a raw SNMP int64 value to a signed int64,
// interpreting Gauge32 values > int32 max as two's-complement negative.
func toSignedPower(n int64) int64 {
	const maxInt32 = int64(2147483647)
	const uint32Max = int64(4294967296)
	if n > maxInt32 {
		return n - uint32Max
	}
	return n
}

// extractPower reads a power value from a walk result, handling both integer
// (VSOL-style) and string float (Richerlink-style like "-9.17") encodings.
func extractPower(walk map[string]interface{}, suffix string, divisor float64, negate bool) *float64 {
	rv, ok := walk[suffix]
	if !ok {
		return nil
	}
	var f float64
	switch v := rv.(type) {
	case int64:
		if v == 0 {
			return nil
		}
		f = float64(toSignedPower(v)) / divisor
	case string:
		clean := strings.Trim(v, "\" \t\n\r")
		val, err := strconv.ParseFloat(clean, 64)
		if err != nil || val == 0 {
			return nil
		}
		f = val
	default:
		return nil
	}
	if negate {
		f = -f
	}
	return &f
}

// normalizeMac converts various MAC formats to colon-separated lowercase.
// Handles Cisco "80d4.a51b.4eef" and standard "80:d4:a5:1b:4e:ef" formats.
func normalizeMac(mac string) string {
	clean := strings.Trim(mac, "\" \t\n\r")
	// Cisco format: "80d4.a51b.4eef"
	if strings.Count(clean, ".") == 2 {
		hex := strings.ReplaceAll(clean, ".", "")
		if len(hex) == 12 {
			parts := make([]string, 6)
			for i := 0; i < 6; i++ {
				parts[i] = hex[i*2 : i*2+2]
			}
			return strings.Join(parts, ":")
		}
	}
	return clean
}

func profileInt(oids models.OIDMap, key string, def int) int {
	if v, ok := oids[key]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func profileFloat(oids models.OIDMap, key string, def float64) float64 {
	if v, ok := oids[key]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f != 0 {
			return f
		}
	}
	return def
}

// parseEPONIfname parses a VSOL V2 interface name of the form "EPON<shelf>/<port>:<slot>"
// into portIdx and onuSlot. Returns an error for non-PON names (e.g. "CPU", "GE1").
func parseEPONIfname(name string) (portIdx, onuSlot int, err error) {
	name = strings.TrimSpace(name)
	slashIdx := strings.LastIndex(name, "/")
	colonIdx := strings.LastIndex(name, ":")
	if slashIdx < 0 || colonIdx < 0 || colonIdx <= slashIdx {
		return 0, 0, fmt.Errorf("not EPON ifname: %q", name)
	}
	p, err := strconv.Atoi(name[slashIdx+1 : colonIdx])
	if err != nil {
		return 0, 0, fmt.Errorf("bad port in %q: %w", name, err)
	}
	s, err := strconv.Atoi(name[colonIdx+1:])
	if err != nil {
		return 0, 0, fmt.Errorf("bad slot in %q: %w", name, err)
	}
	return p, s, nil
}

// parseONUIfDescr extracts (portIdx, onuSlot) from ONU interface description strings
// returned by SNMP ifDescr. Supports multiple common EPON naming formats:
//
//	"EPON0/1:12"  → port=1, slot=12  (VSOL V2 / slash-colon format)
//	"epon1:onu12" → port=1, slot=12  (epon<P>:onu<S> format)
//	"epon1:12"    → port=1, slot=12  (epon<P>:<S> format)
//
// Returns an error for non-ONU interfaces (uplink GE ports, CPU, etc.).
func parseONUIfDescr(name string) (portIdx, onuSlot int, err error) {
	s := strings.TrimSpace(strings.ToLower(name))

	// Format 1: "EPON0/P:S" — existing logic covers this
	if strings.Contains(s, "/") && strings.Contains(s, ":") {
		return parseEPONIfname(name)
	}

	// Format 2 & 3: "eponP:onuS" or "eponP:S"
	if strings.HasPrefix(s, "epon") {
		rest := s[4:] // strip "epon"
		colonIdx := strings.Index(rest, ":")
		if colonIdx < 0 {
			return 0, 0, fmt.Errorf("not ONU ifDescr: %q", name)
		}
		portStr := rest[:colonIdx]
		slotPart := strings.TrimPrefix(rest[colonIdx+1:], "onu") // strip optional "onu" prefix
		p, err1 := strconv.Atoi(portStr)
		sl, err2 := strconv.Atoi(slotPart)
		if err1 != nil || err2 != nil {
			return 0, 0, fmt.Errorf("not ONU ifDescr: %q", name)
		}
		return p, sl, nil
	}

	return 0, 0, fmt.Errorf("not ONU ifDescr: %q", name)
}

// walkFDBTable walks the OLT's bridge MAC address table (FDB) via SNMP and
// returns a list of (customer_CPE_MAC, portIdx, onuSlot) entries.
//
// It uses three OIDs from the SNMP profile:
//   - mac_table_oid           dot1qTpFdbPort  1.3.6.1.2.1.17.7.1.2.2.1.2
//   - mac_table_port_ifindex_oid  dot1dBasePortIfIndex  1.3.6.1.2.1.17.1.4.1.2
//   - mac_table_ifdescr_oid   ifDescr  1.3.6.1.2.1.2.2.1.2
//
// If any OID is missing or a walk fails, an empty slice is returned silently.
func (s *oltSyncService) walkFDBTable(client *snmp.Client, oids models.OIDMap) []repositories.MACTableEntry {
	fdbOID, ok1 := oids["mac_table_oid"]
	portIfIdxOID, ok2 := oids["mac_table_port_ifindex_oid"]
	ifDescrOID, ok3 := oids["mac_table_ifdescr_oid"]
	if !ok1 || !ok2 || !ok3 || fdbOID == "" || portIfIdxOID == "" || ifDescrOID == "" {
		return nil
	}

	// ── Step 1: Walk dot1qTpFdbPort ──────────────────────────────────────────
	// Index suffix: <fid>.<b1>.<b2>.<b3>.<b4>.<b5>.<b6>  value: port_number
	fdbWalk, err := client.Walk(fdbOID)
	if err != nil || len(fdbWalk) == 0 {
		log.Printf("[SNMP-FDB] mac_table_oid walk failed or empty: %v", err)
		return nil
	}

	// (port_number → list of MACs learned on that port)
	portMACs := map[int][]string{}
	for suffix, val := range fdbWalk {
		parts := snmp.ParseIndex(suffix)
		if len(parts) < 6 {
			continue // need at least 6 MAC bytes
		}
		// Always take the last 6 elements as MAC bytes.
		// Different OLT vendors prefix the index differently:
		//   Standard dot1q: <fid>.<mac6>            → 7 elements
		//   VSOL EPON:      <extra>.<fid>.<mac6>    → 8 elements
		//   Richerlink:     <mac6> only             → 6 elements
		macBytes := parts[len(parts)-6:]
		var hexParts [6]string
		for i, b := range macBytes {
			hexParts[i] = fmt.Sprintf("%02x", b)
		}
		mac := strings.Join(hexParts[:], ":")

		portNum, ok := val.(int64)
		if !ok || portNum <= 0 {
			continue
		}
		portMACs[int(portNum)] = append(portMACs[int(portNum)], mac)
	}

	if len(portMACs) == 0 {
		return nil
	}

	// ── Step 2: Walk dot1dBasePortIfIndex ────────────────────────────────────
	// Index suffix: <port_number>  value: ifIndex
	portIfIdxWalk, err := client.Walk(portIfIdxOID)
	if err != nil {
		log.Printf("[SNMP-FDB] mac_table_port_ifindex_oid walk failed: %v", err)
		return nil
	}
	portToIfIdx := map[int]int{}
	for suffix, val := range portIfIdxWalk {
		parts := snmp.ParseIndex(suffix)
		if len(parts) < 1 {
			continue
		}
		ifIdx, ok := val.(int64)
		if !ok || ifIdx <= 0 {
			continue
		}
		portToIfIdx[parts[0]] = int(ifIdx)
	}

	// ── Step 3: Walk ifDescr ─────────────────────────────────────────────────
	// Index suffix: <ifIndex>  value: interface name string
	ifDescrWalk, err := client.Walk(ifDescrOID)
	if err != nil {
		log.Printf("[SNMP-FDB] mac_table_ifdescr_oid walk failed: %v", err)
		return nil
	}
	ifIdxToDescr := map[int]string{}
	for suffix, val := range ifDescrWalk {
		parts := snmp.ParseIndex(suffix)
		if len(parts) < 1 {
			continue
		}
		ifIdxToDescr[parts[0]] = fmt.Sprintf("%v", val)
	}

	// ── Step 4: Build (MAC → ONU) mapping ────────────────────────────────────
	var entries []repositories.MACTableEntry
	seen := map[string]bool{} // deduplicate (mac, port, slot)

	for portNum, macs := range portMACs {
		ifIdx, ok := portToIfIdx[portNum]
		if !ok {
			// Fallback: treat the FDB port value directly as an ifIndex.
			// VSOL EPON OLTs map customer MACs to ONU sub-interface ifIndexes
			// (e.g., FDB port 49 = ifIndex 49 = "EPON0/3:21"). These ONU
			// sub-interface ports do not appear in dot1dBasePortIfIndex — only
			// physical uplink ports (1-8) do.
			ifIdx = portNum
		}
		descr, ok := ifIdxToDescr[ifIdx]
		if !ok {
			continue
		}
		portIdx, onuSlot, err := parseONUIfDescr(descr)
		if err != nil {
			continue // not an ONU sub-interface (uplink port, CPU, etc.)
		}
		for _, mac := range macs {
			key := fmt.Sprintf("%d:%d:%s", portIdx, onuSlot, mac)
			if seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, repositories.MACTableEntry{
				MAC:     mac,
				PortIdx: portIdx,
				ONUSlot: onuSlot,
			})
		}
	}

	log.Printf("[SNMP-FDB] MAC table walk: %d raw FDB entries → %d ONU-MAC mappings", len(fdbWalk), len(entries))
	return entries
}

// walkCLIMACTable connects to the OLT via Telnet/SSH, runs
// "show mac-address-table", and returns the parsed ONU-level MAC entries.
// Used for OLTs whose SNMP FDB only exposes port-level MACs (e.g. Richerlink).
func (s *oltSyncService) walkCLIMACTable(olt *models.OLT) []repositories.MACTableEntry {
	cliPass, err := utils.Decrypt(olt.CLIPassword, s.jwtSecret)
	if err != nil {
		log.Printf("[CLI-MAC] OLT %s: decrypt CLI password: %v", olt.Name, err)
		return nil
	}
	enablePass := ""
	if olt.CLIEnablePassword != "" {
		enablePass, err = utils.Decrypt(olt.CLIEnablePassword, s.jwtSecret)
		if err != nil {
			log.Printf("[CLI-MAC] OLT %s: decrypt enable password: %v", olt.Name, err)
			return nil
		}
	}

	port := olt.CLIPort
	if port == 0 {
		if strings.ToLower(olt.CLIProtocol) == "ssh" {
			port = 22
		} else {
			port = 23
		}
	}

	client, err := telnet.New(olt.ManagementIP, port, 30*time.Second)
	if err != nil {
		log.Printf("[CLI-MAC] OLT %s: connect %s:%d: %v", olt.Name, olt.ManagementIP, port, err)
		return nil
	}
	defer client.Close()

	if err := client.Login(olt.CLIUsername, cliPass, enablePass); err != nil {
		log.Printf("[CLI-MAC] OLT %s: login failed: %v", olt.Name, err)
		return nil
	}

	output, err := client.RunCommand("show mac-address-table", 60*time.Second, "EPON#", "#")
	if err != nil {
		log.Printf("[CLI-MAC] OLT %s: show mac-address-table: %v", olt.Name, err)
		return nil
	}

	entries := telnet.ParseRicherlinkMACTable(output)
	log.Printf("[CLI-MAC] OLT %s: parsed %d ONU-MAC entries from CLI", olt.Name, len(entries))
	return entries
}
