// Package telnet provides a minimal Telnet client for OLT CLI access.
// It handles IAC negotiation inline and exposes a simple prompt-based
// read/write API suitable for scraping CLI command output.
package telnet

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// Telnet command bytes
const (
	iacByte  = byte(255)
	dontByte = byte(254)
	doByte   = byte(253)
	wontByte = byte(252)
	willByte = byte(251)
	sbByte   = byte(250)
	seByte   = byte(240)
)

// Client is a minimal Telnet session.
type Client struct {
	conn    net.Conn
	reader  *bufio.Reader
	timeout time.Duration
}

// New opens a TCP connection to host:port with the given connect timeout.
func New(host string, port int, timeout time.Duration) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("telnet connect %s: %w", addr, err)
	}
	return &Client{conn: conn, reader: bufio.NewReader(conn), timeout: timeout}, nil
}

// Close terminates the connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Send writes raw bytes to the connection.
func (c *Client) Send(s string) error {
	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err := c.conn.Write([]byte(s))
	return err
}

// ReadUntil accumulates bytes from the stream until any of the given prompt
// strings appears at the end of the buffer. IAC option negotiation is handled
// inline — all WILL offers are refused (DONT) and all DO requests are refused
// (WONT), so the device falls back to a plain byte stream.
//
// Returns (accumulated text, index of matched prompt, error).
// Returns index -1 on timeout or read error.
func (c *Client) ReadUntil(timeout time.Duration, prompts ...string) (string, int, error) {
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer c.conn.SetReadDeadline(time.Time{})

	var buf strings.Builder
	for {
		b, err := c.reader.ReadByte()
		if err != nil {
			return buf.String(), -1, fmt.Errorf("read: %w", err)
		}

		if b == iacByte {
			cmd, err := c.reader.ReadByte()
			if err != nil {
				return buf.String(), -1, err
			}
			switch cmd {
			case willByte, wontByte:
				opt, _ := c.reader.ReadByte()
				c.conn.Write([]byte{iacByte, dontByte, opt}) // refuse all WILL
			case doByte, dontByte:
				opt, _ := c.reader.ReadByte()
				c.conn.Write([]byte{iacByte, wontByte, opt}) // refuse all DO
			case sbByte:
				// Skip subnegotiation until IAC SE
				for {
					b2, _ := c.reader.ReadByte()
					if b2 == iacByte {
						b3, _ := c.reader.ReadByte()
						if b3 == seByte {
							break
						}
					}
				}
			}
			continue
		}

		buf.WriteByte(b)
		s := buf.String()
		for i, p := range prompts {
			if strings.HasSuffix(s, p) {
				return s, i, nil
			}
		}
	}
}

// Login performs the Richerlink EPON OLT login sequence:
//  1. Username prompt → send username
//  2. Password prompt → send password
//  3. User-mode prompt (EPON>) → send "enable"
//  4. Enable password prompt → send enablePassword (falls back to password if empty)
//  5. Privileged-mode prompt (EPON#) → ready
func (c *Client) Login(username, password, enablePassword string) error {
	// ── Username ──────────────────────────────────────────────────────────────
	if _, _, err := c.ReadUntil(15*time.Second, "Username:", "username:", "login:", "Login:"); err != nil {
		return fmt.Errorf("waiting for username prompt: %w", err)
	}
	if err := c.Send(username + "\r\n"); err != nil {
		return err
	}

	// ── Login password ────────────────────────────────────────────────────────
	if _, _, err := c.ReadUntil(10*time.Second, "Password:", "password:"); err != nil {
		return fmt.Errorf("waiting for password prompt: %w", err)
	}
	if err := c.Send(password + "\r\n"); err != nil {
		return err
	}

	// ── User-mode prompt (EPON>) ──────────────────────────────────────────────
	if _, _, err := c.ReadUntil(10*time.Second, ">"); err != nil {
		return fmt.Errorf("waiting for user-mode prompt: %w", err)
	}

	// ── Enter privileged mode ─────────────────────────────────────────────────
	if err := c.Send("enable\r\n"); err != nil {
		return err
	}

	// May get a Password prompt or jump straight to EPON# (no enable password)
	_, idx, err := c.ReadUntil(10*time.Second, "Password:", "password:", "#")
	if err != nil {
		return fmt.Errorf("after enable command: %w", err)
	}
	if idx == 2 {
		// Already at privileged prompt — no enable password required
		return nil
	}

	// Send enable password (fall back to login password if not set)
	ep := enablePassword
	if ep == "" {
		ep = password
	}
	if err := c.Send(ep + "\r\n"); err != nil {
		return err
	}
	if _, _, err := c.ReadUntil(10*time.Second, "#"); err != nil {
		return fmt.Errorf("waiting for privileged prompt: %w", err)
	}
	return nil
}

// RunCommand sends cmd + CRLF and returns all output up to the next prompt match.
func (c *Client) RunCommand(cmd string, timeout time.Duration, prompts ...string) (string, error) {
	if err := c.Send(cmd + "\r\n"); err != nil {
		return "", err
	}
	out, _, err := c.ReadUntil(timeout, prompts...)
	return out, err
}
