package credentials

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
)

var (
	ErrCredentialNotFound = errors.New("credential not found")
	ErrInvalidCredential  = errors.New("invalid credential")
)

type Storage interface {
	Create(cred *model.Credential) error
	Update(cred *model.Credential) error
	Get(id string) (*model.Credential, error)
	List(datacenterID string) ([]model.Credential, error)
	Delete(id string) error
}

type SQLiteStorage struct {
	db        *sql.DB
	encryptor *Encryptor
}

func NewSQLiteStorage(db *sql.DB, encryptionKey []byte) (*SQLiteStorage, error) {
	enc, err := NewEncryptor(encryptionKey)
	if err != nil {
		return nil, err
	}
	s := &SQLiteStorage{db: db, encryptor: enc}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStorage) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS credentials (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			snmp_community TEXT,
			snmp_v3_user TEXT,
			snmp_v3_auth TEXT,
			snmp_v3_priv TEXT,
			ssh_username TEXT,
			ssh_key_id TEXT,
			datacenter_id TEXT,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *SQLiteStorage) Create(cred *model.Credential) error {
	if err := cred.Validate(); err != nil {
		return err
	}
	if cred.ID == "" {
		cred.ID = uuid.Must(uuid.NewV7()).String()
	}
	now := time.Now()
	cred.CreatedAt = now
	cred.UpdatedAt = now

	community, err := s.encryptor.Encrypt(cred.SNMPCommunity)
	if err != nil {
		return fmt.Errorf("encrypt snmp_community: %w", err)
	}
	v3user, err := s.encryptor.Encrypt(cred.SNMPV3User)
	if err != nil {
		return fmt.Errorf("encrypt snmp_v3_user: %w", err)
	}
	v3auth, err := s.encryptor.Encrypt(cred.SNMPV3Auth)
	if err != nil {
		return fmt.Errorf("encrypt snmp_v3_auth: %w", err)
	}
	v3priv, err := s.encryptor.Encrypt(cred.SNMPV3Priv)
	if err != nil {
		return fmt.Errorf("encrypt snmp_v3_priv: %w", err)
	}
	sshKey, err := s.encryptor.Encrypt(cred.SSHKeyID)
	if err != nil {
		return fmt.Errorf("encrypt ssh_key_id: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO credentials (id, name, type, snmp_community, snmp_v3_user, snmp_v3_auth, snmp_v3_priv, ssh_username, ssh_key_id, datacenter_id, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cred.ID, cred.Name, cred.Type, community, v3user, v3auth, v3priv, cred.SSHUsername, sshKey, cred.DatacenterID, cred.Description, cred.CreatedAt, cred.UpdatedAt)
	return err
}

func (s *SQLiteStorage) Update(cred *model.Credential) error {
	if err := cred.Validate(); err != nil {
		return err
	}
	cred.UpdatedAt = time.Now()

	community, err := s.encryptor.Encrypt(cred.SNMPCommunity)
	if err != nil {
		return fmt.Errorf("encrypt snmp_community: %w", err)
	}
	v3user, err := s.encryptor.Encrypt(cred.SNMPV3User)
	if err != nil {
		return fmt.Errorf("encrypt snmp_v3_user: %w", err)
	}
	v3auth, err := s.encryptor.Encrypt(cred.SNMPV3Auth)
	if err != nil {
		return fmt.Errorf("encrypt snmp_v3_auth: %w", err)
	}
	v3priv, err := s.encryptor.Encrypt(cred.SNMPV3Priv)
	if err != nil {
		return fmt.Errorf("encrypt snmp_v3_priv: %w", err)
	}
	sshKey, err := s.encryptor.Encrypt(cred.SSHKeyID)
	if err != nil {
		return fmt.Errorf("encrypt ssh_key_id: %w", err)
	}

	res, err := s.db.Exec(`
		UPDATE credentials SET name=?, type=?, snmp_community=?, snmp_v3_user=?, snmp_v3_auth=?, snmp_v3_priv=?, ssh_username=?, ssh_key_id=?, datacenter_id=?, description=?, updated_at=?
		WHERE id=?
	`, cred.Name, cred.Type, community, v3user, v3auth, v3priv, cred.SSHUsername, sshKey, cred.DatacenterID, cred.Description, cred.UpdatedAt, cred.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

func (s *SQLiteStorage) Get(id string) (*model.Credential, error) {
	row := s.db.QueryRow(`SELECT id, name, type, snmp_community, snmp_v3_user, snmp_v3_auth, snmp_v3_priv, ssh_username, ssh_key_id, datacenter_id, description, created_at, updated_at FROM credentials WHERE id=?`, id)
	return s.scanCredential(row)
}

func (s *SQLiteStorage) List(datacenterID string) ([]model.Credential, error) {
	var rows *sql.Rows
	var err error
	if datacenterID == "" {
		rows, err = s.db.Query(`SELECT id, name, type, snmp_community, snmp_v3_user, snmp_v3_auth, snmp_v3_priv, ssh_username, ssh_key_id, datacenter_id, description, created_at, updated_at FROM credentials ORDER BY name`)
	} else {
		rows, err = s.db.Query(`SELECT id, name, type, snmp_community, snmp_v3_user, snmp_v3_auth, snmp_v3_priv, ssh_username, ssh_key_id, datacenter_id, description, created_at, updated_at FROM credentials WHERE datacenter_id=? OR datacenter_id='' ORDER BY name`, datacenterID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []model.Credential
	for rows.Next() {
		cred, err := s.scanCredentialRows(rows)
		if err != nil {
			return nil, err
		}
		creds = append(creds, *cred)
	}
	return creds, rows.Err()
}

func (s *SQLiteStorage) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM credentials WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

func (s *SQLiteStorage) scanCredential(row *sql.Row) (*model.Credential, error) {
	var cred model.Credential
	var community, v3user, v3auth, v3priv, sshKey sql.NullString
	var datacenterID, description sql.NullString
	err := row.Scan(&cred.ID, &cred.Name, &cred.Type, &community, &v3user, &v3auth, &v3priv, &cred.SSHUsername, &sshKey, &datacenterID, &description, &cred.CreatedAt, &cred.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	var decErr error
	if community.String != "" {
		if cred.SNMPCommunity, decErr = s.encryptor.Decrypt(community.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_community: %w", decErr)
		}
	}
	if v3user.String != "" {
		if cred.SNMPV3User, decErr = s.encryptor.Decrypt(v3user.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_v3_user: %w", decErr)
		}
	}
	if v3auth.String != "" {
		if cred.SNMPV3Auth, decErr = s.encryptor.Decrypt(v3auth.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_v3_auth: %w", decErr)
		}
	}
	if v3priv.String != "" {
		if cred.SNMPV3Priv, decErr = s.encryptor.Decrypt(v3priv.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_v3_priv: %w", decErr)
		}
	}
	if sshKey.String != "" {
		if cred.SSHKeyID, decErr = s.encryptor.Decrypt(sshKey.String); decErr != nil {
			return nil, fmt.Errorf("decrypt ssh_key_id: %w", decErr)
		}
	}

	cred.DatacenterID = datacenterID.String
	cred.Description = description.String
	return &cred, nil
}

func (s *SQLiteStorage) scanCredentialRows(rows *sql.Rows) (*model.Credential, error) {
	var cred model.Credential
	var community, v3user, v3auth, v3priv, sshKey sql.NullString
	var datacenterID, description sql.NullString
	err := rows.Scan(&cred.ID, &cred.Name, &cred.Type, &community, &v3user, &v3auth, &v3priv, &cred.SSHUsername, &sshKey, &datacenterID, &description, &cred.CreatedAt, &cred.UpdatedAt)
	if err != nil {
		return nil, err
	}
	var decErr error
	if community.String != "" {
		if cred.SNMPCommunity, decErr = s.encryptor.Decrypt(community.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_community: %w", decErr)
		}
	}
	if v3user.String != "" {
		if cred.SNMPV3User, decErr = s.encryptor.Decrypt(v3user.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_v3_user: %w", decErr)
		}
	}
	if v3auth.String != "" {
		if cred.SNMPV3Auth, decErr = s.encryptor.Decrypt(v3auth.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_v3_auth: %w", decErr)
		}
	}
	if v3priv.String != "" {
		if cred.SNMPV3Priv, decErr = s.encryptor.Decrypt(v3priv.String); decErr != nil {
			return nil, fmt.Errorf("decrypt snmp_v3_priv: %w", decErr)
		}
	}
	if sshKey.String != "" {
		if cred.SSHKeyID, decErr = s.encryptor.Decrypt(sshKey.String); decErr != nil {
			return nil, fmt.Errorf("decrypt ssh_key_id: %w", decErr)
		}
	}

	cred.DatacenterID = datacenterID.String
	cred.Description = description.String
	return &cred, nil
}
