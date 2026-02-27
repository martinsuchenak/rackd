package model

import "time"

// CustomFieldType defines the type of a custom field
type CustomFieldType string

const (
	CustomFieldTypeText   CustomFieldType = "text"
	CustomFieldTypeNumber CustomFieldType = "number"
	CustomFieldTypeBool   CustomFieldType = "boolean"
	CustomFieldTypeSelect CustomFieldType = "select"
)

// IsValid returns true if the custom field type is valid
func (t CustomFieldType) IsValid() bool {
	switch t {
	case CustomFieldTypeText, CustomFieldTypeNumber, CustomFieldTypeBool, CustomFieldTypeSelect:
		return true
	default:
		return false
	}
}

// CustomFieldDefinition defines the schema for a custom field
type CustomFieldDefinition struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`        // Display name
	Key         string          `json:"key"`         // Unique key for API/queries
	Type        CustomFieldType `json:"type"`        // text, number, boolean, select
	Required    bool            `json:"required"`    // Whether field is required
	Options     []string        `json:"options"`     // For select type - available options
	Description string          `json:"description"` // Help text
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CustomFieldValue stores a value for a device
type CustomFieldValue struct {
	ID          string `json:"id"`
	DeviceID    string `json:"device_id"`
	FieldID     string `json:"field_id"`
	StringValue string `json:"string_value"` // Used for text, select
	NumberValue *int64 `json:"number_value"` // Used for number
	BoolValue   *bool  `json:"bool_value"`   // Used for boolean
}

// GetValue returns the value as an interface based on the field type
func (v *CustomFieldValue) GetValue(fieldType CustomFieldType) interface{} {
	switch fieldType {
	case CustomFieldTypeNumber:
		if v.NumberValue != nil {
			return *v.NumberValue
		}
		return nil
	case CustomFieldTypeBool:
		if v.BoolValue != nil {
			return *v.BoolValue
		}
		return nil
	default:
		return v.StringValue
	}
}

// SetValue sets the value based on the field type
func (v *CustomFieldValue) SetValue(fieldType CustomFieldType, value interface{}) {
	switch fieldType {
	case CustomFieldTypeNumber:
		if n, ok := value.(int64); ok {
			v.NumberValue = &n
		} else if n, ok := value.(int); ok {
			n64 := int64(n)
			v.NumberValue = &n64
		} else if value == nil {
			v.NumberValue = nil
		}
	case CustomFieldTypeBool:
		if b, ok := value.(bool); ok {
			v.BoolValue = &b
		} else if value == nil {
			v.BoolValue = nil
		}
	default:
		if s, ok := value.(string); ok {
			v.StringValue = s
		} else if value == nil {
			v.StringValue = ""
		}
	}
}

// CustomFieldValueInput is used for API input
type CustomFieldValueInput struct {
	FieldID string      `json:"field_id"`
	Value   interface{} `json:"value"`
}

// CustomFieldFilter for filtering devices by custom fields
type CustomFieldFilter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CustomFieldDefinitionFilter for listing definitions
type CustomFieldDefinitionFilter struct {
	Type string `json:"type,omitempty"`
}

// CreateCustomFieldDefinitionRequest for creating a definition
type CreateCustomFieldDefinitionRequest struct {
	Name        string          `json:"name"`
	Key         string          `json:"key"`
	Type        CustomFieldType `json:"type"`
	Required    bool            `json:"required"`
	Options     []string        `json:"options"`
	Description string          `json:"description"`
}

// UpdateCustomFieldDefinitionRequest for updating a definition
type UpdateCustomFieldDefinitionRequest struct {
	Name        *string         `json:"name,omitempty"`
	Key         *string         `json:"key,omitempty"`
	Type        *CustomFieldType `json:"type,omitempty"`
	Required    *bool           `json:"required,omitempty"`
	Options     *[]string       `json:"options,omitempty"`
	Description *string         `json:"description,omitempty"`
}

// CustomFieldWithDefinition combines a value with its definition for display
type CustomFieldWithDefinition struct {
	Definition CustomFieldDefinition `json:"definition"`
	Value      interface{}           `json:"value"`
}
