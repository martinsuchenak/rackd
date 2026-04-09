package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Custom Field Definition Tests
// ============================================================================

func TestCustomFieldDefinition_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{
		Name:        "Asset Tag",
		Key:         "asset_tag",
		Type:        model.CustomFieldTypeText,
		Required:    false,
		Description: "Asset tag identifier",
	}

	// Create definition
	err := storage.CreateCustomFieldDefinition(context.Background(), def)
	if err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	if def.ID == "" {
		t.Error("definition ID should be set after creation")
	}
	if def.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if def.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}

	// Get definition
	retrieved, err := storage.GetCustomFieldDefinition(context.Background(), def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldDefinition failed: %v", err)
	}

	if retrieved.Name != def.Name {
		t.Errorf("expected name %s, got %s", def.Name, retrieved.Name)
	}
	if retrieved.Key != def.Key {
		t.Errorf("expected key %s, got %s", def.Key, retrieved.Key)
	}
	if retrieved.Type != def.Type {
		t.Errorf("expected type %s, got %s", def.Type, retrieved.Type)
	}
	if retrieved.Required != def.Required {
		t.Errorf("expected required %v, got %v", def.Required, retrieved.Required)
	}
	if retrieved.Description != def.Description {
		t.Errorf("expected description %s, got %s", def.Description, retrieved.Description)
	}
}

func TestCustomFieldDefinition_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetCustomFieldDefinition(context.Background(), "non-existent-id")
	if err != ErrCustomFieldNotFound {
		t.Errorf("expected ErrCustomFieldNotFound, got %v", err)
	}
}

func TestCustomFieldDefinition_GetByKey(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{
		Name: "Cost Center",
		Key:  "cost_center",
		Type: model.CustomFieldTypeText,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	// Get by key
	retrieved, err := storage.GetCustomFieldDefinitionByKey(context.Background(), "cost_center")
	if err != nil {
		t.Fatalf("GetCustomFieldDefinitionByKey failed: %v", err)
	}

	if retrieved.ID != def.ID {
		t.Errorf("expected ID %s, got %s", def.ID, retrieved.ID)
	}

	// Non-existent key
	_, err = storage.GetCustomFieldDefinitionByKey(context.Background(), "non_existent")
	if err != ErrCustomFieldNotFound {
		t.Errorf("expected ErrCustomFieldNotFound, got %v", err)
	}
}

func TestCustomFieldDefinition_DuplicateKey(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def1 := &model.CustomFieldDefinition{
		Name: "First",
		Key:  "duplicate_key",
		Type: model.CustomFieldTypeText,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def1); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	def2 := &model.CustomFieldDefinition{
		Name: "Second",
		Key:  "duplicate_key", // Same key
		Type: model.CustomFieldTypeNumber,
	}
	err := storage.CreateCustomFieldDefinition(context.Background(), def2)
	if err != ErrDuplicateFieldKey {
		t.Errorf("expected ErrDuplicateFieldKey, got %v", err)
	}
}

func TestCustomFieldDefinition_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create definition
	def := &model.CustomFieldDefinition{
		Name:     "Original Name",
		Key:      "original_key",
		Type:     model.CustomFieldTypeText,
		Required: false,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	originalUpdatedAt := def.UpdatedAt

	// Update definition
	def.Name = "Updated Name"
	def.Required = true
	def.Description = "New description"

	err := storage.UpdateCustomFieldDefinition(context.Background(), def)
	if err != nil {
		t.Fatalf("UpdateCustomFieldDefinition failed: %v", err)
	}

	if !def.UpdatedAt.After(originalUpdatedAt) {
		t.Error("updated_at should be updated")
	}

	// Verify update
	retrieved, err := storage.GetCustomFieldDefinition(context.Background(), def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldDefinition failed: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", retrieved.Name)
	}
	if !retrieved.Required {
		t.Error("expected required to be true")
	}
	if retrieved.Description != "New description" {
		t.Errorf("expected description 'New description', got %s", retrieved.Description)
	}
}

func TestCustomFieldDefinition_UpdateNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{
		ID:   "non-existent-id",
		Name: "Test",
		Key:  "test",
		Type: model.CustomFieldTypeText,
	}

	err := storage.UpdateCustomFieldDefinition(context.Background(), def)
	if err != ErrCustomFieldNotFound {
		t.Errorf("expected ErrCustomFieldNotFound, got %v", err)
	}
}

