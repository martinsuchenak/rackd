package model

import (
	"testing"
)

func TestCredential_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cred    *Credential
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid snmp_v2c",
			cred: &Credential{
				Type:          "snmp_v2c",
				SNMPCommunity: "public",
			},
			wantErr: false,
		},
		{
			name: "invalid snmp_v2c - missing community",
			cred: &Credential{
				Type: "snmp_v2c",
			},
			wantErr: true,
			errMsg:  "SNMP community required",
		},
		{
			name: "valid snmp_v3",
			cred: &Credential{
				Type:       "snmp_v3",
				SNMPV3User: "admin",
				SNMPV3Auth: "authpass",
				SNMPV3Priv: "privpass",
			},
			wantErr: false,
		},
		{
			name: "invalid snmp_v3 - missing user",
			cred: &Credential{
				Type:       "snmp_v3",
				SNMPV3Auth: "authpass",
				SNMPV3Priv: "privpass",
			},
			wantErr: true,
			errMsg:  "SNMP v3 user required",
		},
		{
			name: "invalid snmp_v3 - missing auth",
			cred: &Credential{
				Type:       "snmp_v3",
				SNMPV3User: "admin",
				SNMPV3Priv: "privpass",
			},
			wantErr: true,
			errMsg:  "SNMP v3 auth password required",
		},
		{
			name: "invalid snmp_v3 - missing priv",
			cred: &Credential{
				Type:       "snmp_v3",
				SNMPV3User: "admin",
				SNMPV3Auth: "authpass",
			},
			wantErr: true,
			errMsg:  "SNMP v3 privacy password required",
		},
		{
			name: "valid ssh_key",
			cred: &Credential{
				Type:        "ssh_key",
				SSHUsername: "root",
				SSHKeyID:    "key123",
			},
			wantErr: false,
		},
		{
			name: "invalid ssh_key - missing username",
			cred: &Credential{
				Type:     "ssh_key",
				SSHKeyID: "key123",
			},
			wantErr: true,
			errMsg:  "SSH username required",
		},
		{
			name: "invalid ssh_key - missing key",
			cred: &Credential{
				Type:        "ssh_key",
				SSHUsername: "root",
			},
			wantErr: true,
			errMsg:  "SSH key ID required",
		},
		{
			name: "valid ssh_password",
			cred: &Credential{
				Type:        "ssh_password",
				SSHUsername: "admin",
				SSHKeyID:    "pass123",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			cred: &Credential{
				Type: "invalid_type",
			},
			wantErr: true,
			errMsg:  "invalid credential type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cred.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCredentialInput_ToCredential(t *testing.T) {
	input := &CredentialInput{
		ID:            "cred-123",
		Name:          "Test Cred",
		Type:          "snmp_v2c",
		SNMPCommunity: "public",
		DatacenterID:  "dc-1",
		Description:   "Test credential",
	}

	cred := input.ToCredential()

	if cred.ID != input.ID {
		t.Errorf("ID = %v, want %v", cred.ID, input.ID)
	}
	if cred.Name != input.Name {
		t.Errorf("Name = %v, want %v", cred.Name, input.Name)
	}
	if cred.Type != input.Type {
		t.Errorf("Type = %v, want %v", cred.Type, input.Type)
	}
	if cred.SNMPCommunity != input.SNMPCommunity {
		t.Errorf("SNMPCommunity = %v, want %v", cred.SNMPCommunity, input.SNMPCommunity)
	}
	if cred.DatacenterID != input.DatacenterID {
		t.Errorf("DatacenterID = %v, want %v", cred.DatacenterID, input.DatacenterID)
	}
	if cred.Description != input.Description {
		t.Errorf("Description = %v, want %v", cred.Description, input.Description)
	}
}

func TestValidCredentialTypes(t *testing.T) {
	expected := []string{"snmp_v2c", "snmp_v3", "ssh_key", "ssh_password"}
	for _, typ := range expected {
		if !ValidCredentialTypes[typ] {
			t.Errorf("expected %q to be valid credential type", typ)
		}
	}

	invalid := []string{"invalid", "http", "telnet"}
	for _, typ := range invalid {
		if ValidCredentialTypes[typ] {
			t.Errorf("expected %q to be invalid credential type", typ)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
