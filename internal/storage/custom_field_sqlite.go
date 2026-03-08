package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateCustomFieldDefinition creates a new custom field definition
func (s *SQLiteStorage) CreateCustomFieldDefinition(ctx context.Context, def *model.CustomFieldDefinition) error {
	if def.ID == "" {
		def.ID = newUUID()
	}
	def.CreatedAt = time.Now().UTC()
	def.UpdatedAt = def.CreatedAt

	optionsJSON, err := json.Marshal(def.Options)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO custom_field_definitions (id, name, key, type, required, options, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, def.ID, def.Name, def.Key, def.Type, def.Required, string(optionsJSON), def.Description, def.CreatedAt, def.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation on key
		if isUniqueConstraintError(err) {
			return ErrDuplicateFieldKey
		}
		return err
	}

	return nil
}

// GetCustomFieldDefinition retrieves a custom field definition by ID
func (s *SQLiteStorage) GetCustomFieldDefinition(ctx context.Context, id string) (*model.CustomFieldDefinition, error) {
	def := &model.CustomFieldDefinition{}
	var optionsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, key, type, required, options, description, created_at, updated_at
		FROM custom_field_definitions WHERE id = ?
	`, id).Scan(&def.ID, &def.Name, &def.Key, &def.Type, &def.Required, &optionsJSON, &def.Description, &def.CreatedAt, &def.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrCustomFieldNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(optionsJSON), &def.Options); err != nil {
		return nil, err
	}

	return def, nil
}

// GetCustomFieldDefinitionByKey retrieves a custom field definition by its unique key
func (s *SQLiteStorage) GetCustomFieldDefinitionByKey(ctx context.Context, key string) (*model.CustomFieldDefinition, error) {
	def := &model.CustomFieldDefinition{}
	var optionsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, key, type, required, options, description, created_at, updated_at
		FROM custom_field_definitions WHERE key = ?
	`, key).Scan(&def.ID, &def.Name, &def.Key, &def.Type, &def.Required, &optionsJSON, &def.Description, &def.CreatedAt, &def.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrCustomFieldNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(optionsJSON), &def.Options); err != nil {
		return nil, err
	}

	return def, nil
}

// ListCustomFieldDefinitions lists all custom field definitions
func (s *SQLiteStorage) ListCustomFieldDefinitions(ctx context.Context, filter *model.CustomFieldDefinitionFilter) ([]model.CustomFieldDefinition, error) {
	query := `SELECT id, name, key, type, required, options, description, created_at, updated_at
		FROM custom_field_definitions WHERE 1=1`
	var args []any

	if filter != nil && filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	query += " ORDER BY name ASC"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCustomFieldDefinitions(rows)
}

// UpdateCustomFieldDefinition updates an existing custom field definition
func (s *SQLiteStorage) UpdateCustomFieldDefinition(ctx context.Context, def *model.CustomFieldDefinition) error {
	def.UpdatedAt = time.Now().UTC()

	optionsJSON, err := json.Marshal(def.Options)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE custom_field_definitions SET name = ?, key = ?, type = ?, required = ?, options = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, def.Name, def.Key, def.Type, def.Required, string(optionsJSON), def.Description, def.UpdatedAt, def.ID)

	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicateFieldKey
		}
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCustomFieldNotFound
	}

	return nil
}

// DeleteCustomFieldDefinition deletes a custom field definition and all its values
func (s *SQLiteStorage) DeleteCustomFieldDefinition(ctx context.Context, id string) error {
	// First delete all values for this definition
	_, err := s.db.ExecContext(ctx, `DELETE FROM custom_field_values WHERE field_id = ?`, id)
	if err != nil {
		return err
	}

	// Then delete the definition
	result, err := s.db.ExecContext(ctx, `DELETE FROM custom_field_definitions WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCustomFieldNotFound
	}

	return nil
}

// SetCustomFieldValue sets a custom field value for a device (upsert)
func (s *SQLiteStorage) SetCustomFieldValue(ctx context.Context, value *model.CustomFieldValue) error {
	if value.ID == "" {
		value.ID = newUUID()
	}

	// Use INSERT OR REPLACE for upsert behavior
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO custom_field_values (id, device_id, field_id, string_value, number_value, bool_value)
		VALUES (
			COALESCE((SELECT id FROM custom_field_values WHERE device_id = ? AND field_id = ?), ?),
			?, ?, ?, ?, ?
		)
	`, value.DeviceID, value.FieldID, value.ID,
		value.DeviceID, value.FieldID, value.StringValue, value.NumberValue, value.BoolValue)

	return err
}

