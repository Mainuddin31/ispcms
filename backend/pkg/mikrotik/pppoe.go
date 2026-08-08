package mikrotik

import (
	"strings"
	"time"
)

type PPPoESecretData struct {
	RouterOSID    string
	Username      string
	Password      string
	Profile       string
	Service       string
	LocalAddress  string
	RemoteAddress string
	CallerID      string
	Disabled      bool
	Comment       string
}

type PPPoESessionData struct {
	RouterOSID     string
	Username       string
	CurrentIP      string
	Uptime         string
	SessionID      string
	ConnectedSince *time.Time
}

func (c *Client) GetPPPoESecrets() ([]PPPoESecretData, error) {
	reply, err := c.Run("/ppp/secret/print")
	if err != nil {
		return nil, err
	}

	var secrets []PPPoESecretData
	for _, re := range reply.Re {
		s := PPPoESecretData{
			RouterOSID:    re.Map[".id"],
			Username:      re.Map["name"],
			Password:      re.Map["password"],
			Profile:       re.Map["profile"],
			Service:       re.Map["service"],
			LocalAddress:  re.Map["local-address"],
			RemoteAddress: re.Map["remote-address"],
			CallerID:      re.Map["caller-id"],
			Disabled:      re.Map["disabled"] == "true",
			Comment:       re.Map["comment"],
		}
		secrets = append(secrets, s)
	}
	return secrets, nil
}

type PPPoEProfileData struct {
	RouterOSID string
	Name       string
}

func (c *Client) GetPPPoEProfiles() ([]PPPoEProfileData, error) {
	reply, err := c.Run("/ppp/profile/print")
	if err != nil {
		return nil, err
	}
	var profiles []PPPoEProfileData
	for _, re := range reply.Re {
		profiles = append(profiles, PPPoEProfileData{
			RouterOSID: re.Map[".id"],
			Name:       re.Map["name"],
		})
	}
	return profiles, nil
}

func (c *Client) GetActiveSessions() ([]PPPoESessionData, error) {
	reply, err := c.Run("/ppp/active/print")
	if err != nil {
		// Older RouterOS versions respond with !empty when there are no active sessions.
		// Treat this as a valid empty result rather than an error.
		if strings.Contains(err.Error(), "!empty") {
			return []PPPoESessionData{}, nil
		}
		return nil, err
	}

	var sessions []PPPoESessionData
	for _, re := range reply.Re {
		sessions = append(sessions, PPPoESessionData{
			RouterOSID: re.Map[".id"],
			Username:   re.Map["name"],
			CurrentIP:  re.Map["address"],
			Uptime:     re.Map["uptime"],
			SessionID:  re.Map["session-id"],
		})
	}
	return sessions, nil
}
