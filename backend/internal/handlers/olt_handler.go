package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
	"github.com/ispcms/backend/pkg/utils"
)

// ── SNMP Profile Handler ──────────────────────────────────────────────────────

type SNMPProfileHandler struct{ svc services.SNMPProfileService }

func NewSNMPProfileHandler(svc services.SNMPProfileService) *SNMPProfileHandler {
	return &SNMPProfileHandler{svc: svc}
}

func (h *SNMPProfileHandler) List(c *fiber.Ctx) error {
	profiles, err := h.svc.List()
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, profiles)
}

func (h *SNMPProfileHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	p, err := h.svc.Get(id)
	if err != nil {
		return utils.NotFound(c, "profile not found")
	}
	return utils.OK(c, p)
}

func (h *SNMPProfileHandler) Create(c *fiber.Ctx) error {
	var p struct {
		Name        string            `json:"name"`
		Vendor      string            `json:"vendor"`
		Technology  string            `json:"technology"`
		OIDMap      map[string]string `json:"oid_map"`
		Description string            `json:"description"`
	}
	if err := c.BodyParser(&p); err != nil {
		return utils.BadRequest(c, "invalid body")
	}
	profile := &services.CreateSNMPProfileInput{
		Name: p.Name, Vendor: p.Vendor, Technology: p.Technology,
		OIDMap: p.OIDMap, Description: p.Description,
	}
	created, err := h.svc.CreateFromInput(profile)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": created})
}

func (h *SNMPProfileHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	var p struct {
		Name        string            `json:"name"`
		Vendor      string            `json:"vendor"`
		Technology  string            `json:"technology"`
		OIDMap      map[string]string `json:"oid_map"`
		Description string            `json:"description"`
	}
	if err := c.BodyParser(&p); err != nil {
		return utils.BadRequest(c, "invalid body")
	}
	profile := &services.CreateSNMPProfileInput{
		Name: p.Name, Vendor: p.Vendor, Technology: p.Technology,
		OIDMap: p.OIDMap, Description: p.Description,
	}
	updated, err := h.svc.UpdateFromInput(id, profile)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OK(c, updated)
}

func (h *SNMPProfileHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	if err := h.svc.Delete(id); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OK(c, fiber.Map{"message": "deleted"})
}

// ── OLT Handler ───────────────────────────────────────────────────────────────

type OLTHandler struct {
	svc     services.OLTService
	syncSvc services.OLTSyncService
	cfg     oltHandlerConfig
}

type oltHandlerConfig struct{ JWTSecret string }

func NewOLTHandler(svc services.OLTService, syncSvc services.OLTSyncService, jwtSecret string) *OLTHandler {
	return &OLTHandler{svc: svc, syncSvc: syncSvc, cfg: oltHandlerConfig{JWTSecret: jwtSecret}}
}

func (h *OLTHandler) List(c *fiber.Ctx) error {
	olts, err := h.svc.List(repositories.OLTFilter{
		Search: c.Query("search"),
		Status: c.Query("status"),
	})
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, olts)
}

func (h *OLTHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	o, err := h.svc.Get(id)
	if err != nil {
		return utils.NotFound(c, "OLT not found")
	}
	// mask passwords
	o.V3AuthPassword = ""
	o.V3PrivPassword = ""
	return utils.OK(c, o)
}

func (h *OLTHandler) Create(c *fiber.Ctx) error {
	input, err := parseOLTBody(c)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	o, err := h.svc.Create(*input, h.cfg.JWTSecret)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	o.V3AuthPassword = ""
	o.V3PrivPassword = ""
	return c.Status(201).JSON(fiber.Map{"success": true, "data": o})
}

func (h *OLTHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	input, err := parseOLTBody(c)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	o, err := h.svc.Update(id, *input, h.cfg.JWTSecret)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	o.V3AuthPassword = ""
	o.V3PrivPassword = ""
	return utils.OK(c, o)
}

func (h *OLTHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	userID, _ := c.Locals("userID").(uuid.UUID)
	if err := h.svc.Delete(id, userID); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OK(c, fiber.Map{"message": "deleted"})
}

func (h *OLTHandler) Sync(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	log, err := h.syncSvc.Sync(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error(), "log": log})
	}
	return utils.OK(c, log)
}

func (h *OLTHandler) TestConnection(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	if err := h.syncSvc.TestConnection(id); err != nil {
		return c.Status(200).JSON(fiber.Map{"success": false, "reachable": false, "error": err.Error()})
	}
	return utils.OK(c, fiber.Map{"reachable": true})
}

