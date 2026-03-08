package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateCircuit creates a new circuit
func (s *SQLiteStorage) CreateCircuit(ctx context.Context, circuit *model.Circuit) error {
	if circuit.ID == "" {
		return ErrInvalidID
	}

	circuit.CreatedAt = time.Now().UTC()
	circuit.UpdatedAt = circuit.CreatedAt

	tagsJSON, _ := json.Marshal(circuit.Tags)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO circuits (
			id, name, circuit_id, provider, type, status, capacity_mbps,
			datacenter_a_id, datacenter_b_id, device_a_id, device_b_id,
			port_a, port_b, ip_address_a, ip_address_b, vlan_id,
			description, install_date, terminate_date, monthly_cost,
			contract_number, contact_name, contact_phone, contact_email,
			tags, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		circuit.ID, circuit.Name, circuit.CircuitID, circuit.Provider, circuit.Type, circuit.Status, circuit.CapacityMbps,
		nullString(circuit.DatacenterAID), nullString(circuit.DatacenterBID), nullString(circuit.DeviceAID), nullString(circuit.DeviceBID),
		circuit.PortA, circuit.PortB, circuit.IPAddressA, circuit.IPAddressB, circuit.VLANID,
		circuit.Description, nullTime(circuit.InstallDate), nullTime(circuit.TerminateDate), circuit.MonthlyCost,
		circuit.ContractNumber, circuit.ContactName, circuit.ContactPhone, circuit.ContactEmail,
		string(tagsJSON), circuit.CreatedAt, circuit.UpdatedAt,
	)

	return err
}

// GetCircuit retrieves a circuit by ID
func (s *SQLiteStorage) GetCircuit(ctx context.Context, id string) (*model.Circuit, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	circuit := &model.Circuit{}
	var tagsJSON string
	var datacenterAID, datacenterBID, deviceAID, deviceBID sql.NullString
	var installDate, terminateDate sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, circuit_id, provider, type, status, capacity_mbps,
			datacenter_a_id, datacenter_b_id, device_a_id, device_b_id,
			port_a, port_b, ip_address_a, ip_address_b, vlan_id,
			description, install_date, terminate_date, monthly_cost,
			contract_number, contact_name, contact_phone, contact_email,
			tags, created_at, updated_at
		FROM circuits WHERE id = ?
	`, id).Scan(
		&circuit.ID, &circuit.Name, &circuit.CircuitID, &circuit.Provider, &circuit.Type, &circuit.Status, &circuit.CapacityMbps,
		&datacenterAID, &datacenterBID, &deviceAID, &deviceBID,
		&circuit.PortA, &circuit.PortB, &circuit.IPAddressA, &circuit.IPAddressB, &circuit.VLANID,
		&circuit.Description, &installDate, &terminateDate, &circuit.MonthlyCost,
		&circuit.ContractNumber, &circuit.ContactName, &circuit.ContactPhone, &circuit.ContactEmail,
		&tagsJSON, &circuit.CreatedAt, &circuit.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCircuitNotFound
	}
	if err != nil {
		return nil, err
	}

	if datacenterAID.Valid {
		circuit.DatacenterAID = datacenterAID.String
	}
	if datacenterBID.Valid {
		circuit.DatacenterBID = datacenterBID.String
	}
	if deviceAID.Valid {
		circuit.DeviceAID = deviceAID.String
	}
	if deviceBID.Valid {
		circuit.DeviceBID = deviceBID.String
	}
	if installDate.Valid {
		circuit.InstallDate = &installDate.Time
	}
	if terminateDate.Valid {
		circuit.TerminateDate = &terminateDate.Time
	}

	if err := json.Unmarshal([]byte(tagsJSON), &circuit.Tags); err != nil {
		circuit.Tags = []string{}
	}

	return circuit, nil
}

// GetCircuitByCircuitID retrieves a circuit by provider's circuit ID
func (s *SQLiteStorage) GetCircuitByCircuitID(ctx context.Context, circuitID string) (*model.Circuit, error) {
	if circuitID == "" {
		return nil, ErrInvalidID
	}

	circuit := &model.Circuit{}
	var tagsJSON string
	var datacenterAID, datacenterBID, deviceAID, deviceBID sql.NullString
	var installDate, terminateDate sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, circuit_id, provider, type, status, capacity_mbps,
			datacenter_a_id, datacenter_b_id, device_a_id, device_b_id,
			port_a, port_b, ip_address_a, ip_address_b, vlan_id,
			description, install_date, terminate_date, monthly_cost,
			contract_number, contact_name, contact_phone, contact_email,
			tags, created_at, updated_at
		FROM circuits WHERE circuit_id = ?
	`, circuitID).Scan(
		&circuit.ID, &circuit.Name, &circuit.CircuitID, &circuit.Provider, &circuit.Type, &circuit.Status, &circuit.CapacityMbps,
		&datacenterAID, &datacenterBID, &deviceAID, &deviceBID,
		&circuit.PortA, &circuit.PortB, &circuit.IPAddressA, &circuit.IPAddressB, &circuit.VLANID,
		&circuit.Description, &installDate, &terminateDate, &circuit.MonthlyCost,
		&circuit.ContractNumber, &circuit.ContactName, &circuit.ContactPhone, &circuit.ContactEmail,
		&tagsJSON, &circuit.CreatedAt, &circuit.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCircuitNotFound
	}
	if err != nil {
		return nil, err
	}

	if datacenterAID.Valid {
		circuit.DatacenterAID = datacenterAID.String
	}
	if datacenterBID.Valid {
		circuit.DatacenterBID = datacenterBID.String
	}
	if deviceAID.Valid {
		circuit.DeviceAID = deviceAID.String
	}
	if deviceBID.Valid {
		circuit.DeviceBID = deviceBID.String
	}
	if installDate.Valid {
		circuit.InstallDate = &installDate.Time
	}
	if terminateDate.Valid {
		circuit.TerminateDate = &terminateDate.Time
	}

	if err := json.Unmarshal([]byte(tagsJSON), &circuit.Tags); err != nil {
		circuit.Tags = []string{}
	}

	return circuit, nil
}

