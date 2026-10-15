package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestNormalizeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"model field error", &model.FieldError{Field: "type", Message: "bad type"}, ErrValidation},
		{"missing id", storage.ErrInvalidID, ErrValidation},
		{"storage not found", fmt.Errorf("loading: %w", storage.ErrNetworkNotFound), ErrNotFound},
		{"duplicate", errors.New("constraint failed: UNIQUE constraint failed: devices.name (2067)"), ErrAlreadyExists},
		{"missing column", errors.New("constraint failed: NOT NULL constraint failed: devices.name (1299)"), ErrValidation},
		{"broken reference", errors.New("constraint failed: FOREIGN KEY constraint failed (787)"), ErrValidation},
		{"dns provider not found", ErrDNSProviderNotFound, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeError(tt.err); !errors.Is(got, tt.want) {
				t.Fatalf("NormalizeError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}

	t.Run("not found keeps the storage message", func(t *testing.T) {
		if got := NormalizeError(storage.ErrNetworkNotFound).Error(); got != "network not found" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("field error keeps its field", func(t *testing.T) {
		var verrs ValidationErrors
		if !errors.As(NormalizeError(&model.FieldError{Field: "max_workers", Message: "too many"}), &verrs) || verrs[0].Field != "max_workers" {
			t.Fatalf("got %v", verrs)
		}
	})
	t.Run("other errors pass through", func(t *testing.T) {
		internal := errors.New("disk I/O error")
		if got := NormalizeError(internal); got != internal {
			t.Fatalf("got %v", got)
		}
	})
}
