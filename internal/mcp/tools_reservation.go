package mcp

import (
	"context"
	"time"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerReservationTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("reservation_list", "List IP reservations with optional filters",
			mcp.String("pool_id", "Filter by pool"),
			mcp.String("status", "Filter by status (active, expired, claimed, released)"),
			mcp.String("ip_address", "Filter by IP address"),
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		).Discoverable("reservation", "ip", "pool", "allocate", "assign"),
		s.handleReservationList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("reservation_get", "Get a reservation by ID",
			mcp.String("id", "Reservation ID", mcp.Required()),
		).Discoverable("reservation", "ip", "pool"),
		s.handleReservationGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("reservation_create", "Reserve an IP address from a pool",
			mcp.String("pool_id", "Pool ID", mcp.Required()),
			mcp.String("ip_address", "Specific IP to reserve (omit to auto-assign)"),
			mcp.String("hostname", "Hostname for the reservation"),
			mcp.String("purpose", "Purpose or description"),
			mcp.String("expires_at", "Expiry time (RFC3339, e.g. 2026-12-31T00:00:00Z)"),
			mcp.String("notes", "Additional notes"),
		).Discoverable("reservation", "ip", "pool", "allocate", "assign", "create"),
		s.handleReservationCreate,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("reservation_update", "Update a reservation",
			mcp.String("id", "Reservation ID", mcp.Required()),
			mcp.String("hostname", "Hostname"),
			mcp.String("purpose", "Purpose"),
			mcp.String("expires_at", "Expiry time (RFC3339)"),
			mcp.String("notes", "Notes"),
		).Discoverable("reservation", "ip", "update", "modify"),
		s.handleReservationUpdate,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("reservation_release", "Release a reservation back to the pool",
			mcp.String("id", "Reservation ID", mcp.Required()),
		).Discoverable("reservation", "ip", "release", "free", "pool"),
		s.handleReservationRelease,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("reservation_delete", "Delete a reservation record",
			mcp.String("id", "Reservation ID", mcp.Required()),
		).Discoverable("reservation", "ip", "delete", "remove"),
		s.handleReservationDelete,
	)
}

func (s *Server) handleReservationList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	filter := &model.ReservationFilter{
		Pagination: pg,
		PoolID:     req.StringOr("pool_id", ""),
		Status:     model.ReservationStatus(req.StringOr("status", "")),
		IPAddress:  req.StringOr("ip_address", ""),
	}
	reservations, err := s.svc.Reservations.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(reservations, len(reservations), pg)), nil
}

func (s *Server) handleReservationGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	reservation, err := s.svc.Reservations.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(reservation), nil
}

func (s *Server) handleReservationCreate(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	poolID, _ := req.String("pool_id")
	createReq := &model.CreateReservationRequest{
		PoolID:    poolID,
		IPAddress: req.StringOr("ip_address", ""),
		Hostname:  req.StringOr("hostname", ""),
		Purpose:   req.StringOr("purpose", ""),
		Notes:     req.StringOr("notes", ""),
	}

	if expiresStr := req.StringOr("expires_at", ""); expiresStr != "" {
		t, err := time.Parse(time.RFC3339, expiresStr)
		if err != nil {
			return nil, mcp.NewToolErrorInvalidParams("expires_at must be RFC3339 format, e.g. 2026-12-31T00:00:00Z")
		}
		createReq.ExpiresAt = &t
	}

	reservation, err := s.svc.Reservations.Create(ctx, createReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(reservation), nil
}

func (s *Server) handleReservationUpdate(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	updateReq := &model.UpdateReservationRequest{
		Hostname: req.StringOr("hostname", ""),
		Purpose:  req.StringOr("purpose", ""),
		Notes:    req.StringOr("notes", ""),
	}

	if expiresStr := req.StringOr("expires_at", ""); expiresStr != "" {
		t, err := time.Parse(time.RFC3339, expiresStr)
		if err != nil {
			return nil, mcp.NewToolErrorInvalidParams("expires_at must be RFC3339 format, e.g. 2026-12-31T00:00:00Z")
		}
		updateReq.ExpiresAt = &t
	}

	reservation, err := s.svc.Reservations.Update(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(reservation), nil
}

func (s *Server) handleReservationRelease(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Reservations.Release(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "released", "id": id}), nil
}

func (s *Server) handleReservationDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Reservations.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}
