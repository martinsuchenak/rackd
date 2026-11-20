package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/pkg/model"
	"github.com/martinsuchenak/xdal/pkg/sqlite"
)

var ErrProfileNotFound = errors.New("scan profile not found")

type ProfileStorage interface {
	Create(ctx context.Context, profile *model.ScanProfile) error
	Update(ctx context.Context, profile *model.ScanProfile) error
	Get(ctx context.Context, id string) (*model.ScanProfile, error)
	List(ctx context.Context) ([]model.ScanProfile, error)
	Delete(ctx context.Context, id string) error
}

type SQLiteProfileStorage struct {
	q portableDB
}

func NewSQLiteProfileStorage(db *sql.DB) (*SQLiteProfileStorage, error) {
	if err := RequireTables(db, "scan_profiles"); err != nil {
		return nil, err
	}
	return &SQLiteProfileStorage{q: portableDB{x: sqlite.NewSQLiteDB(db, nil)}}, nil
}

// Profiles returns the scan profiles store on this storage's database, whichever
// backend it is.
func (s *SQLiteStorage) Profiles() *SQLiteProfileStorage {
	return &SQLiteProfileStorage{q: s.q}
}

func (s *SQLiteProfileStorage) Create(ctx context.Context, profile *model.ScanProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	if profile.ID == "" {
		profile.ID = uuid.Must(uuid.NewV7()).String()
	}
	now := nowUTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	portsJSON, _ := json.Marshal(profile.Ports)
	_, err := s.q.ExecContext(context.Background(), `
		INSERT INTO scan_profiles (id, name, scan_type, ports, enable_snmp, enable_ssh, snmp_credential_id, ssh_credential_id, timeout_sec, max_workers, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Name, profile.ScanType, string(portsJSON), profile.EnableSNMP, profile.EnableSSH,
		nullString(profile.SNMPCredentialID), nullString(profile.SSHCredentialID), profile.TimeoutSec, profile.MaxWorkers, profile.Description, profile.CreatedAt, profile.UpdatedAt)
	return err
}

func (s *SQLiteProfileStorage) Update(ctx context.Context, profile *model.ScanProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	profile.UpdatedAt = nowUTC()

	portsJSON, _ := json.Marshal(profile.Ports)
	res, err := s.q.ExecContext(context.Background(), `
		UPDATE scan_profiles SET name=?, scan_type=?, ports=?, enable_snmp=?, enable_ssh=?, snmp_credential_id=?, ssh_credential_id=?,
		       timeout_sec=?, max_workers=?, description=?, updated_at=?
		WHERE id=?
	`, profile.Name, profile.ScanType, string(portsJSON), profile.EnableSNMP, profile.EnableSSH,
		nullString(profile.SNMPCredentialID), nullString(profile.SSHCredentialID), profile.TimeoutSec, profile.MaxWorkers, profile.Description, profile.UpdatedAt, profile.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (s *SQLiteProfileStorage) Get(ctx context.Context, id string) (*model.ScanProfile, error) {
	row := s.q.QueryRowContext(ctx, `SELECT id, name, scan_type, ports, enable_snmp, enable_ssh, snmp_credential_id, ssh_credential_id, timeout_sec, max_workers, description, created_at, updated_at FROM scan_profiles WHERE id=?`, id)
	return s.scanProfile(row)
}

func (s *SQLiteProfileStorage) List(ctx context.Context) ([]model.ScanProfile, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT id, name, scan_type, ports, enable_snmp, enable_ssh, snmp_credential_id, ssh_credential_id, timeout_sec, max_workers, description, created_at, updated_at FROM scan_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var profiles []model.ScanProfile
	for rows.Next() {
		var p model.ScanProfile
		var portsJSON sql.NullString
		var description, snmpCred, sshCred sql.NullString
		err := rows.Scan(&p.ID, &p.Name, &p.ScanType, &portsJSON, &p.EnableSNMP, &p.EnableSSH, &snmpCred, &sshCred, &p.TimeoutSec, &p.MaxWorkers, &description, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if portsJSON.Valid {
			_ = json.Unmarshal([]byte(portsJSON.String), &p.Ports)
		}
		p.Description = description.String
		p.SNMPCredentialID, p.SSHCredentialID = snmpCred.String, sshCred.String
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (s *SQLiteProfileStorage) Delete(ctx context.Context, id string) error {
	res, err := s.q.ExecContext(ctx, `DELETE FROM scan_profiles WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (s *SQLiteProfileStorage) scanProfile(row *sql.Row) (*model.ScanProfile, error) {
	var p model.ScanProfile
	var portsJSON sql.NullString
	var description, snmpCred, sshCred sql.NullString
	err := row.Scan(&p.ID, &p.Name, &p.ScanType, &portsJSON, &p.EnableSNMP, &p.EnableSSH, &snmpCred, &sshCred, &p.TimeoutSec, &p.MaxWorkers, &description, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	if portsJSON.Valid {
		_ = json.Unmarshal([]byte(portsJSON.String), &p.Ports)
	}
	p.Description = description.String
	p.SNMPCredentialID, p.SSHCredentialID = snmpCred.String, sshCred.String
	return &p, nil
}
