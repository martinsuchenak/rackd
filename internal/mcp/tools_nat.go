package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerNATTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("nat_list", "List NAT mappings with optional filters",
			mcp.String("external_ip", "Filter by external IP"),
			mcp.String("internal_ip", "Filter by internal IP"),
			mcp.String("protocol", "Filter by protocol (tcp, udp, any)"),
			mcp.String("device_id", "Filter by device"),
			mcp.String("datacenter_id", "Filter by datacenter"),
			mcp.String("network_id", "Filter by network"),
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		).Discoverable("nat", "port", "forward", "mapping", "translation", "firewall"),
		s.handleNATList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("nat_get", "Get a NAT mapping by ID",
			mcp.String("id", "NAT mapping ID", mcp.Required()),
		).Discoverable("nat", "port", "forward", "mapping"),
		s.handleNATGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("nat_save", "Create or update a NAT mapping",
			mcp.String("id", "NAT mapping ID (omit for new)"),
			mcp.String("name", "Mapping name", mcp.Required()),
			mcp.String("external_ip", "External IP address", mcp.Required()),
			mcp.Number("external_port", "External port"),
			mcp.String("internal_ip", "Internal IP address", mcp.Required()),
			mcp.Number("internal_port", "Internal port"),
			mcp.String("protocol", "Protocol (tcp, udp, any)"),
			mcp.String("device_id", "Associated device ID"),
			mcp.String("description", "Description"),
			mcp.Boolean("enabled", "Whether the mapping is enabled"),
			mcp.String("datacenter_id", "Datacenter ID"),
			mcp.String("network_id", "Network ID"),
			mcp.StringArray("tags", "Tags"),
		).Discoverable("nat", "port", "forward", "create", "update", "mapping"),
		s.handleNATSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("nat_delete", "Delete a NAT mapping",
			mcp.String("id", "NAT mapping ID", mcp.Required()),
		).Discoverable("nat", "delete", "remove", "mapping"),
		s.handleNATDelete,
	)
}

func (s *Server) handleNATList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	filter := &model.NATFilter{
		Pagination:   pg,
		ExternalIP:   req.StringOr("external_ip", ""),
		InternalIP:   req.StringOr("internal_ip", ""),
		Protocol:     model.NATProtocol(req.StringOr("protocol", "")),
		DeviceID:     req.StringOr("device_id", ""),
		DatacenterID: req.StringOr("datacenter_id", ""),
		NetworkID:    req.StringOr("network_id", ""),
	}
	mappings, err := s.svc.NAT.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(mappings, len(mappings), pg)), nil
}

func (s *Server) handleNATGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	mapping, err := s.svc.NAT.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(mapping), nil
}

func (s *Server) handleNATSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")
	externalIP, _ := req.String("external_ip")
	internalIP, _ := req.String("internal_ip")

	if id == "" {
		createReq := &model.CreateNATRequest{
			Name:         name,
			ExternalIP:   externalIP,
			ExternalPort: req.IntOr("external_port", 0),
			InternalIP:   internalIP,
			InternalPort: req.IntOr("internal_port", 0),
			Protocol:     model.NATProtocol(req.StringOr("protocol", string(model.NATProtocolAny))),
			DeviceID:     req.StringOr("device_id", ""),
			Description:  req.StringOr("description", ""),
			Enabled:      req.BoolOr("enabled", true),
			DatacenterID: req.StringOr("datacenter_id", ""),
			NetworkID:    req.StringOr("network_id", ""),
			Tags:         req.StringSliceOr("tags", []string{}),
		}
		mapping, err := s.svc.NAT.Create(ctx, createReq)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(mapping), nil
	}

	// Update — only set fields that were provided
	updateReq := &model.UpdateNATRequest{}
	if name != "" {
		updateReq.Name = &name
	}
	if externalIP != "" {
		updateReq.ExternalIP = &externalIP
	}
	if internalIP != "" {
		updateReq.InternalIP = &internalIP
	}
	if v := req.StringOr("description", ""); v != "" {
		updateReq.Description = &v
	}
	if v := req.StringOr("protocol", ""); v != "" {
		p := model.NATProtocol(v)
		updateReq.Protocol = &p
	}

	mapping, err := s.svc.NAT.Update(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(mapping), nil
}

func (s *Server) handleNATDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.NAT.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}
