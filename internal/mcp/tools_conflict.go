package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerConflictTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("conflict_list", "List detected IP/subnet conflicts",
			mcp.String("type", "Filter by conflict type"),
			mcp.String("status", "Filter by status"),
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		).Discoverable("conflict", "duplicate", "ip", "subnet", "overlap", "collision"),
		s.handleConflictList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("conflict_detect", "Run conflict detection (duplicate IPs and overlapping subnets)",
		).Discoverable("conflict", "detect", "scan", "duplicate", "ip", "subnet", "overlap"),
		s.handleConflictDetect,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("conflict_resolve", "Resolve a conflict",
			mcp.String("conflict_id", "Conflict ID", mcp.Required()),
			mcp.String("keep_device_id", "For duplicate IP: device ID to keep the IP"),
			mcp.String("keep_network_id", "For overlapping subnet: network ID that is correct"),
		).Discoverable("conflict", "resolve", "fix", "duplicate", "ip"),
		s.handleConflictResolve,
	)
}

func (s *Server) handleConflictList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	filter := &model.ConflictFilter{
		Pagination: pg,
		Type:       model.ConflictType(req.StringOr("type", "")),
		Status:     model.ConflictStatus(req.StringOr("status", "")),
	}
	conflicts, err := s.svc.Conflicts.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(conflicts, len(conflicts), pg)), nil
}

func (s *Server) handleConflictDetect(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	dupIPs, err := s.svc.Conflicts.DetectDuplicateIPs(ctx)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	overlapping, err := s.svc.Conflicts.DetectOverlappingSubnets(ctx)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]interface{}{
		"duplicate_ips":       dupIPs,
		"overlapping_subnets": overlapping,
	}), nil
}

func (s *Server) handleConflictResolve(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	conflictID, _ := req.String("conflict_id")
	resolution := &model.ConflictResolution{
		ConflictID:   conflictID,
		KeepDeviceID: req.StringOr("keep_device_id", ""),
	}
	if err := s.svc.Conflicts.Resolve(ctx, resolution); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "resolved", "conflict_id": conflictID}), nil
}
