// Package snmp provides a thin wrapper around gosnmp for OLT synchronization.
// It supports SNMP v2c and v3 (authPriv / authNoPriv / noAuthNoPriv).
package snmp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// Client wraps a gosnmp session.
type Client struct {
	g          *gosnmp.GoSNMP
	useGetNext bool // if true, always use GETNEXT Walk (never GETBULK)
}

// Config holds connection parameters resolved from an OLT record.
type Config struct {
	Host           string
	Port           uint16
	Version        string // "v2c" | "v3"
	Community      string
	Timeout        time.Duration
	Retries        int
	V3Username     string
	V3AuthProtocol string // MD5 | SHA
	V3AuthPassword string
	V3PrivProtocol string // DES | AES
	V3PrivPassword string
}

// New creates and connects an SNMP client.
func New(cfg Config) (*Client, error) {
	g := &gosnmp.GoSNMP{
		Target:    cfg.Host,
		Port:      cfg.Port,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
		MaxOids:   gosnmp.MaxOids,
	}

	switch cfg.Version {
	case "v3":
		g.Version = gosnmp.Version3
		g.SecurityModel = gosnmp.UserSecurityModel
		msgFlag := gosnmp.NoAuthNoPriv
		authProto := gosnmp.NoAuth
		privProto := gosnmp.NoPriv

		switch strings.ToUpper(cfg.V3AuthProtocol) {
		case "MD5":
			authProto = gosnmp.MD5
			msgFlag = gosnmp.AuthNoPriv
		case "SHA":
			authProto = gosnmp.SHA
			msgFlag = gosnmp.AuthNoPriv
		}
		switch strings.ToUpper(cfg.V3PrivProtocol) {
		case "DES":
			privProto = gosnmp.DES
			msgFlag = gosnmp.AuthPriv
		case "AES":
			privProto = gosnmp.AES
			msgFlag = gosnmp.AuthPriv
		}

		g.MsgFlags = msgFlag
		g.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 cfg.V3Username,
			AuthenticationProtocol:  authProto,
			AuthenticationPassphrase: cfg.V3AuthPassword,
			PrivacyProtocol:         privProto,
			PrivacyPassphrase:       cfg.V3PrivPassword,
		}
	default: // v2c
		g.Version = gosnmp.Version2c
		g.Community = cfg.Community
	}

	if err := g.Connect(); err != nil {
		return nil, fmt.Errorf("snmp connect: %w", err)
	}
	return &Client{g: g}, nil
}

// Close terminates the SNMP connection.
func (c *Client) Close() {
	if c.g.Conn != nil {
		c.g.Conn.Close()
	}
}

// SetContext attaches a context to the SNMP session so that Walk/BulkWalk
// operations are cancelled when the context deadline is exceeded.
func (c *Client) SetContext(ctx context.Context) {
	c.g.Context = ctx
}

// SetUseGetNext forces all Walk calls to use GETNEXT (RFC 1157) instead of
// GETBULK. Use for devices that do not support GETBULK (e.g. some Richerlink
// firmware versions) — using GETBULK on those devices causes request timeouts.
func (c *Client) SetUseGetNext(v bool) {
	c.useGetNext = v
}

// TestConnection sends a sysDescr GET and returns success/error.
func (c *Client) TestConnection() error {
	result, err := c.g.Get([]string{"1.3.6.1.2.1.1.1.0"})
	if err != nil {
		return fmt.Errorf("snmp test failed: %w", err)
	}
	if result.Error != gosnmp.NoError {
		return fmt.Errorf("snmp error: %s", result.Error.String())
	}
	return nil
}

// GetString fetches a single scalar OID and returns it as a string.
func (c *Client) GetString(oid string) (string, error) {
	result, err := c.g.Get([]string{oid})
	if err != nil {
		return "", err
	}
	if len(result.Variables) == 0 {
		return "", nil
	}
	return valueToString(result.Variables[0]), nil
}

