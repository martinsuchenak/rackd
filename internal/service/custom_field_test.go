package service

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func TestCustomFieldService_CreateDefinitionValidatesKeyAndSelectOptions(t *testing.T) {
	store := newServiceTestStorage()
	svc := NewCustomFieldService(store)
	ctx := SystemContext(context.Background(), "test")

	_, err := svc.CreateDefinition(ctx, &model.CreateCustomFieldDefinitionRequest{
		Name: "Rack Unit",
		Key:  "Rack Unit",
		Type: model.CustomFieldTypeText,
	})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid key, got %v", err)
	}

	_, err = svc.CreateDefinition(ctx, &model.CreateCustomFieldDefinitionRequest{
		Name: "Environment",
		Key:  "environment",
		Type: model.CustomFieldTypeSelect,
	})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing select options, got %v", err)
	}
}

func TestCustomFieldService_SetValuesRejectsInvalidSelectOption(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "devices", "update", true)
	store.customDefs["field-1"] = &model.CustomFieldDefinition{
		ID:      "field-1",
		Name:    "Environment",
		Key:     "environment",
		Type:    model.CustomFieldTypeSelect,
		Options: []string{"prod", "dev"},
	}
	svc := NewCustomFieldService(store)

	err := svc.SetValues(userContext("user-1"), "device-1", []model.CustomFieldValueInput{{
		FieldID: "field-1",
		Value:   "qa",
	}})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid select value, got %v", err)
	}
}

func TestCustomFieldService_ValidateRequiredFieldsReportsMissingNames(t *testing.T) {
	store := newServiceTestStorage()
	store.customDefs["field-1"] = &model.CustomFieldDefinition{ID: "field-1", Name: "Environment", Required: true}
	store.customDefs["field-2"] = &model.CustomFieldDefinition{ID: "field-2", Name: "Rack Unit", Required: true}
	svc := NewCustomFieldService(store)

	err := svc.ValidateRequiredFields(context.Background(), []model.CustomFieldValueInput{{
		FieldID: "field-1",
		Value:   "prod",
	}})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing required field, got %v", err)
	}
	var validationErrs ValidationErrors
	if !errors.As(err, &validationErrs) || len(validationErrs) != 1 {
		t.Fatalf("expected one validation error, got %#v", err)
	}
	if validationErrs[0].Field != "custom_fields" || validationErrs[0].Message == "" {
		t.Fatalf("expected required-field message, got %#v", validationErrs[0])
	}
}

func TestCustomFieldService_DeleteValueMapsMissingFieldToNotFound(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "devices", "update", true)
	store.deleteCustomFieldErr = storage.ErrCustomFieldNotFound
	svc := NewCustomFieldService(store)

	err := svc.DeleteValue(userContext("user-1"), "device-1", "field-1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestCustomFieldHelpersValidateKeysAndTypedValues(t *testing.T) {
	if !isValidFieldKey("rack_unit_1") {
		t.Fatal("expected valid field key")
	}
	if isValidFieldKey("Rack Unit") {
		t.Fatal("expected invalid field key with spaces and uppercase letters")
	}

	selectDef := &model.CustomFieldDefinition{Type: model.CustomFieldTypeSelect, Options: []string{"prod", "dev"}}
	if err := validateCustomFieldValue(selectDef, "prod"); err != nil {
		t.Fatalf("expected valid select option, got %v", err)
	}
	if err := validateCustomFieldValue(selectDef, "qa"); err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid select option, got %v", err)
	}
}
