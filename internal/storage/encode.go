package storage

import "encoding/json"

// EncodeJSON encodes a value to JSON string for storage.
// Returns "[]" for nil slices to ensure consistent empty array representation.
func EncodeJSON(v any) string {
	if v == nil {
		return "[]"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// DecodeJSON decodes a JSON string into the target value.
// Handles empty strings and null values gracefully.
func DecodeJSON(data string, v any) {
	if data == "" || data == "null" {
		return
	}
	json.Unmarshal([]byte(data), v)
}

// DecodeJSONNullable decodes from sql.NullString-like values.
// Only decodes if valid is true and data is non-empty.
func DecodeJSONNullable(data string, valid bool, v any) {
	if !valid || data == "" {
		return
	}
	json.Unmarshal([]byte(data), v)
}
