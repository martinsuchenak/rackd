package service

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/storage/storagetest"
	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestScanProfileCredentialValidation(t *testing.T) {
	store := storagetest.New(t)
	profiles, err := storage.NewSQLiteProfileStorage(store.DB())
	if err != nil {
		t.Fatal(err)
	}
	creds := map[string]*model.Credential{
		"snmp": {ID: "snmp", Name: "snmp", Type: "snmp_v3"},
		"ssh":  {ID: "ssh", Name: "ssh", Type: "ssh_key"},
	}
	svc := NewScanProfileService(profiles, store)
	svc.credential = func(id string) (*model.Credential, error) {
		if c, ok := creds[id]; ok {
			return c, nil
		}
		return nil, errors.New("not found")
	}
	ctx := SystemContext(context.Background(), "test")

	ok := &model.ScanProfile{Name: "ok", ScanType: "full", EnableSNMP: true, SNMPCredentialID: "snmp", EnableSSH: true, SSHCredentialID: "ssh"}
	if err := svc.Create(ctx, ok); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}

	for _, c := range []struct {
		name  string
		p     *model.ScanProfile
		field string
	}{
		{"missing", &model.ScanProfile{Name: "m", ScanType: "full", SNMPCredentialID: "nope"}, "snmp_credential_id"},
		{"ssh as snmp", &model.ScanProfile{Name: "w1", ScanType: "full", SNMPCredentialID: "ssh"}, "snmp_credential_id"},
		{"snmp as ssh", &model.ScanProfile{Name: "w2", ScanType: "full", SSHCredentialID: "snmp"}, "ssh_credential_id"},
	} {
		err := svc.Create(ctx, c.p)
		var verrs ValidationErrors
		if !errors.As(err, &verrs) || len(verrs) != 1 || verrs[0].Field != c.field {
			t.Errorf("%s: got %v, want a validation error on %s", c.name, err, c.field)
		}
	}

	// Update validates too.
	ok.SSHCredentialID = "snmp"
	if err := svc.Update(ctx, ok.ID, ok); err == nil {
		t.Error("update with a wrong-type credential was accepted")
	}
}

// recordingScanner records how a scan was started.
type recordingScanner struct {
	discovery.AdvancedScanner
	profile  *model.ScanProfile
	scanType string
}

func (r *recordingScanner) Scan(_ context.Context, n *model.Network, scanType string) (*model.DiscoveryScan, error) {
	r.scanType = scanType
	return &model.DiscoveryScan{NetworkID: n.ID, ScanType: scanType}, nil
}

func (r *recordingScanner) ScanAdvanced(_ context.Context, n *model.Network, p *model.ScanProfile, _, _ string) (*model.DiscoveryScan, error) {
	r.profile = p
	return &model.DiscoveryScan{NetworkID: n.ID, ScanType: p.ScanType}, nil
}

func TestStartScanWithProfile(t *testing.T) {
	store := storagetest.New(t)
	profiles, err := storage.NewSQLiteProfileStorage(store.DB())
	if err != nil {
		t.Fatal(err)
	}
	ctx := SystemContext(context.Background(), "test")
	network := &model.Network{Name: "scan-net", Subnet: "10.20.0.0/24"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}
	scanner := &recordingScanner{}
	svc := NewDiscoveryService(store, scanner)
	svc.profiles = profiles

	if _, err := svc.StartScanWithProfile(ctx, network.ID, "deep", ""); err != nil || scanner.scanType != "deep" || scanner.profile != nil {
		t.Fatalf("plain scan: err %v, type %q, profile %v", err, scanner.scanType, scanner.profile)
	}

	// default-full is seeded by the migrations.
	scan, err := svc.StartScanWithProfile(ctx, network.ID, "quick", "default-full")
	if err != nil {
		t.Fatal(err)
	}
	if scanner.profile == nil || scanner.profile.ID != "default-full" || scan.ScanType != "full" {
		t.Fatalf("profile scan: profile %+v, scan type %q", scanner.profile, scan.ScanType)
	}

	var verrs ValidationErrors
	if _, err := svc.StartScanWithProfile(ctx, network.ID, "", "missing"); !errors.As(err, &verrs) || verrs[0].Field != "profile_id" {
		t.Fatalf("unknown profile: got %v", err)
	}
}
