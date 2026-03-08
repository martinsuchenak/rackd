package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerSearchTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("search", "Search across devices, networks, and datacenters",
			mcp.String("query", "Search query", mcp.Required()),
		),
		s.handleSearch,
	)
}

func (s *Server) registerDeviceTools() {
	// Native tools — core daily use
	s.mcpServer.RegisterTool(
		mcp.NewTool("device_list", "List devices with optional filters",
			mcp.String("query", "Search query"),
			mcp.StringArray("tags", "Filter by tags"),
			mcp.String("datacenter_id", "Filter by datacenter"),
			mcp.String("network_id", "Filter by network"),
			mcp.String("pool_id", "Filter by IP pool"),
			mcp.String("status", "Filter by status (planned, active, maintenance, decommissioned)"),
		),
		s.handleDeviceList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_get", "Get a device by ID",
			mcp.String("id", "Device ID", mcp.Required()),
		),
		s.handleDeviceGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_save", "Create or update a device",
			mcp.String("id", "Device ID (omit for new device)"),
			mcp.String("name", "Device name", mcp.Required()),
			mcp.String("hostname", "Hostname"),
			mcp.String("description", "Device description"),
			mcp.String("make_model", "Device make and model"),
			mcp.String("os", "Operating system"),
			mcp.String("status", "Status (planned, active, maintenance, decommissioned)"),
			mcp.String("datacenter_id", "Datacenter ID"),
			mcp.String("username", "Login username"),
			mcp.String("location", "Physical location"),
			mcp.StringArray("tags", "Device tags"),
			mcp.ObjectArray("addresses", "IP addresses", mcp.String("ip", "IP address"), mcp.String("type", "Address type")),
			mcp.StringArray("domains", "Domain names"),
			mcp.ObjectArray("custom_fields", "Custom field values",
				mcp.String("field_id", "Custom field definition ID"),
				mcp.String("value", "Field value"),
			),
		),
		s.handleDeviceSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_delete", "Delete a device",
			mcp.String("id", "Device ID", mcp.Required()),
		),
		s.handleDeviceDelete,
	)

	// Discoverable tools — less frequent
	s.mcpServer.RegisterTool(
		mcp.NewTool("device_add_relationship", "Add a relationship between devices",
			mcp.String("parent_id", "Parent device ID", mcp.Required()),
			mcp.String("child_id", "Child device ID", mcp.Required()),
			mcp.String("type", "Relationship type (contains, connected_to, depends_on)", mcp.Required()),
			mcp.String("notes", "Optional notes"),
		).Discoverable("device", "relationship", "link", "connect", "dependency"),
		s.handleAddRelationship,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_get_relationships", "Get all relationships for a device",
			mcp.String("id", "Device ID", mcp.Required()),
		).Discoverable("device", "relationship", "link", "connect", "dependency"),
		s.handleGetRelationships,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_get_custom_fields", "Get custom field values with definitions for a device",
			mcp.String("id", "Device ID", mcp.Required()),
		).Discoverable("device", "custom", "field", "metadata", "attribute"),
		s.handleDeviceGetCustomFields,
	)
}

// Search handler

func (s *Server) handleSearch(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	query, _ := req.String("query")
	devices, _ := s.svc.Devices.Search(ctx, query)
	networks, _ := s.svc.Networks.Search(ctx, query)
	datacenters, _ := s.svc.Datacenters.Search(ctx, query)
	return mcp.NewToolResponseJSON(map[string]interface{}{
		"devices":     devices,
		"networks":    networks,
		"datacenters": datacenters,
	}), nil
}

// Device handlers

func (s *Server) handleDeviceList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	query := req.StringOr("query", "")
	if query != "" {
		devices, err := s.svc.Devices.Search(ctx, query)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(devices), nil
	}

	filter := &model.DeviceFilter{
		Tags:         req.StringSliceOr("tags", nil),
		DatacenterID: req.StringOr("datacenter_id", ""),
		NetworkID:    req.StringOr("network_id", ""),
		PoolID:       req.StringOr("pool_id", ""),
		Status:       model.DeviceStatus(req.StringOr("status", "")),
	}
	devices, err := s.svc.Devices.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(devices), nil
}

func (s *Server) handleDeviceGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	device, err := s.svc.Devices.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(device), nil
}

func (s *Server) handleDeviceSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")

	device := &model.Device{
		ID:           id,
		Name:         name,
		Hostname:     req.StringOr("hostname", ""),
		Description:  req.StringOr("description", ""),
		MakeModel:    req.StringOr("make_model", ""),
		OS:           req.StringOr("os", ""),
		Status:       model.DeviceStatus(req.StringOr("status", "")),
		DatacenterID: req.StringOr("datacenter_id", ""),
		Username:     req.StringOr("username", ""),
		Location:     req.StringOr("location", ""),
		Tags:         req.StringSliceOr("tags", []string{}),
		Domains:      req.StringSliceOr("domains", []string{}),
	}

	for _, addr := range req.ObjectSliceOr("addresses", nil) {
		ip, _ := addr["ip"].(string)
		addrType, _ := addr["type"].(string)
		if ip != "" {
			device.Addresses = append(device.Addresses, model.Address{IP: ip, Type: addrType})
		}
	}

	if id == "" {
		if err := s.svc.Devices.Create(ctx, device); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	} else {
		if err := s.svc.Devices.Update(ctx, device); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	}

	// Apply custom fields if provided
	if cfRaw := req.ObjectSliceOr("custom_fields", nil); len(cfRaw) > 0 && s.svc.CustomFields != nil {
		var inputs []model.CustomFieldValueInput
		for _, cf := range cfRaw {
			fieldID, _ := cf["field_id"].(string)
			value, _ := cf["value"].(string)
			if fieldID != "" {
				inputs = append(inputs, model.CustomFieldValueInput{FieldID: fieldID, Value: value})
			}
		}
		if len(inputs) > 0 {
			_ = s.svc.CustomFields.SetValues(ctx, device.ID, inputs)
		}
	}

	return mcp.NewToolResponseJSON(device), nil
}

func (s *Server) handleDeviceDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Devices.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

func (s *Server) handleAddRelationship(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	parentID, _ := req.String("parent_id")
	childID, _ := req.String("child_id")
	relType, _ := req.String("type")
	notes := req.StringOr("notes", "")

	if relType != model.RelationshipContains && relType != model.RelationshipConnectedTo && relType != model.RelationshipDependsOn {
		return nil, mcp.NewToolErrorInvalidParams("type must be one of: contains, connected_to, depends_on")
	}

	if err := s.svc.Relationships.Add(ctx, parentID, childID, relType, notes); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "created"}), nil
}

func (s *Server) handleGetRelationships(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	rels, err := s.svc.Relationships.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(rels), nil
}

func (s *Server) handleDeviceGetCustomFields(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if s.svc.CustomFields == nil {
		return mcp.NewToolResponseJSON([]interface{}{}), nil
	}
	fields, err := s.svc.CustomFields.GetValuesWithDefinitions(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(fields), nil
}
