package storage

import (
	"testing"
)

func TestEncodeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil slice", nil, "[]"},
		{"empty int slice", []int{}, "[]"},
		{"int slice", []int{22, 80, 443}, "[22,80,443]"},
		{"empty string slice", []string{}, "[]"},
		{"string slice", []string{"web", "db"}, `["web","db"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeJSON(tt.input)
			if result != tt.expected {
				t.Errorf("EncodeJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Run("decode int slice", func(t *testing.T) {
		var result []int
		DecodeJSON("[22,80,443]", &result)
		if len(result) != 3 || result[0] != 22 {
			t.Errorf("DecodeJSON() = %v, want [22,80,443]", result)
		}
	})

	t.Run("decode string slice", func(t *testing.T) {
		var result []string
		DecodeJSON(`["web","db"]`, &result)
		if len(result) != 2 || result[0] != "web" {
			t.Errorf("DecodeJSON() = %v, want [web,db]", result)
		}
	})

	t.Run("empty string leaves nil", func(t *testing.T) {
		var result []int
		DecodeJSON("", &result)
		if result != nil {
			t.Errorf("DecodeJSON() = %v, want nil", result)
		}
	})

	t.Run("null string leaves nil", func(t *testing.T) {
		var result []int
		DecodeJSON("null", &result)
		if result != nil {
			t.Errorf("DecodeJSON() = %v, want nil", result)
		}
	})
}

func TestDecodeJSONNullable(t *testing.T) {
	t.Run("valid data decodes", func(t *testing.T) {
		var result []int
		DecodeJSONNullable("[1,2,3]", true, &result)
		if len(result) != 3 {
			t.Errorf("DecodeJSONNullable() = %v, want [1,2,3]", result)
		}
	})

	t.Run("invalid leaves nil", func(t *testing.T) {
		var result []int
		DecodeJSONNullable("[1,2,3]", false, &result)
		if result != nil {
			t.Errorf("DecodeJSONNullable() = %v, want nil", result)
		}
	})

	t.Run("empty data leaves nil", func(t *testing.T) {
		var result []int
		DecodeJSONNullable("", true, &result)
		if result != nil {
			t.Errorf("DecodeJSONNullable() = %v, want nil", result)
		}
	})
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	original := []int{22, 80, 443, 8080}
	encoded := EncodeJSON(original)

	var decoded []int
	DecodeJSON(encoded, &decoded)

	if len(decoded) != len(original) {
		t.Fatalf("roundtrip length mismatch: got %d, want %d", len(decoded), len(original))
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("roundtrip mismatch at %d: got %d, want %d", i, decoded[i], original[i])
		}
	}
}