// ListCircuits lists circuits with optional filtering
func (s *SQLiteStorage) ListCircuits(ctx context.Context, filter *model.CircuitFilter) ([]model.Circuit, error) {
	query := `SELECT id, name, circuit_id, provider, type, status, capacity_mbps,
		datacenter_a_id, datacenter_b_id, device_a_id, device_b_id,
		port_a, port_b, ip_address_a, ip_address_b, vlan_id,
		description, install_date, terminate_date, monthly_cost,
		contract_number, contact_name, contact_phone, contact_email,
		tags, created_at, updated_at
		FROM circuits`

	var args []any
	var conditions []string

	if filter != nil {
		if filter.Provider != "" {
			conditions = append(conditions, "provider = ?")
			args = append(args, filter.Provider)
		}
		if filter.Status != "" {
			conditions = append(conditions, "status = ?")
			args = append(args, filter.Status)
		}
		if filter.DatacenterID != "" {
			conditions = append(conditions, "(datacenter_a_id = ? OR datacenter_b_id = ?)")
			args = append(args, filter.DatacenterID, filter.DatacenterID)
		}
		if filter.Type != "" {
			conditions = append(conditions, "type = ?")
			args = append(args, filter.Type)
		}
		if len(filter.Tags) > 0 {
			for _, tag := range filter.Tags {
				conditions = append(conditions, "tags LIKE ?")
				args = append(args, "%\""+tag+"\"%")
			}
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}

	query += " ORDER BY name"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list circuits: %w", err)
	}
	defer rows.Close()

	var circuits []model.Circuit
	for rows.Next() {
		var circuit model.Circuit
		var tagsJSON string
		var datacenterAID, datacenterBID, deviceAID, deviceBID sql.NullString
		var installDate, terminateDate sql.NullTime

		if err := rows.Scan(
			&circuit.ID, &circuit.Name, &circuit.CircuitID, &circuit.Provider, &circuit.Type, &circuit.Status, &circuit.CapacityMbps,
			&datacenterAID, &datacenterBID, &deviceAID, &deviceBID,
			&circuit.PortA, &circuit.PortB, &circuit.IPAddressA, &circuit.IPAddressB, &circuit.VLANID,
			&circuit.Description, &installDate, &terminateDate, &circuit.MonthlyCost,
			&circuit.ContractNumber, &circuit.ContactName, &circuit.ContactPhone, &circuit.ContactEmail,
			&tagsJSON, &circuit.CreatedAt, &circuit.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan circuit: %w", err)
		}

		if datacenterAID.Valid {
			circuit.DatacenterAID = datacenterAID.String
		}
		if datacenterBID.Valid {
			circuit.DatacenterBID = datacenterBID.String
		}
		if deviceAID.Valid {
			circuit.DeviceAID = deviceAID.String
		}
		if deviceBID.Valid {
			circuit.DeviceBID = deviceBID.String
		}
		if installDate.Valid {
			circuit.InstallDate = &installDate.Time
		}
		if terminateDate.Valid {
			circuit.TerminateDate = &terminateDate.Time
		}

		if err := json.Unmarshal([]byte(tagsJSON), &circuit.Tags); err != nil {
			circuit.Tags = []string{}
		}

		circuits = append(circuits, circuit)
	}

	if circuits == nil {
		circuits = []model.Circuit{}
	}

	return circuits, nil
}

