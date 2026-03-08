package mcp

import (
	"context"
	"time"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerAuditTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("audit_list", "List audit log entries",
			mcp.String("resource", "Filter by resource type (device, network, datacenter, etc.)"),
			mcp.String("resource_id", "Filter by specific resource ID"),
			mcp.String("user_id", "Filter by user ID"),
			mcp.String("action", "Filter by action (create, update, delete, etc.)"),
			mcp.String("start_time", "Start time filter (RFC3339)"),
			mcp.String("end_time", "End time filter (RFC3339)"),
			mcp.Number("limit", "Maximum number of entries to return (default 50)"),
			mcp.Number("offset", "Offset for pagination"),
		).Discoverable("audit", "log", "history", "activity", "change", "trail"),
		s.handleAuditList,
	)
}

func (s *Server) handleAuditList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	filter := &model.AuditFilter{
		Resource:   req.StringOr("resource", ""),
		ResourceID: req.StringOr("resource_id", ""),
		UserID:     req.StringOr("user_id", ""),
		Action:     req.StringOr("action", ""),
		Limit:      req.IntOr("limit", 50),
		Offset:     req.IntOr("offset", 0),
	}

	if v := req.StringOr("start_time", ""); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, mcp.NewToolErrorInvalidParams("start_time must be RFC3339 format")
		}
		filter.StartTime = &t
	}
	if v := req.StringOr("end_time", ""); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, mcp.NewToolErrorInvalidParams("end_time must be RFC3339 format")
		}
		filter.EndTime = &t
	}

	entries, err := s.svc.Audit.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(entries), nil
}
