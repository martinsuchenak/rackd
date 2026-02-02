package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
)

var ErrProfileNotFound = errors.New("scan profile not found")

type ProfileStorage interface {
	Create(profile *model.ScanProfile) error
	Update(profile *model.ScanProfile) error
	Get(id string) (*model.ScanProfile, error)
	List() ([]model.ScanProfile, error)
	Delete(id string) error
}

type SQLiteProfileStorage struct {
	db *sql.DB
}

func NewSQLiteProfileStorage(db *sql.DB) (*SQLiteProfileStorage, error) {
	s := &SQLiteProfileStorage{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	if err := s.seedDefaults(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteProfileStorage) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS scan_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			scan_type TEXT NOT NULL,
			ports TEXT,
			enable_snmp INTEGER DEFAULT 0,
			enable_ssh INTEGER DEFAULT 0,
			timeout_sec INTEGER DEFAULT 30,
			max_workers INTEGER DEFAULT 10,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *SQLiteProfileStorage) seedDefaults() error {
	defaults := []model.ScanProfile{
		{ID: "default-quick", Name: "Quick Scan", ScanType: "quick", Ports: []int{22, 80, 443, 3389}, EnableSNMP: false, EnableSSH: false, TimeoutSec: 10, MaxWorkers: 50},
		{ID: "default-full", Name: "Full Scan", ScanType: "full", Ports: []int{22, 80, 443, 3389, 8080, 8443}, EnableSNMP: true, EnableSSH: false, TimeoutSec: 30, MaxWorkers: 20},
		{ID: "default-deep", Name: "Deep Scan", ScanType: "deep", Ports: []int{22, 80, 443, 3389, 8080, 8443, 3306, 5432}, EnableSNMP: true, EnableSSH: true, TimeoutSec: 60, MaxWorkers: 10},
	}
	for _, p := range defaults {
		existing, _ := s.Get(p.ID)
		if existing == nil {
			s.Create(&p)
		}
	}
	return nil
}

func (s *SQLiteProfileStorage) Create(profile *model.ScanProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	if profile.ID == "" {
		profile.ID = uuid.Must(uuid.NewV7()).String()
	}
	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	portsJSON, _ := json.Marshal(profile.Ports)
	_, err := s.db.Exec(`
		INSERT INTO scan_profiles (id, name, scan_type, ports, enable_snmp, enable_ssh, timeout_sec, max_workers, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Name, profile.ScanType, string(portsJSON), profile.EnableSNMP, profile.EnableSSH, profile.TimeoutSec, profile.MaxWorkers, profile.Description, profile.CreatedAt, profile.UpdatedAt)
	return err
}

func (s *SQLiteProfileStorage) Update(profile *model.ScanProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	profile.UpdatedAt = time.Now()

	portsJSON, _ := json.Marshal(profile.Ports)
	res, err := s.db.Exec(`
		UPDATE scan_profiles SET name=?, scan_type=?, ports=?, enable_snmp=?, enable_ssh=?, timeout_sec=?, max_workers=?, description=?, updated_at=?
		WHERE id=?
	`, profile.Name, profile.ScanType, string(portsJSON), profile.EnableSNMP, profile.EnableSSH, profile.TimeoutSec, profile.MaxWorkers, profile.Description, profile.UpdatedAt, profile.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (s *SQLiteProfileStorage) Get(id string) (*model.ScanProfile, error) {
	row := s.db.QueryRow(`SELECT id, name, scan_type, ports, enable_snmp, enable_ssh, timeout_sec, max_workers, description, created_at, updated_at FROM scan_profiles WHERE id=?`, id)
	return s.scanProfile(row)
}

func (s *SQLiteProfileStorage) List() ([]model.ScanProfile, error) {
	rows, err := s.db.Query(`SELECT id, name, scan_type, ports, enable_snmp, enable_ssh, timeout_sec, max_workers, description, created_at, updated_at FROM scan_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []model.ScanProfile
	for rows.Next() {
		var p model.ScanProfile
		var portsJSON sql.NullString
		var description sql.NullString
		err := rows.Scan(&p.ID, &p.Name, &p.ScanType, &portsJSON, &p.EnableSNMP, &p.EnableSSH, &p.TimeoutSec, &p.MaxWorkers, &description, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if portsJSON.Valid {
			json.Unmarshal([]byte(portsJSON.String), &p.Ports)
		}
		p.Description = description.String
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (s *SQLiteProfileStorage) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM scan_profiles WHERE id=?`, id)
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
	var description sql.NullString
	err := row.Scan(&p.ID, &p.Name, &p.ScanType, &portsJSON, &p.EnableSNMP, &p.EnableSSH, &p.TimeoutSec, &p.MaxWorkers, &description, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	if portsJSON.Valid {
		json.Unmarshal([]byte(portsJSON.String), &p.Ports)
	}
	p.Description = description.String
	return &p, nil
}