func TestCustomFieldDefinition_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create definition
	def := &model.CustomFieldDefinition{
		Name: "To Delete",
		Key:  "to_delete",
		Type: model.CustomFieldTypeText,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	// Delete definition
	err := storage.DeleteCustomFieldDefinition(context.Background(), def.ID)
	if err != nil {
		t.Fatalf("DeleteCustomFieldDefinition failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetCustomFieldDefinition(context.Background(), def.ID)
	if err != ErrCustomFieldNotFound {
		t.Errorf("expected ErrCustomFieldNotFound after deletion, got %v", err)
	}
}

func TestCustomFieldDefinition_DeleteNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteCustomFieldDefinition(context.Background(), "non-existent-id")
	if err != ErrCustomFieldNotFound {
		t.Errorf("expected ErrCustomFieldNotFound, got %v", err)
	}
}

func TestCustomFieldDefinition_List(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple definitions
	definitions := []*model.CustomFieldDefinition{
		{Name: "Alpha", Key: "alpha", Type: model.CustomFieldTypeText},
		{Name: "Beta", Key: "beta", Type: model.CustomFieldTypeNumber},
		{Name: "Gamma", Key: "gamma", Type: model.CustomFieldTypeSelect, Options: []string{"opt1", "opt2"}},
	}

	for _, def := range definitions {
		if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
			t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
		}
	}

	// List all
	list, err := storage.ListCustomFieldDefinitions(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListCustomFieldDefinitions failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 definitions, got %d", len(list))
	}

	// Filter by type
	filter := &model.CustomFieldDefinitionFilter{Type: "number"}
	list, err = storage.ListCustomFieldDefinitions(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListCustomFieldDefinitions with filter failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("expected 1 definition with type number, got %d", len(list))
	}
	if len(list) > 0 && list[0].Key != "beta" {
		t.Errorf("expected key 'beta', got %s", list[0].Key)
	}
}

func TestCustomFieldValuesWithDefinitionsAndValidation(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()
	ctx := context.Background()

	device := &model.Device{Name: "CustomFieldDevice"}
	if err := storage.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	selectDef := &model.CustomFieldDefinition{
		Name:    "Environment",
		Key:     "environment",
		Type:    model.CustomFieldTypeSelect,
		Options: []string{"prod", "dev"},
	}
	textDef := &model.CustomFieldDefinition{
		Name: "Asset Tag",
		Key:  "asset_tag_2",
		Type: model.CustomFieldTypeText,
	}
	for _, def := range []*model.CustomFieldDefinition{selectDef, textDef} {
		if err := storage.CreateCustomFieldDefinition(ctx, def); err != nil {
			t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
		}
	}

	if err := storage.SetCustomFieldValue(ctx, &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     selectDef.ID,
		StringValue: "prod",
	}); err != nil {
		t.Fatalf("SetCustomFieldValue select failed: %v", err)
	}
	if err := storage.SetCustomFieldValue(ctx, &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     textDef.ID,
		StringValue: "asset-123",
	}); err != nil {
		t.Fatalf("SetCustomFieldValue text failed: %v", err)
	}

	values, err := storage.GetCustomFieldValuesWithDefinitions(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValuesWithDefinitions failed: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 custom field values with definitions, got %d", len(values))
	}

	if err := storage.ValidateCustomFieldValue(ctx, selectDef.ID, "prod"); err != nil {
		t.Fatalf("ValidateCustomFieldValue valid select failed: %v", err)
	}
	if err := storage.ValidateCustomFieldValue(ctx, selectDef.ID, "invalid"); err == nil {
		t.Fatal("expected invalid select value to fail validation")
	}

	if err := storage.DeleteCustomFieldValuesForDefinition(ctx, selectDef.ID); err != nil {
		t.Fatalf("DeleteCustomFieldValuesForDefinition failed: %v", err)
	}
	values, err = storage.GetCustomFieldValuesWithDefinitions(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValuesWithDefinitions after delete failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 remaining custom field value, got %d", len(values))
	}
}