func (h *OLTHandler) TestConnectionRaw(c *fiber.Ctx) error {
	var body struct {
		IP        string `json:"ip"`
		Community string `json:"community"`
		Port      int    `json:"port"`
		Version   string `json:"version"`
	}
	if err := c.BodyParser(&body); err != nil {
		return utils.BadRequest(c, "invalid body")
	}
	if body.Port == 0 {
		body.Port = 161
	}
	if body.Version == "" {
		body.Version = "v2c"
	}
	if err := h.syncSvc.TestConnectionRaw(body.IP, body.Community, body.Port, body.Version); err != nil {
		return c.Status(200).JSON(fiber.Map{"success": false, "reachable": false, "error": err.Error()})
	}
	return utils.OK(c, fiber.Map{"reachable": true})
}

func (h *OLTHandler) Stats(c *fiber.Ctx) error {
	stats, err := h.svc.Stats()
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, stats)
}

func (h *OLTHandler) SyncLogs(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	limit := c.QueryInt("limit", 20)
	logs, err := h.svc.GetSyncLogs(id, limit)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, logs)
}

func (h *OLTHandler) RecentSyncLogs(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	logs, err := h.svc.RecentSyncLogs(limit)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, logs)
}

func (h *OLTHandler) PONPorts(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	ports, err := h.svc.ListPONPorts(id)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, ports)
}

// ── ONU Handler ───────────────────────────────────────────────────────────────

type ONUHandler struct{ svc services.ONUService }

func NewONUHandler(svc services.ONUService) *ONUHandler { return &ONUHandler{svc: svc} }

func (h *ONUHandler) List(c *fiber.Ctx) error {
	f := repositories.ONUFilter{
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Unlinked: c.Query("unlinked") == "true",
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 50),
	}
	if id := c.Query("olt_id"); id != "" {
		if uid, err := uuid.Parse(id); err == nil {
			f.OLTID = &uid
		}
	}
	if id := c.Query("pon_port_id"); id != "" {
		if uid, err := uuid.Parse(id); err == nil {
			f.PONPortID = &uid
		}
	}
	onus, total, err := h.svc.List(f)
	if err != nil {
		return utils.InternalError(c, err)
	}
	pageSize := f.PageSize
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": onus, "total": total,
		"page": f.Page, "page_size": pageSize, "total_pages": totalPages,
	})
}

func (h *ONUHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	o, err := h.svc.Get(id)
	if err != nil {
		return utils.NotFound(c, "ONU not found")
	}
	return utils.OK(c, o)
}

func (h *ONUHandler) Link(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid id")
	}
	var body struct {
		InternetAccountID *string `json:"internet_account_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return utils.BadRequest(c, "invalid body")
	}
	var accountID *uuid.UUID
	if body.InternetAccountID != nil && *body.InternetAccountID != "" {
		uid, err := uuid.Parse(*body.InternetAccountID)
		if err != nil {
			return utils.BadRequest(c, "invalid internet_account_id")
		}
		accountID = &uid
	}
	if err := h.svc.Link(id, accountID); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OK(c, fiber.Map{"message": "linked"})
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func parseOLTBody(c *fiber.Ctx) (*services.CreateOLTInput, error) {
	var body struct {
		Name           string `json:"name"`
		Vendor         string `json:"vendor"`
		Model          string `json:"model"`
		SNMPProfileID  string `json:"snmp_profile_id"`
		ManagementIP   string `json:"management_ip"`
		SNMPVersion    string `json:"snmp_version"`
		SNMPPort       int    `json:"snmp_port"`
		Timeout        int    `json:"timeout"`
		Retries        int    `json:"retries"`
		Community      string `json:"community"`
		V3Username     string `json:"v3_username"`
		V3AuthProtocol string `json:"v3_auth_protocol"`
		V3AuthPassword string `json:"v3_auth_password"`
		V3PrivProtocol string `json:"v3_priv_protocol"`
		V3PrivPassword string `json:"v3_priv_password"`
		POP            string `json:"pop"`
		Rack           string `json:"rack"`
		Cabinet        string `json:"cabinet"`
		Description    string `json:"description"`
		Status         string `json:"status"`
		SyncInterval   int    `json:"sync_interval"`
	}
	if err := c.BodyParser(&body); err != nil {
		return nil, err
	}
	profileID, err := uuid.Parse(body.SNMPProfileID)
	if err != nil {
		return nil, fiber.NewError(400, "invalid snmp_profile_id")
	}
	return &services.CreateOLTInput{
		Name: body.Name, Vendor: body.Vendor, Model: body.Model,
		SNMPProfileID: profileID, ManagementIP: body.ManagementIP,
		SNMPVersion: body.SNMPVersion, SNMPPort: body.SNMPPort,
		Timeout: body.Timeout, Retries: body.Retries,
		Community: body.Community,
		V3Username: body.V3Username, V3AuthProtocol: body.V3AuthProtocol,
		V3AuthPassword: body.V3AuthPassword, V3PrivProtocol: body.V3PrivProtocol,
		V3PrivPassword: body.V3PrivPassword,
		POP: body.POP, Rack: body.Rack, Cabinet: body.Cabinet,
		Description: body.Description, Status: body.Status,
		SyncInterval: body.SyncInterval,
	}, nil
}
