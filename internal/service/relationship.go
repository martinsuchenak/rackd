package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type RelationshipService struct {
	store storage.ExtendedStorage
}

func NewRelationshipService(store storage.ExtendedStorage) *RelationshipService {
	return &RelationshipService{store: store}
}

func (s *RelationshipService) ListAll(ctx context.Context) ([]model.DeviceRelationship, error) {
	if err := requirePermission(ctx, s.store, "relationships", "list"); err != nil {
		return nil, err
	}
	return s.store.ListAllRelationships(ctx)
}

func (s *RelationshipService) Add(ctx context.Context, parentID, childID, relationshipType, notes string) error {
	if err := requirePermission(ctx, s.store, "relationships", "create"); err != nil {
		return err
	}

	if parentID == "" {
		return ValidationErrors{{Field: "parent_id", Message: "Parent ID is required"}}
	}

	if childID == "" {
		return ValidationErrors{{Field: "child_id", Message: "Child ID is required"}}
	}

	if relationshipType == "" {
		return ValidationErrors{{Field: "type", Message: "Relationship type is required"}}
	}

	if relationshipType != model.RelationshipContains &&
		relationshipType != model.RelationshipConnectedTo &&
		relationshipType != model.RelationshipDependsOn {
		return ValidationErrors{{Field: "type", Message: "Relationship type must be one of: contains, connected_to, depends_on"}}
	}

	return s.store.AddRelationship(enrichAuditCtx(ctx), parentID, childID, relationshipType, notes)
}

func (s *RelationshipService) Get(ctx context.Context, deviceID string) ([]model.DeviceRelationship, error) {
	if err := requirePermission(ctx, s.store, "relationships", "read"); err != nil {
		return nil, err
	}

	if deviceID == "" {
		return nil, ValidationErrors{{Field: "device_id", Message: "Device ID is required"}}
	}

	return s.store.GetRelationships(ctx, deviceID)
}

func (s *RelationshipService) GetRelated(ctx context.Context, deviceID, relationshipType string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "relationships", "read"); err != nil {
		return nil, err
	}

	if deviceID == "" {
		return nil, ValidationErrors{{Field: "device_id", Message: "Device ID is required"}}
	}

	return s.store.GetRelatedDevices(ctx, deviceID, relationshipType)
}

func (s *RelationshipService) UpdateNotes(ctx context.Context, parentID, childID, relationshipType, notes string) error {
	if err := requirePermission(ctx, s.store, "relationships", "update"); err != nil {
		return err
	}

	if parentID == "" {
		return ValidationErrors{{Field: "parent_id", Message: "Parent ID is required"}}
	}

	if childID == "" {
		return ValidationErrors{{Field: "child_id", Message: "Child ID is required"}}
	}

	if relationshipType == "" {
		return ValidationErrors{{Field: "type", Message: "Relationship type is required"}}
	}

	return s.store.UpdateRelationshipNotes(enrichAuditCtx(ctx), parentID, childID, relationshipType, notes)
}

func (s *RelationshipService) Remove(ctx context.Context, parentID, childID, relationshipType string) error {
	if err := requirePermission(ctx, s.store, "relationships", "delete"); err != nil {
		return err
	}

	if parentID == "" {
		return ValidationErrors{{Field: "parent_id", Message: "Parent ID is required"}}
	}

	if childID == "" {
		return ValidationErrors{{Field: "child_id", Message: "Child ID is required"}}
	}

	if relationshipType == "" {
		return ValidationErrors{{Field: "type", Message: "Relationship type is required"}}
	}

	if err := s.store.RemoveRelationship(enrichAuditCtx(ctx), parentID, childID, relationshipType); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