// GetCustomFieldValues retrieves all custom field values for a device
func (s *SQLiteStorage) GetCustomFieldValues(ctx context.Context, deviceID string) ([]model.CustomFieldValue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_id, field_id, string_value, number_value, bool_value
		FROM custom_field_values WHERE device_id = ?
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCustomFieldValues(rows)
}

// GetCustomFieldValue retrieves a specific custom field value for a device
func (s *SQLiteStorage) GetCustomFieldValue(ctx context.Context, deviceID, fieldID string) (*model.CustomFieldValue, error) {
	value := &model.CustomFieldValue{}
	var numberValue sql.NullInt64
	var boolValue sql.NullBool

	err := s.db.QueryRowContext(ctx, `
		SELECT id, device_id, field_id, string_value, number_value, bool_value
		FROM custom_field_values WHERE device_id = ? AND field_id = ?
	`, deviceID, fieldID).Scan(&value.ID, &value.DeviceID, &value.FieldID, &value.StringValue, &numberValue, &boolValue)

	if err == sql.ErrNoRows {
		return nil, ErrCustomFieldNotFound
	}
	if err != nil {
		return nil, err
	}

	if numberValue.Valid {
		value.NumberValue = &numberValue.Int64
	}
	if boolValue.Valid {
		value.BoolValue = &boolValue.Bool
	}

	return value, nil
}

// DeleteCustomFieldValue deletes a specific custom field value
func (s *SQLiteStorage) DeleteCustomFieldValue(ctx context.Context, deviceID, fieldID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM custom_field_values WHERE device_id = ? AND field_id = ?`, deviceID, fieldID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCustomFieldNotFound
	}

	return nil
}

// DeleteCustomFieldValuesForDevice deletes all custom field values for a device
func (s *SQLiteStorage) DeleteCustomFieldValuesForDevice(ctx context.Context, deviceID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM custom_field_values WHERE device_id = ?`, deviceID)
	return err
}