// Walk performs an SNMP subtree walk on the given base OID.
// Returns a map of OID-suffix → raw value.
// The suffix is the part after baseOID + ".".
// If SetUseGetNext(true) was called, it uses GETNEXT only (no GETBULK).
// Otherwise it tries GETBULK first and falls back to GETNEXT on error.
func (c *Client) Walk(baseOID string) (map[string]interface{}, error) {
	cb := func(result map[string]interface{}) func(gosnmp.SnmpPDU) error {
		return func(pdu gosnmp.SnmpPDU) error {
			suffix := strings.TrimPrefix(pdu.Name, "."+baseOID)
			suffix = strings.TrimPrefix(suffix, baseOID)
			suffix = strings.TrimPrefix(suffix, ".")
			result[suffix] = pduValue(pdu)
			return nil
		}
	}

	if c.useGetNext {
		// GETNEXT-only path: compatible with devices that don't support GETBULK
		result := make(map[string]interface{})
		if err := c.g.Walk(baseOID, cb(result)); err != nil {
			return nil, fmt.Errorf("walk %s: %w", baseOID, err)
		}
		return result, nil
	}

	// Default: try GETBULK first, fall back to GETNEXT.
	// After a BulkWalk timeout the UDP socket is in a broken state — reconnect
	// before retrying with GETNEXT so the fallback has a clean connection.
	result := make(map[string]interface{})
	if err := c.g.BulkWalk(baseOID, cb(result)); err != nil {
		// Reconnect to get a fresh socket; preserve the context that was set.
		savedCtx := c.g.Context
		if c.g.Conn != nil {
			c.g.Conn.Close()
		}
		if connErr := c.g.Connect(); connErr != nil {
			return nil, fmt.Errorf("walk %s: %w", baseOID, err)
		}
		c.g.Context = savedCtx

		result2 := make(map[string]interface{})
		if err2 := c.g.Walk(baseOID, cb(result2)); err2 != nil {
			return nil, fmt.Errorf("walk %s (getnext fallback): %w", baseOID, err2)
		}
		return result2, nil
	}
	return result, nil
}

// SysInfo returns basic system information.
func (c *Client) SysInfo() (name, descr string, err error) {
	result, err := c.g.Get([]string{
		"1.3.6.1.2.1.1.5.0",
		"1.3.6.1.2.1.1.1.0",
	})
	if err != nil {
		return "", "", err
	}
	for _, v := range result.Variables {
		switch {
		case strings.HasSuffix(v.Name, "1.5.0"):
			name = valueToString(v)
		case strings.HasSuffix(v.Name, "1.1.0"):
			descr = valueToString(v)
		}
	}
	return name, descr, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func pduValue(pdu gosnmp.SnmpPDU) interface{} {
	switch pdu.Type {
	case gosnmp.OctetString:
		b, ok := pdu.Value.([]byte)
		if ok {
			// Try to see if it's printable
			if isPrintable(b) {
				return string(b)
			}
			return macBytesToString(b)
		}
		return fmt.Sprintf("%v", pdu.Value)
	case gosnmp.Integer, gosnmp.Gauge32, gosnmp.Counter32, gosnmp.Counter64,
		gosnmp.TimeTicks, gosnmp.Uinteger32:
		return gosnmp.ToBigInt(pdu.Value).Int64()
	default:
		return fmt.Sprintf("%v", pdu.Value)
	}
}

func valueToString(v gosnmp.SnmpPDU) string {
	switch vv := v.Value.(type) {
	case []byte:
		if isPrintable(vv) {
			return strings.TrimSpace(string(vv))
		}
		return macBytesToString(vv)
	case string:
		return strings.TrimSpace(vv)
	default:
		return fmt.Sprintf("%v", vv)
	}
}

func isPrintable(b []byte) bool {
	for _, c := range b {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

func macBytesToString(b []byte) string {
	if len(b) == 6 {
		return net.HardwareAddr(b).String()
	}
	parts := make([]string, len(b))
	for i, bb := range b {
		parts[i] = fmt.Sprintf("%02x", bb)
	}
	return strings.Join(parts, ":")
}

// ParseIndex splits a dot-separated SNMP index suffix into integer parts.
// e.g. "1.3" → [1, 3]
func ParseIndex(suffix string) []int {
	parts := strings.Split(suffix, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.Atoi(p); err == nil {
			result = append(result, n)
		}
	}
	return result
}
