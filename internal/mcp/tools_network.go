package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerNetworkTools() {
	// Native tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("datacenter_list", "List all datacenters",
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		),
		s.handleDatacenterList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("datacenter_get", "Get a datacenter by ID",
			mcp.String("id", "Datacenter ID", mcp.Required()),
		),
		s.handleDatacenterGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("network_list", "List all networks",
			mcp.String("datacenter_id", "Filter by datacenter"),
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		),
		s.handleNetworkList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("network_get", "Get a network by ID",
			mcp.String("id", "Network ID", mcp.Required()),
		),
		s.handleNetworkGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("pool_get_next_ip", "Get the next available IP from a pool",
			mcp.String("pool_id", "Pool ID", mcp.Required()),
		),
		s.handleGetNextIP,
	)

	// Discoverable tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("datacenter_save", "Create or update a datacenter",
			mcp.String("id", "Datacenter ID (omit for new)"),
			mcp.String("name", "Datacenter name", mcp.Required()),
			mcp.String("location", "Physical location"),
			mcp.String("description", "Description"),
		).Discoverable("datacenter", "create", "update", "location", "facility"),
		s.handleDatacenterSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("datacenter_delete", "Delete a datacenter",
			mcp.String("id", "Datacenter ID", mcp.Required()),
		).Discoverable("datacenter", "delete", "remove"),
		s.handleDatacenterDelete,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("network_save", "Create or update a network",
			mcp.String("id", "Network ID (omit for new)"),
			mcp.String("name", "Network name", mcp.Required()),
			mcp.String("subnet", "CIDR subnet (e.g., 192.168.1.0/24)", mcp.Required()),
			mcp.String("datacenter_id", "Datacenter ID"),
			mcp.Number("vlan_id", "VLAN ID"),
			mcp.String("description", "Description"),
		).Discoverable("network", "subnet", "create", "update", "vlan"),
		s.handleNetworkSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("network_delete", "Delete a network",
			mcp.String("id", "Network ID", mcp.Required()),
		).Discoverable("network", "delete", "remove"),
		s.handleNetworkDelete,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("pool_list", "List IP pools for a network",
			mcp.String("network_id", "Network ID", mcp.Required()),
		).Discoverable("pool", "ip", "network", "list", "range"),
		s.handlePoolList,
	)
}

// Datacenter handlers

func (s *Server) handleDatacenterList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	filter := &model.DatacenterFilter{Pagination: pg}
	dcs, err := s.svc.Datacenters.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(dcs, len(dcs), pg)), nil
}

func (s *Server) handleDatacenterGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	dc, err := s.svc.Datacenters.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(dc), nil
}

func (s *Server) handleDatacenterSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")

	dc := &model.Datacenter{
		ID:          id,
		Name:        name,
		Location:    req.StringOr("location", ""),
		Description: req.StringOr("description", ""),
	}

	if id == "" {
		if err := s.svc.Datacenters.Create(ctx, dc); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	} else {
		if err := s.svc.Datacenters.Update(ctx, dc); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	}
	return mcp.NewToolResponseJSON(dc), nil
}

func (s *Server) handleDatacenterDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Datacenters.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

// Network handlers

func (s *Server) handleNetworkList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	filter := &model.NetworkFilter{
		Pagination:   pg,
		DatacenterID: req.StringOr("datacenter_id", ""),
	}
	networks, err := s.svc.Networks.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(networks, len(networks), pg)), nil
}

func (s *Server) handleNetworkGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	network, err := s.svc.Networks.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(network), nil
}

func (s *Server) handleNetworkSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")
	subnet, _ := req.String("subnet")

	network := &model.Network{
		ID:           id,
		Name:         name,
		Subnet:       subnet,
		DatacenterID: req.StringOr("datacenter_id", ""),
		VLANID:       req.IntOr("vlan_id", 0),
		Description:  req.StringOr("description", ""),
	}

	if id == "" {
		if err := s.svc.Networks.Create(ctx, network); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	} else {
		if err := s.svc.Networks.Update(ctx, network); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	}
	return mcp.NewToolResponseJSON(network), nil
}

func (s *Server) handleNetworkDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Networks.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

// Pool handlers

func (s *Server) handlePoolList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	networkID, _ := req.String("network_id")
	pools, err := s.svc.Pools.ListByNetwork(ctx, networkID)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(pools), nil
}

func (s *Server) handleGetNextIP(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	poolID, _ := req.String("pool_id")
	ip, err := s.svc.Pools.GetNextIP(ctx, poolID)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"ip": ip}), nil
}