func TestCustomFieldDefinition_SelectTypeWithOptions(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{
		Name:    "Environment",
		Key:     "environment",
		Type:    model.CustomFieldTypeSelect,
		Options: []string{"production", "staging", "development"},
	}

	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	retrieved, err := storage.GetCustomFieldDefinition(context.Background(), def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldDefinition failed: %v", err)
	}

	if len(retrieved.Options) != 3 {
		t.Errorf("expected 3 options, got %d", len(retrieved.Options))
	}

	// Verify options
	expectedOpts := map[string]bool{"production": true, "staging": true, "development": true}
	for _, opt := range retrieved.Options {
		if !expectedOpts[opt] {
			t.Errorf("unexpected option: %s", opt)
		}
	}
}

// ============================================================================
// Custom Field Value Tests
// ============================================================================

func TestCustomFieldValue_SetAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create definition and device
	def := &model.CustomFieldDefinition{
		Name: "Asset Tag",
		Key:  "asset_tag",
		Type: model.CustomFieldTypeText,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	device := &model.Device{Name: "Test Device", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Set value
	value := &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     def.ID,
		StringValue: "ASSET-12345",
	}

	err := storage.SetCustomFieldValue(context.Background(), value)
	if err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	if value.ID == "" {
		t.Error("value ID should be set after creation")
	}

	// Get value
	retrieved, err := storage.GetCustomFieldValue(context.Background(), device.ID, def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValue failed: %v", err)
	}

	if retrieved.StringValue != "ASSET-12345" {
		t.Errorf("expected string value 'ASSET-12345', got %s", retrieved.StringValue)
	}
}

func TestCustomFieldValue_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Setup
	def := &model.CustomFieldDefinition{
		Name: "Cost Center",
		Key:  "cost_center",
		Type: model.CustomFieldTypeText,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	device := &model.Device{Name: "Test Device", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Set initial value
	value := &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     def.ID,
		StringValue: "CC-001",
	}
	if err := storage.SetCustomFieldValue(context.Background(), value); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	// Update value
	value.StringValue = "CC-002"
	if err := storage.SetCustomFieldValue(context.Background(), value); err != nil {
		t.Fatalf("SetCustomFieldValue update failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetCustomFieldValue(context.Background(), device.ID, def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValue failed: %v", err)
	}

	if retrieved.StringValue != "CC-002" {
		t.Errorf("expected string value 'CC-002', got %s", retrieved.StringValue)
	}
}

