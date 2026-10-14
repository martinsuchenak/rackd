package model

import "time"

type CredentialResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	DatacenterID string `json:"datacenter_id,omitempty"`
	Description  string `json:"description,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	HasCommunity bool   `json:"has_community,omitempty"`
	HasAuth      bool   `json:"has_auth,omitempty"`
	HasPriv      bool   `json:"has_priv,omitempty"`
	HasKeyRef    bool   `json:"has_key_ref,omitempty"`
	HasUsername  bool   `json:"has_username,omitempty"`
}

func CredentialToResponse(c *Credential) CredentialResponse {
	return CredentialResponse{
		ID:           c.ID,
		Name:         c.Name,
		Type:         c.Type,
		DatacenterID: c.DatacenterID,
		Description:  c.Description,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
		HasCommunity: c.SNMPCommunity != "",
		HasAuth:      c.SNMPV3Auth != "",
		HasPriv:      c.SNMPV3Priv != "",
		HasKeyRef:    c.SSHKeyID != "",
		HasUsername:  c.SSHUsername != "",
	}
}

func (c *Credential) ToResponse() CredentialResponse {
	return CredentialToResponse(c)
}
