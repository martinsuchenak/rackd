package service

import (
	"context"
	"strings"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type CustomFieldService struct {
	store storage.ExtendedStorage
}

func NewCustomFieldService(store storage.ExtendedStorage) *CustomFieldService {
	return &CustomFieldService{store: store}
}

// ListDefinitions returns all custom field definitions
func (s *CustomFieldService) ListDefinitions(ctx context.Context, filter *model.CustomFieldDefinitionFilter) ([]model.CustomFieldDefinition, error) {
	if err := requirePermission(ctx, s.store, "custom-fields", "list"); err != nil {
		return nil, err
	}

	return s.store.ListCustomFieldDefinitions(ctx, filter)
}

// GetDefinition returns a single custom field definition by ID
func (s *CustomFieldService) GetDefinition(ctx context.Context, id string) (*model.CustomFieldDefinition, error) {
	if err := requirePermission(ctx, s.store, "custom-fields", "read"); err != nil {
		return nil, err
	}

	def, err := s.store.GetCustomFieldDefinition(ctx, id)
	if err != nil {
		if err == storage.ErrCustomFieldNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return def, nil
}

// CreateDefinition creates a new custom field definition
func (s *CustomFieldService) CreateDefinition(ctx context.Context, req *model.CreateCustomFieldDefinitionRequest) (*model.CustomFieldDefinition, error) {
	if err := requirePermission(ctx, s.store, "custom-fields", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.Key == "" {
		return nil, ValidationErrors{{Field: "key", Message: "Key is required"}}
	}

	// Validate key format (alphanumeric and underscores only)
	if !isValidFieldKey(req.Key) {
		return nil, ValidationErrors{{Field: "key", Message: "Key must contain only lowercase letters, numbers, and underscores"}}
	}

	// Validate type
	if !req.Type.IsValid() {
		return nil, ValidationErrors{{Field: "type", Message: "Invalid field type: " + string(req.Type)}}
	}

	// Validate select type has options
	if req.Type == model.CustomFieldTypeSelect && len(req.Options) == 0 {
		return nil, ValidationErrors{{Field: "options", Message: "Select type requires at least one option"}}
	}

	def := &model.CustomFieldDefinition{
		Name:        req.Name,
		Key:         strings.ToLower(req.Key),
		Type:        req.Type,
		Required:    req.Required,
		Options:     req.Options,
		Description: req.Description,
	}

	if err := s.store.CreateCustomFieldDefinition(ctx, def); err != nil {
		if err == storage.ErrDuplicateFieldKey {
			return nil, ValidationErrors{{Field: "key", Message: "A field with this key already exists"}}
		}
		return nil, err
	}

	return def, nil
}

// UpdateDefinition updates an existing custom field definition
func (s *CustomFieldService) UpdateDefinition(ctx context.Context, id string, req *model.UpdateCustomFieldDefinitionRequest) (*model.CustomFieldDefinition, error) {
	if err := requirePermission(ctx, s.store, "custom-fields", "update"); err != nil {
		return nil, err
	}

	def, err := s.store.GetCustomFieldDefinition(ctx, id)
	if err != nil {
		if err == storage.ErrCustomFieldNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ValidationErrors{{Field: "name", Message: "Name cannot be empty"}}
		}
		def.Name = *req.Name
	}
	if req.Key != nil {
		if *req.Key == "" {
			return nil, ValidationErrors{{Field: "key", Message: "Key cannot be empty"}}
		}
		if !isValidFieldKey(*req.Key) {
			return nil, ValidationErrors{{Field: "key", Message: "Key must contain only lowercase letters, numbers, and underscores"}}
		}
		def.Key = strings.ToLower(*req.Key)
	}
	if req.Type != nil {
		if !req.Type.IsValid() {
			return nil, ValidationErrors{{Field: "type", Message: "Invalid field type"}}
		}
		// Changing type from select to another type is allowed
		// Changing to select requires options to be set
		if *req.Type == model.CustomFieldTypeSelect && (req.Options == nil || len(*req.Options) == 0) && len(def.Options) == 0 {
			return nil, ValidationErrors{{Field: "options", Message: "Select type requires at least one option"}}
		}
		def.Type = *req.Type
	}
	if req.Required != nil {
		def.Required = *req.Required
	}
	if req.Options != nil {
		if def.Type == model.CustomFieldTypeSelect && len(*req.Options) == 0 {
			return nil, ValidationErrors{{Field: "options", Message: "Select type requires at least one option"}}
		}
		def.Options = *req.Options
	}
	if req.Description != nil {
		def.Description = *req.Description
	}

	if err := s.store.UpdateCustomFieldDefinition(ctx, def); err != nil {
		if err == storage.ErrDuplicateFieldKey {
			return nil, ValidationErrors{{Field: "key", Message: "A field with this key already exists"}}
		}
		return nil, err
	}

	return def, nil
}

// DeleteDefinition deletes a custom field definition
func (s *CustomFieldService) DeleteDefinition(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "custom-fields", "delete"); err != nil {
		return err
	}

	err := s.store.DeleteCustomFieldDefinition(ctx, id)
	if err != nil {
		if err == storage.ErrCustomFieldNotFound {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// GetValues retrieves all custom field values for a device
func (s *CustomFieldService) GetValues(ctx context.Context, deviceID string) ([]model.CustomFieldValue, error) {
	if err := requirePermission(ctx, s.store, "custom-fields", "read"); err != nil {
		return nil, err
	}

	return s.store.GetCustomFieldValues(ctx, deviceID)
}

// GetValuesWithDefinitions retrieves all custom field values for a device with their definitions
func (s *CustomFieldService) GetValuesWithDefinitions(ctx context.Context, deviceID string) ([]model.CustomFieldWithDefinition, error) {
	if err := requirePermission(ctx, s.store, "custom-fields", "read"); err != nil {
		return nil, err
	}

	return s.store.GetCustomFieldValuesWithDefinitions(ctx, deviceID)
}

// SetValue sets a custom field value for a device
func (s *CustomFieldService) SetValue(ctx context.Context, deviceID string, input *model.CustomFieldValueInput) error {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return err
	}

	// Get the definition to validate type
	def, err := s.store.GetCustomFieldDefinition(ctx, input.FieldID)
	if err != nil {
		if err == storage.ErrCustomFieldNotFound {
			return ValidationErrors{{Field: "field_id", Message: "Invalid field ID"}}
		}
		return err
	}

	// Validate value against definition type
	if err := validateCustomFieldValue(def, input.Value); err != nil {
		return err
	}

	// Create the value
	value := &model.CustomFieldValue{
		DeviceID: deviceID,
		FieldID:  input.FieldID,
	}
	value.SetValue(def.Type, input.Value)

	return s.store.SetCustomFieldValue(ctx, value)
}

// SetValues sets multiple custom field values for a device
func (s *CustomFieldService) SetValues(ctx context.Context, deviceID string, inputs []model.CustomFieldValueInput) error {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return err
	}

	// Get all definitions for validation
	definitions, err := s.store.ListCustomFieldDefinitions(ctx, nil)
	if err != nil {
		return err
	}

	// Create a map of definitions by ID
	defMap := make(map[string]*model.CustomFieldDefinition)
	for i := range definitions {
		defMap[definitions[i].ID] = &definitions[i]
	}

	// Validate all values first
	for _, input := range inputs {
		def, ok := defMap[input.FieldID]
		if !ok {
			return ValidationErrors{{Field: "field_id", Message: "Invalid field ID: " + input.FieldID}}
		}
		if err := validateCustomFieldValue(def, input.Value); err != nil {
			return err
		}
	}

	// Set all values
	for _, input := range inputs {
		def := defMap[input.FieldID]
		value := &model.CustomFieldValue{
			DeviceID: deviceID,
			FieldID:  input.FieldID,
		}
		value.SetValue(def.Type, input.Value)

		if err := s.store.SetCustomFieldValue(ctx, value); err != nil {
			return err
		}
	}

	return nil
}

// DeleteValue removes a custom field value from a device
func (s *CustomFieldService) DeleteValue(ctx context.Context, deviceID, fieldID string) error {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return err
	}

	err := s.store.DeleteCustomFieldValue(ctx, deviceID, fieldID)
	if err != nil {
		if err == storage.ErrCustomFieldNotFound {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// ValidateRequiredFields checks that all required custom fields have values
func (s *CustomFieldService) ValidateRequiredFields(ctx context.Context, values []model.CustomFieldValueInput) error {
	// Get all definitions
	definitions, err := s.store.ListCustomFieldDefinitions(ctx, nil)
	if err != nil {
		return err
	}

	// Create a map of provided values
	providedFields := make(map[string]bool)
	for _, v := range values {
		providedFields[v.FieldID] = true
	}

	// Check required fields
	var missingFields []string
	for _, def := range definitions {
		if def.Required {
			if !providedFields[def.ID] {
				missingFields = append(missingFields, def.Name)
			}
		}
	}

	if len(missingFields) > 0 {
		return ValidationErrors{{Field: "custom_fields", Message: "Missing required fields: " + strings.Join(missingFields, ", ")}}
	}

	return nil
}

// Helper functions

func isValidFieldKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for _, c := range key {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func validateCustomFieldValue(def *model.CustomFieldDefinition, value interface{}) error {
	if value == nil {
		return nil // nil is valid (clears the field)
	}

	switch def.Type {
	case model.CustomFieldTypeText:
		// Any string is valid
		if _, ok := value.(string); !ok {
			return ValidationErrors{{Field: "value", Message: "Expected string value for text field"}}
		}
	case model.CustomFieldTypeNumber:
		switch v := value.(type) {
		case int, int64, float64:
			// Valid number types
		case string:
			// Try to parse string as number (for JSON input)
			if v == "" {
				return nil // Empty string is treated as nil
			}
		default:
			return ValidationErrors{{Field: "value", Message: "Expected number value for number field"}}
		}
	case model.CustomFieldTypeBool:
		if _, ok := value.(bool); !ok {
			return ValidationErrors{{Field: "value", Message: "Expected boolean value for boolean field"}}
		}
	case model.CustomFieldTypeSelect:
		strVal, ok := value.(string)
		if !ok {
			return ValidationErrors{{Field: "value", Message: "Expected string value for select field"}}
		}
		valid := false
		for _, opt := range def.Options {
			if opt == strVal {
				valid = true
				break
			}
		}
		if !valid {
			return ValidationErrors{{Field: "value", Message: "Invalid option '" + strVal + "' for select field"}}
		}
	}

	return nil
}
