package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type ScanProfileService struct {
	store storage.ProfileStorage
	rbac  auditedPermissionStore
	// credential looks up a stored credential; nil when credential storage
	// isn't configured, in which case a profile can't name one.
	credential func(id string) (*model.Credential, error)
}

func NewScanProfileService(store storage.ProfileStorage, rbac auditedPermissionStore) *ScanProfileService {
	return &ScanProfileService{store: store, rbac: rbac}
}

func (s *ScanProfileService) List(ctx context.Context) ([]model.ScanProfile, error) {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "list"); err != nil {
		return nil, err
	}

	return s.store.List(ctx)
}

func (s *ScanProfileService) Create(ctx context.Context, profile *model.ScanProfile) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "create"); err != nil {
		return err
	}
	applyScanProfileDefaults(profile)
	if err := requireCredentialUseIfAdded(ctx, s.rbac, nil, profile); err != nil {
		return err
	}
	if err := s.checkCredentials(profile); err != nil {
		return err
	}

	if err := s.store.Create(ctx, profile); err != nil {
		return NormalizeError(err)
	}

	recordAudit(ctx, s.rbac, "create", "scan_profile", profile.ID, profile)
	return nil
}

func (s *ScanProfileService) Get(ctx context.Context, id string) (*model.ScanProfile, error) {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "read"); err != nil {
		return nil, err
	}

	profile, err := s.store.Get(ctx, id)
	if errors.Is(err, storage.ErrProfileNotFound) {
		return nil, ErrNotFound
	}
	return profile, err
}

func (s *ScanProfileService) Update(ctx context.Context, id string, profile *model.ScanProfile) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "update"); err != nil {
		return err
	}

	profile.ID = id
	applyScanProfileDefaults(profile)
	existing, err := s.store.Get(ctx, id)
	if errors.Is(err, storage.ErrProfileNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := requireCredentialUseIfAdded(ctx, s.rbac, existing, profile); err != nil {
		return err
	}
	if err := s.checkCredentials(profile); err != nil {
		return err
	}
	if err := s.store.Update(ctx, profile); err != nil {
		return NormalizeError(err)
	}

	recordAudit(ctx, s.rbac, "update", "scan_profile", id, profile)
	return nil
}

func (s *ScanProfileService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "delete"); err != nil {
		return err
	}

	err := s.store.Delete(ctx, id)
	if errors.Is(err, storage.ErrProfileNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	recordAudit(ctx, s.rbac, "delete", "scan_profile", id, nil)
	return nil
}

// Defaults for the scan profile fields the API lets a client leave out; they
// match the scan_profiles column defaults.
const (
	defaultScanTimeoutSec = 30
	defaultScanMaxWorkers = 10
)

func applyScanProfileDefaults(p *model.ScanProfile) {
	if p.TimeoutSec == 0 {
		p.TimeoutSec = defaultScanTimeoutSec
	}
	if p.MaxWorkers == 0 {
		p.MaxWorkers = defaultScanMaxWorkers
	}
}

// checkCredentials makes sure the credentials a profile names exist and are
// of the kind its probe needs: an SNMP credential for SNMP, an SSH one for SSH.
func (s *ScanProfileService) checkCredentials(p *model.ScanProfile) error {
	var lookupErr error
	check := func(field, id string, types ...string) *ValidationError {
		if id == "" {
			return nil
		}
		if s.credential == nil {
			return &ValidationError{Field: field, Message: "credential storage is not configured"}
		}
		cred, err := s.credential(id)
		if errors.Is(err, credentials.ErrCredentialNotFound) || errors.Is(err, ErrNotFound) || (err == nil && cred == nil) {
			return &ValidationError{Field: field, Message: "credential not found"}
		}
		if err != nil {
			lookupErr = err
			return nil
		}
		if !slices.Contains(types, cred.Type) {
			return &ValidationError{Field: field, Message: fmt.Sprintf("credential %q is %s; it must be one of %s", cred.Name, cred.Type, strings.Join(types, ", "))}
		}
		return nil
	}
	var errs ValidationErrors
	if e := check("snmp_credential_id", p.SNMPCredentialID, "snmp_v2c", "snmp_v3"); e != nil {
		errs = append(errs, *e)
	}
	if e := check("ssh_credential_id", p.SSHCredentialID, "ssh_key", "ssh_password"); e != nil {
		errs = append(errs, *e)
	}
	if lookupErr != nil {
		return lookupErr
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Using a stored credential sends it to every host a scan probes, so a
// caller who could otherwise only run scans could point one at a host they
// control and collect the secret. Attaching a credential to a profile, and
// running or scheduling a scan that uses one, therefore needs the right to
// manage credentials (credentials:update), not just to read their names.

// profileCredentialIDs returns the credentials a scan with the profile would
// use: a credential only counts when its probe is enabled.
func profileCredentialIDs(p *model.ScanProfile) (snmp, ssh string) {
	if p == nil {
		return "", ""
	}
	if p.EnableSNMP {
		snmp = p.SNMPCredentialID
	}
	if p.EnableSSH {
		ssh = p.SSHCredentialID
	}
	return snmp, ssh
}

// requireCredentialUse checks the caller may have scans use p's credentials.
func requireCredentialUse(ctx context.Context, rbac permissionStore, p *model.ScanProfile) error {
	if snmp, ssh := profileCredentialIDs(p); snmp == "" && ssh == "" {
		return nil
	}
	return requirePermission(ctx, rbac, "credentials", "update")
}

// requireCredentialUseIfAdded is requireCredentialUse for a profile edit:
// it applies only when the edit makes scans use a credential they didn't
// use before (a new credential, or a probe switched on), so editing other
// fields of a credentialed profile needs no extra right.
func requireCredentialUseIfAdded(ctx context.Context, rbac permissionStore, before, after *model.ScanProfile) error {
	bSNMP, bSSH := profileCredentialIDs(before)
	aSNMP, aSSH := profileCredentialIDs(after)
	if (aSNMP != "" && aSNMP != bSNMP) || (aSSH != "" && aSSH != bSSH) {
		return requirePermission(ctx, rbac, "credentials", "update")
	}
	return nil
}