// UpdateCircuit updates an existing circuit
func (s *SQLiteStorage) UpdateCircuit(ctx context.Context, circuit *model.Circuit) error {
	if circuit.ID == "" {
		return ErrInvalidID
	}

	circuit.UpdatedAt = time.Now().UTC()

	tagsJSON, _ := json.Marshal(circuit.Tags)

	result, err := s.db.ExecContext(ctx, `
		UPDATE circuits SET
			name = ?, circuit_id = ?, provider = ?, type = ?, status = ?, capacity_mbps = ?,
			datacenter_a_id = ?, datacenter_b_id = ?, device_a_id = ?, device_b_id = ?,
			port_a = ?, port_b = ?, ip_address_a = ?, ip_address_b = ?, vlan_id = ?,
			description = ?, install_date = ?, terminate_date = ?, monthly_cost = ?,
			contract_number = ?, contact_name = ?, contact_phone = ?, contact_email = ?,
			tags = ?, updated_at = ?
		WHERE id = ?
	`,
		circuit.Name, circuit.CircuitID, circuit.Provider, circuit.Type, circuit.Status, circuit.CapacityMbps,
		nullString(circuit.DatacenterAID), nullString(circuit.DatacenterBID), nullString(circuit.DeviceAID), nullString(circuit.DeviceBID),
		circuit.PortA, circuit.PortB, circuit.IPAddressA, circuit.IPAddressB, circuit.VLANID,
		circuit.Description, nullTime(circuit.InstallDate), nullTime(circuit.TerminateDate), circuit.MonthlyCost,
		circuit.ContractNumber, circuit.ContactName, circuit.ContactPhone, circuit.ContactEmail,
		string(tagsJSON), circuit.UpdatedAt, circuit.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrCircuitNotFound
	}

	return nil
}

// DeleteCircuit deletes a circuit
func (s *SQLiteStorage) DeleteCircuit(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM circuits WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrCircuitNotFound
	}

	return nil
}

// GetCircuitsByDatacenter retrieves all circuits for a datacenter
func (s *SQLiteStorage) GetCircuitsByDatacenter(ctx context.Context, datacenterID string) ([]model.Circuit, error) {
	return s.ListCircuits(ctx, &model.CircuitFilter{DatacenterID: datacenterID})
}

// GetCircuitsByDevice retrieves all circuits linked to a device
func (s *SQLiteStorage) GetCircuitsByDevice(ctx context.Context, deviceID string) ([]model.Circuit, error) {
	query := `SELECT id, name, circuit_id, provider, type, status, capacity_mbps,
		datacenter_a_id, datacenter_b_id, device_a_id, device_b_id,
		port_a, port_b, ip_address_a, ip_address_b, vlan_id,
		description, install_date, terminate_date, monthly_cost,
		contract_number, contact_name, contact_phone, contact_email,
		tags, created_at, updated_at
		FROM circuits WHERE device_a_id = ? OR device_b_id = ?`

	rows, err := s.db.QueryContext(ctx, query, deviceID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get circuits by device: %w", err)
	}
	defer rows.Close()

	var circuits []model.Circuit
	for rows.Next() {
		var circuit model.Circuit
		var tagsJSON string
		var datacenterAID, datacenterBID, deviceAID, deviceBID sql.NullString
		var installDate, terminateDate sql.NullTime

		if err := rows.Scan(
			&circuit.ID, &circuit.Name, &circuit.CircuitID, &circuit.Provider, &circuit.Type, &circuit.Status, &circuit.CapacityMbps,
			&datacenterAID, &datacenterBID, &deviceAID, &deviceBID,
			&circuit.PortA, &circuit.PortB, &circuit.IPAddressA, &circuit.IPAddressB, &circuit.VLANID,
			&circuit.Description, &installDate, &terminateDate, &circuit.MonthlyCost,
			&circuit.ContractNumber, &circuit.ContactName, &circuit.ContactPhone, &circuit.ContactEmail,
			&tagsJSON, &circuit.CreatedAt, &circuit.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan circuit: %w", err)
		}

		if datacenterAID.Valid {
			circuit.DatacenterAID = datacenterAID.String
		}
		if datacenterBID.Valid {
			circuit.DatacenterBID = datacenterBID.String
		}
		if deviceAID.Valid {
			circuit.DeviceAID = deviceAID.String
		}
		if deviceBID.Valid {
			circuit.DeviceBID = deviceBID.String
		}
		if installDate.Valid {
			circuit.InstallDate = &installDate.Time
		}
		if terminateDate.Valid {
			circuit.TerminateDate = &terminateDate.Time
		}

		if err := json.Unmarshal([]byte(tagsJSON), &circuit.Tags); err != nil {
			circuit.Tags = []string{}
		}

		circuits = append(circuits, circuit)
	}

	if circuits == nil {
		circuits = []model.Circuit{}
	}

	return circuits, nil
}

// Helper function for joining conditions
func joinConditions(conditions []string, sep string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += sep
		}
		result += c
	}
	return result
}
