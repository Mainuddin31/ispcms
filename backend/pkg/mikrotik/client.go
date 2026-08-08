package mikrotik

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-routeros/routeros/v3"
)

type Client struct {
	conn *routeros.Client
	host string
	port int
}

func NewClient(host string, port int, username, password string) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := routeros.Dial(addr, username, password)
	if err != nil {
		return nil, fmt.Errorf("%s", cleanError(err))
	}
	return &Client{conn: client, host: host, port: port}, nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) Run(sentence ...string) (*routeros.Reply, error) {
	return c.conn.RunArgs(sentence)
}

func TestConnection(host string, port int, username, password string) error {
	client, err := NewClient(host, port, username, password)
	if err != nil {
		return err
	}
	defer client.Close()
	_, err = client.Run("/system/identity/print")
	if err != nil {
		return errors.New(cleanError(err))
	}
	return nil
}

// cleanError removes the go-routeros library artifact "; close %!w(<nil>)"
// and other noise from error messages, keeping only the meaningful part.
func cleanError(err error) string {
	msg := err.Error()
	// Strip "; close ..." suffix appended by the library on auth failure
	if idx := strings.Index(msg, "; close"); idx != -1 {
		msg = msg[:idx]
	}
	// Strip %!w(<nil>) anywhere it appears
	msg = strings.ReplaceAll(msg, "%!w(<nil>)", "")
	msg = strings.TrimSpace(strings.TrimRight(msg, ";"))
	return msg
}
