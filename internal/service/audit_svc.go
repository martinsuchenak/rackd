package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type AuditService struct {
	store storage.ExtendedStorage
}

func NewAuditService(store storage.ExtendedStorage) *AuditService {
	return &AuditService{store: store}
}

func (s *AuditService) List(ctx context.Context, filter *model.AuditFilter) ([]model.AuditLog, error) {
	if err := requirePermission(ctx, s.store, "audit", "list"); err != nil {
		return nil, err
	}
	return s.store.ListAuditLogs(ctx, filter)
}

func (s *AuditService) Get(ctx context.Context, id string) (*model.AuditLog, error) {
	if err := requirePermission(ctx, s.store, "audit", "list"); err != nil {
		return nil, err
	}

	log, err := s.store.GetAuditLog(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrAuditLogNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return log, nil
}

func (s *AuditService) Export(ctx context.Context, filter *model.AuditFilter, format string) ([]byte, error) {
	if err := requirePermission(ctx, s.store, "audit", "list"); err != nil {
		return nil, err
	}

	logs, err := s.store.ListAuditLogs(ctx, filter)
	if err != nil {
		return nil, err
	}

	switch format {
	case "csv":
		return audit.ExportAuditLogsCSV(logs)
	case "json":
		return audit.ExportAuditLogsJSON(logs)
	default:
		return audit.ExportAuditLogsJSON(logs)
	}
}