// DeleteCustomFieldValuesForDefinition deletes all custom field values for a definition
func (s *SQLiteStorage) DeleteCustomFieldValuesForDefinition(ctx context.Context, fieldID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM custom_field_values WHERE field_id = ?`, fieldID)
	return err
}

// Helper functions

func scanCustomFieldDefinitions(rows *sql.Rows) ([]model.CustomFieldDefinition, error) {
	var definitions []model.CustomFieldDefinition
	for rows.Next() {
		var def model.CustomFieldDefinition
		var optionsJSON string
		if err := rows.Scan(&def.ID, &def.Name, &def.Key, &def.Type, &def.Required, &optionsJSON, &def.Description, &def.CreatedAt, &def.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(optionsJSON), &def.Options); err != nil {
			return nil, err
		}
		definitions = append(definitions, def)
	}
	return definitions, nil
}

func scanCustomFieldValues(rows *sql.Rows) ([]model.CustomFieldValue, error) {
	var values []model.CustomFieldValue
	for rows.Next() {
		var value model.CustomFieldValue
		var numberValue sql.NullInt64
		var boolValue sql.NullBool
		if err := rows.Scan(&value.ID, &value.DeviceID, &value.FieldID, &value.StringValue, &numberValue, &boolValue); err != nil {
			return nil, err
		}
		if numberValue.Valid {
			value.NumberValue = &numberValue.Int64
		}
		if boolValue.Valid {
			value.BoolValue = &boolValue.Bool
		}
		values = append(values, value)
	}
	return values, nil
}

// isUniqueConstraintError checks if the error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite unique constraint error
	return err.Error() != "" && (containsString(err.Error(), "UNIQUE constraint failed") ||
		containsString(err.Error(), "duplicate key"))
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetDevicesByCustomField finds devices that have a specific custom field value
func (s *SQLiteStorage) GetDevicesByCustomField(ctx context.Context, fieldKey, value string) ([]string, error) {
	// First get the field ID from the key
	var fieldID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM custom_field_definitions WHERE key = ?`, fieldKey).Scan(&fieldID)
	if err == sql.ErrNoRows {
		return nil, ErrCustomFieldNotFound
	}
	if err != nil {
		return nil, err
	}

	// Find devices with matching value
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_id FROM custom_field_values WHERE field_id = ? AND string_value = ?
	`, fieldID, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deviceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		deviceIDs = append(deviceIDs, id)
	}

	return deviceIDs, nil
}

// GetCustomFieldValuesWithDefinitions retrieves all custom field values for a device with their definitions
func (s *SQLiteStorage) GetCustomFieldValuesWithDefinitions(ctx context.Context, deviceID string) ([]model.CustomFieldWithDefinition, error) {
	query := `
		SELECT
			d.id, d.name, d.key, d.type, d.required, d.options, d.description, d.created_at, d.updated_at,
			COALESCE(v.id, ''), COALESCE(v.string_value, ''), v.number_value, v.bool_value
		FROM custom_field_definitions d
		LEFT JOIN custom_field_values v ON d.id = v.field_id AND v.device_id = ?
		ORDER BY d.name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.CustomFieldWithDefinition
	for rows.Next() {
		var def model.CustomFieldDefinition
		var optionsJSON string
		var value model.CustomFieldValue
		var numberValue sql.NullInt64
		var boolValue sql.NullBool

		if err := rows.Scan(
			&def.ID, &def.Name, &def.Key, &def.Type, &def.Required, &optionsJSON, &def.Description, &def.CreatedAt, &def.UpdatedAt,
			&value.ID, &value.StringValue, &numberValue, &boolValue,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(optionsJSON), &def.Options); err != nil {
			return nil, err
		}

		if numberValue.Valid {
			value.NumberValue = &numberValue.Int64
		}
		if boolValue.Valid {
			value.BoolValue = &boolValue.Bool
		}

		// Only include if there's a value set
		if value.ID != "" {
			value.DeviceID = deviceID
			value.FieldID = def.ID
			results = append(results, model.CustomFieldWithDefinition{
				Definition: def,
				Value:      value.GetValue(def.Type),
			})
		}
	}

	return results, nil
}

// ValidateCustomFieldValue validates a value against its field definition
func (s *SQLiteStorage) ValidateCustomFieldValue(ctx context.Context, fieldID string, value interface{}) error {
	def, err := s.GetCustomFieldDefinition(ctx, fieldID)
	if err != nil {
		return err
	}

	// Validate based on type
	switch def.Type {
	case model.CustomFieldTypeText:
		// Any string is valid
	case model.CustomFieldTypeNumber:
		// Must be a number
		if value != nil {
			switch value.(type) {
			case int, int64, float64:
				// Valid
			default:
				return fmt.Errorf("invalid value type for number field: expected number")
			}
		}
	case model.CustomFieldTypeBool:
		// Must be a boolean
		if value != nil {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("invalid value type for boolean field: expected boolean")
			}
		}
	case model.CustomFieldTypeSelect:
		// Must be one of the options
		if value != nil {
			strVal, ok := value.(string)
			if !ok {
				return fmt.Errorf("invalid value type for select field: expected string")
			}
			valid := false
			for _, opt := range def.Options {
				if opt == strVal {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid value for select field: %q is not a valid option", strVal)
			}
		}
	}

	return nil
}
