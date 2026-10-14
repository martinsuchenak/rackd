package model

import (
	"fmt"
	"time"
)

type Credential struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	SNMPCommunity string    `json:"-" db:"snmp_community"` // Hidden from JSON output, encrypted in DB
	SNMPV3User    string    `json:"-" db:"snmp_v3_user"`   // Hidden from JSON output, encrypted in DB
	SNMPV3Auth    string    `json:"-" db:"snmp_v3_auth"`   // Hidden from JSON output, encrypted in DB
	SNMPV3Priv    string    `json:"-" db:"snmp_v3_priv"`   // Hidden from JSON output, encrypted in DB
	SSHUsername   string    `json:"ssh_username,omitempty"`
	SSHKeyID      string    `json:"-" db:"ssh_key_id"` // Hidden from JSON output, encrypted in DB
	DatacenterID  string    `json:"datacenter_id,omitempty"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CredentialInput is used for receiving credential data from API requests
// It has JSON tags to accept sensitive fields that are hidden in Credential output
type CredentialInput struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	SNMPCommunity string `json:"snmp_community"`
	SNMPV3User    string `json:"snmp_v3_user"`
	SNMPV3Auth    string `json:"snmp_v3_auth"`
	SNMPV3Priv    string `json:"snmp_v3_priv"`
	SSHUsername   string `json:"ssh_username"`
	SSHKeyID      string `json:"ssh_key_id"`
	DatacenterID  string `json:"datacenter_id"`
	Description   string `json:"description"`
}

// ToCredential converts CredentialInput to Credential
func (i *CredentialInput) ToCredential() *Credential {
	return &Credential{
		ID:            i.ID,
		Name:          i.Name,
		Type:          i.Type,
		SNMPCommunity: i.SNMPCommunity,
		SNMPV3User:    i.SNMPV3User,
		SNMPV3Auth:    i.SNMPV3Auth,
		SNMPV3Priv:    i.SNMPV3Priv,
		SSHUsername:   i.SSHUsername,
		SSHKeyID:      i.SSHKeyID,
		DatacenterID:  i.DatacenterID,
		Description:   i.Description,
	}
}

var ValidCredentialTypes = map[string]bool{
	"snmp_v2c":     true,
	"snmp_v3":      true,
	"ssh_key":      true,
	"ssh_password": true,
}

func (c *Credential) Validate() error {
	if !ValidCredentialTypes[c.Type] {
		return fmt.Errorf("invalid credential type: %s (must be one of: snmp_v2c, snmp_v3, ssh_key, ssh_password)", c.Type)
	}

	switch c.Type {
	case "snmp_v2c":
		if c.SNMPCommunity == "" {
			return fmt.Errorf("SNMP community required for snmp_v2c credentials")
		}
	case "snmp_v3":
		if c.SNMPV3User == "" {
			return fmt.Errorf("SNMP v3 user required for snmp_v3 credentials")
		}
		if c.SNMPV3Auth == "" {
			return fmt.Errorf("SNMP v3 auth password required for snmp_v3 credentials")
		}
		if c.SNMPV3Priv == "" {
			return fmt.Errorf("SNMP v3 privacy password required for snmp_v3 credentials")
		}
	case "ssh_key":
		if c.SSHUsername == "" {
			return fmt.Errorf("SSH username required for ssh_key credentials")
		}
		if c.SSHKeyID == "" {
			return fmt.Errorf("SSH key ID required for ssh_key credentials")
		}
	case "ssh_password":
		if c.SSHUsername == "" {
			return fmt.Errorf("SSH username required for ssh_password credentials")
		}
		if c.SSHKeyID == "" {
			return fmt.Errorf("SSH key ID or password reference required for ssh_password credentials")
		}
	}

	return nil
}
