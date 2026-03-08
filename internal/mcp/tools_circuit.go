package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerCircuitTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("circuit_list", "List circuits with optional filters",
			mcp.String("provider", "Filter by provider"),
			mcp.String("status", "Filter by status (active, maintenance, down, decommissioned)"),
			mcp.String("datacenter_id", "Filter by datacenter"),
			mcp.String("type", "Filter by circuit type"),
		).Discoverable("circuit", "wan", "link", "fiber", "provider", "isp", "cross-connect"),
		s.handleCircuitList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("circuit_get", "Get a circuit by ID",
			mcp.String("id", "Circuit ID", mcp.Required()),
		).Discoverable("circuit", "wan", "link"),
		s.handleCircuitGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("circuit_save", "Create or update a circuit",
			mcp.String("id", "Circuit ID (omit for new)"),
			mcp.String("name", "Circuit name", mcp.Required()),
			mcp.String("circuit_id", "Provider circuit identifier"),
			mcp.String("provider", "ISP or provider name"),
			mcp.String("type", "Circuit type (fiber, copper, microwave, dark_fiber)"),
			mcp.String("status", "Status (active, maintenance, down, decommissioned)"),
			mcp.Number("capacity_mbps", "Bandwidth capacity in Mbps"),
			mcp.String("datacenter_a_id", "Endpoint A datacenter ID"),
			mcp.String("datacenter_b_id", "Endpoint B datacenter ID"),
			mcp.String("device_a_id", "Device at endpoint A"),
			mcp.String("device_b_id", "Device at endpoint B"),
			mcp.String("port_a", "Port/interface at endpoint A"),
			mcp.String("port_b", "Port/interface at endpoint B"),
			mcp.String("ip_address_a", "IP address at endpoint A"),
			mcp.String("ip_address_b", "IP address at endpoint B"),
			mcp.Number("vlan_id", "VLAN ID"),
			mcp.String("description", "Description"),
			mcp.Number("monthly_cost", "Monthly cost"),
			mcp.String("contract_number", "Contract reference"),
			mcp.String("contact_name", "Provider contact name"),
			mcp.String("contact_phone", "Provider contact phone"),
			mcp.String("contact_email", "Provider contact email"),
			mcp.StringArray("tags", "Tags"),
		).Discoverable("circuit", "wan", "link", "create", "update", "provider"),
		s.handleCircuitSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("circuit_delete", "Delete a circuit",
			mcp.String("id", "Circuit ID", mcp.Required()),
		).Discoverable("circuit", "delete", "remove"),
		s.handleCircuitDelete,
	)
}

func (s *Server) handleCircuitList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	filter := &model.CircuitFilter{
		Provider:     req.StringOr("provider", ""),
		Status:       model.CircuitStatus(req.StringOr("status", "")),
		DatacenterID: req.StringOr("datacenter_id", ""),
		Type:         req.StringOr("type", ""),
	}
	circuits, err := s.svc.Circuits.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(circuits), nil
}

func (s *Server) handleCircuitGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	circuit, err := s.svc.Circuits.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(circuit), nil
}

func (s *Server) handleCircuitSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")

	if id == "" {
		createReq := &model.CreateCircuitRequest{
			Name:           name,
			CircuitID:      req.StringOr("circuit_id", ""),
			Provider:       req.StringOr("provider", ""),
			Type:           req.StringOr("type", ""),
			Status:         model.CircuitStatus(req.StringOr("status", string(model.CircuitStatusActive))),
			CapacityMbps:   req.IntOr("capacity_mbps", 0),
			DatacenterAID:  req.StringOr("datacenter_a_id", ""),
			DatacenterBID:  req.StringOr("datacenter_b_id", ""),
			DeviceAID:      req.StringOr("device_a_id", ""),
			DeviceBID:      req.StringOr("device_b_id", ""),
			PortA:          req.StringOr("port_a", ""),
			PortB:          req.StringOr("port_b", ""),
			IPAddressA:     req.StringOr("ip_address_a", ""),
			IPAddressB:     req.StringOr("ip_address_b", ""),
			VLANID:         req.IntOr("vlan_id", 0),
			Description:    req.StringOr("description", ""),
			MonthlyCost:    req.FloatOr("monthly_cost", 0),
			ContractNumber: req.StringOr("contract_number", ""),
			ContactName:    req.StringOr("contact_name", ""),
			ContactPhone:   req.StringOr("contact_phone", ""),
			ContactEmail:   req.StringOr("contact_email", ""),
			Tags:           req.StringSliceOr("tags", []string{}),
		}
		circuit, err := s.svc.Circuits.Create(ctx, createReq)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(circuit), nil
	}

	// Update
	nameStr := name
	statusStr := model.CircuitStatus(req.StringOr("status", ""))
	updateReq := &model.UpdateCircuitRequest{
		Name:   &nameStr,
		Status: &statusStr,
	}
	if v := req.StringOr("circuit_id", ""); v != "" {
		updateReq.CircuitID = &v
	}
	if v := req.StringOr("provider", ""); v != "" {
		updateReq.Provider = &v
	}
	if v := req.StringOr("description", ""); v != "" {
		updateReq.Description = &v
	}

	circuit, err := s.svc.Circuits.Update(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(circuit), nil
}

func (s *Server) handleCircuitDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Circuits.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}