func TestCustomFieldValue_NumberType(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{
		Name: "Port Count",
		Key:  "port_count",
		Type: model.CustomFieldTypeNumber,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	device := &model.Device{Name: "Switch", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	numberValue := int64(48)
	value := &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     def.ID,
		NumberValue: &numberValue,
	}

	if err := storage.SetCustomFieldValue(context.Background(), value); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	retrieved, err := storage.GetCustomFieldValue(context.Background(), device.ID, def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValue failed: %v", err)
	}

	if retrieved.NumberValue == nil || *retrieved.NumberValue != 48 {
		t.Errorf("expected number value 48, got %v", retrieved.NumberValue)
	}

	// Test GetValue
	retrievedVal := retrieved.GetValue(model.CustomFieldTypeNumber)
	if n, ok := retrievedVal.(int64); !ok || n != 48 {
		t.Errorf("expected GetValue to return 48, got %v", retrievedVal)
	}
}

func TestCustomFieldValue_BoolType(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{
		Name: "Monitored",
		Key:  "monitored",
		Type: model.CustomFieldTypeBool,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	device := &model.Device{Name: "Server", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	boolValue := true
	value := &model.CustomFieldValue{
		DeviceID:  device.ID,
		FieldID:   def.ID,
		BoolValue: &boolValue,
	}

	if err := storage.SetCustomFieldValue(context.Background(), value); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	retrieved, err := storage.GetCustomFieldValue(context.Background(), device.ID, def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValue failed: %v", err)
	}

	if retrieved.BoolValue == nil || !*retrieved.BoolValue {
		t.Errorf("expected bool value true, got %v", retrieved.BoolValue)
	}
}

func TestCustomFieldValue_SelectType(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{
		Name:    "Environment",
		Key:     "environment",
		Type:    model.CustomFieldTypeSelect,
		Options: []string{"production", "staging", "development"},
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	device := &model.Device{Name: "App Server", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	value := &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     def.ID,
		StringValue: "production",
	}

	if err := storage.SetCustomFieldValue(context.Background(), value); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	retrieved, err := storage.GetCustomFieldValue(context.Background(), device.ID, def.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValue failed: %v", err)
	}

	if retrieved.StringValue != "production" {
		t.Errorf("expected string value 'production', got %s", retrieved.StringValue)
	}
}

func TestCustomFieldValue_GetValuesForDevice(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create definitions
	def1 := &model.CustomFieldDefinition{Name: "Field1", Key: "field1", Type: model.CustomFieldTypeText}
	def2 := &model.CustomFieldDefinition{Name: "Field2", Key: "field2", Type: model.CustomFieldTypeNumber}
	def3 := &model.CustomFieldDefinition{Name: "Field3", Key: "field3", Type: model.CustomFieldTypeBool}
	for _, def := range []*model.CustomFieldDefinition{def1, def2, def3} {
		if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
			t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
		}
	}

	// Create device
	device := &model.Device{Name: "Multi-field Device", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Set values
	numberVal := int64(42)
	boolVal := true
	values := []*model.CustomFieldValue{
		{DeviceID: device.ID, FieldID: def1.ID, StringValue: "text value"},
		{DeviceID: device.ID, FieldID: def2.ID, NumberValue: &numberVal},
		{DeviceID: device.ID, FieldID: def3.ID, BoolValue: &boolVal},
	}
	for _, v := range values {
		if err := storage.SetCustomFieldValue(context.Background(), v); err != nil {
			t.Fatalf("SetCustomFieldValue failed: %v", err)
		}
	}

	// Get all values for device
	retrieved, err := storage.GetCustomFieldValues(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues failed: %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("expected 3 values, got %d", len(retrieved))
	}
}

func TestCustomFieldValue_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{Name: "Temp", Key: "temp", Type: model.CustomFieldTypeText}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	device := &model.Device{Name: "Device", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	value := &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     def.ID,
		StringValue: "to delete",
	}
	if err := storage.SetCustomFieldValue(context.Background(), value); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	// Delete value
	err := storage.DeleteCustomFieldValue(context.Background(), device.ID, def.ID)
	if err != nil {
		t.Fatalf("DeleteCustomFieldValue failed: %v", err)
	}

	// Verify deletion
	_, err = storage.GetCustomFieldValue(context.Background(), device.ID, def.ID)
	if err != ErrCustomFieldNotFound {
		t.Errorf("expected ErrCustomFieldNotFound, got %v", err)
	}
}

func TestCustomFieldValue_DeleteForDevice(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def1 := &model.CustomFieldDefinition{Name: "F1", Key: "f1", Type: model.CustomFieldTypeText}
	def2 := &model.CustomFieldDefinition{Name: "F2", Key: "f2", Type: model.CustomFieldTypeText}
	for _, def := range []*model.CustomFieldDefinition{def1, def2} {
		if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
			t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
		}
	}

	device := &model.Device{Name: "Device", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Set values
	for _, def := range []*model.CustomFieldDefinition{def1, def2} {
		if err := storage.SetCustomFieldValue(context.Background(), &model.CustomFieldValue{
			DeviceID:    device.ID,
			FieldID:     def.ID,
			StringValue: "value",
		}); err != nil {
			t.Fatalf("SetCustomFieldValue failed: %v", err)
		}
	}

	// Delete all for device
	if err := storage.DeleteCustomFieldValuesForDevice(context.Background(), device.ID); err != nil {
		t.Fatalf("DeleteCustomFieldValuesForDevice failed: %v", err)
	}

	// Verify
	values, err := storage.GetCustomFieldValues(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues failed: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected 0 values after deletion, got %d", len(values))
	}
}

func TestCustomFieldValue_DeleteOnDefinitionDelete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	def := &model.CustomFieldDefinition{Name: "ToDelete", Key: "to_delete", Type: model.CustomFieldTypeText}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	device := &model.Device{Name: "Device", Status: model.DeviceStatusActive}
	if err := storage.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	// Set value
	if err := storage.SetCustomFieldValue(context.Background(), &model.CustomFieldValue{
		DeviceID:    device.ID,
		FieldID:     def.ID,
		StringValue: "value",
	}); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	// Delete definition (should cascade)
	if err := storage.DeleteCustomFieldDefinition(context.Background(), def.ID); err != nil {
		t.Fatalf("DeleteCustomFieldDefinition failed: %v", err)
	}

	// Verify value is also deleted
	values, err := storage.GetCustomFieldValues(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetCustomFieldValues failed: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected 0 values after definition deletion, got %d", len(values))
	}
}

func TestCustomFieldValue_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetCustomFieldValue(context.Background(), "non-existent-device", "non-existent-field")
	if err != ErrCustomFieldNotFound {
		t.Errorf("expected ErrCustomFieldNotFound, got %v", err)
	}
}

func TestCustomFieldValue_GetDevicesByCustomField(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create definition
	def := &model.CustomFieldDefinition{
		Name: "Department",
		Key:  "department",
		Type: model.CustomFieldTypeText,
	}
	if err := storage.CreateCustomFieldDefinition(context.Background(), def); err != nil {
		t.Fatalf("CreateCustomFieldDefinition failed: %v", err)
	}

	// Create devices
	device1 := &model.Device{Name: "Device1", Status: model.DeviceStatusActive}
	device2 := &model.Device{Name: "Device2", Status: model.DeviceStatusActive}
	device3 := &model.Device{Name: "Device3", Status: model.DeviceStatusActive}
	for _, d := range []*model.Device{device1, device2, device3} {
		if err := storage.CreateDevice(context.Background(), d); err != nil {
			t.Fatalf("CreateDevice failed: %v", err)
		}
	}

	// Set values
	if err := storage.SetCustomFieldValue(context.Background(), &model.CustomFieldValue{
		DeviceID:    device1.ID,
		FieldID:     def.ID,
		StringValue: "engineering",
	}); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}
	if err := storage.SetCustomFieldValue(context.Background(), &model.CustomFieldValue{
		DeviceID:    device2.ID,
		FieldID:     def.ID,
		StringValue: "engineering",
	}); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}
	if err := storage.SetCustomFieldValue(context.Background(), &model.CustomFieldValue{
		DeviceID:    device3.ID,
		FieldID:     def.ID,
		StringValue: "marketing",
	}); err != nil {
		t.Fatalf("SetCustomFieldValue failed: %v", err)
	}

	// Find devices by custom field
	deviceIDs, err := storage.GetDevicesByCustomField(context.Background(), "department", "engineering")
	if err != nil {
		t.Fatalf("GetDevicesByCustomField failed: %v", err)
	}

	if len(deviceIDs) != 2 {
		t.Errorf("expected 2 devices with department=engineering, got %d", len(deviceIDs))
	}

	// Verify the IDs
	foundIDs := make(map[string]bool)
	for _, id := range deviceIDs {
		foundIDs[id] = true
	}
	if !foundIDs[device1.ID] || !foundIDs[device2.ID] {
		t.Error("expected device1 and device2 to be in results")
	}
	if foundIDs[device3.ID] {
		t.Error("device3 should not be in results")
	}
}
