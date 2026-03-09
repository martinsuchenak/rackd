package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Provider handlers

func (s *Server) handleDNSProviderList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	filter := &model.DNSProviderFilter{Pagination: pg}
	if t := req.StringOr("type", ""); t != "" {
		filter.Type = model.DNSProviderType(t)
	}
	providers, err := s.svc.DNS.ListProviders(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(providers, len(providers), pg)), nil
}

func (s *Server) handleDNSProviderGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	provider, err := s.svc.DNS.GetProvider(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(provider), nil
}

func (s *Server) handleDNSProviderSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")
	provType, _ := req.String("type")

	if id == "" {
		createReq := &model.CreateDNSProviderRequest{
			Name:        name,
			Type:        model.DNSProviderType(provType),
			Endpoint:    req.StringOr("endpoint", ""),
			Token:       req.StringOr("token", ""),
			Description: req.StringOr("description", ""),
		}
		provider, err := s.svc.DNS.CreateProvider(ctx, createReq)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(provider), nil
	}

	updateReq := &model.UpdateDNSProviderRequest{
		Name: &name,
	}
	pt := model.DNSProviderType(provType)
	updateReq.Type = &pt
	if v := req.StringOr("endpoint", ""); v != "" {
		updateReq.Endpoint = &v
	}
	if v := req.StringOr("token", ""); v != "" {
		updateReq.Token = &v
	}
	if v := req.StringOr("description", ""); v != "" {
		updateReq.Description = &v
	}

	provider, err := s.svc.DNS.UpdateProvider(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(provider), nil
}

func (s *Server) handleDNSProviderDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.DNS.DeleteProvider(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

func (s *Server) handleDNSProviderTest(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.DNS.TestProvider(ctx, id); err != nil {
		return mcp.NewToolResponseJSON(map[string]any{
			"success": false,
			"error":   err.Error(),
		}), nil
	}
	return mcp.NewToolResponseJSON(map[string]any{"success": true}), nil
}

// Zone handlers

func (s *Server) handleDNSZoneList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	filter := &model.DNSZoneFilter{Pagination: pg}
	if v := req.StringOr("provider_id", ""); v != "" {
		filter.ProviderID = v
	}
	if v := req.StringOr("network_id", ""); v != "" {
		filter.NetworkID = &v
	}
	zones, err := s.svc.DNS.ListZones(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(zones, len(zones), pg)), nil
}

func (s *Server) handleDNSZoneGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	zone, err := s.svc.DNS.GetZone(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(zone), nil
}

func (s *Server) handleDNSZoneSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")

	if id == "" {
		ttl := req.IntOr("ttl", 3600)
		var networkID *string
		if v := req.StringOr("network_id", ""); v != "" {
			networkID = &v
		}
		var ptrZone *string
		if v := req.StringOr("ptr_zone", ""); v != "" {
			ptrZone = &v
		}
		createReq := &model.CreateDNSZoneRequest{
			Name:        name,
			ProviderID:  req.StringOr("provider_id", ""),
			NetworkID:   networkID,
			AutoSync:    req.BoolOr("auto_sync", false),
			CreatePTR:   req.BoolOr("create_ptr", false),
			PTRZone:     ptrZone,
			TTL:         ttl,
			Description: req.StringOr("description", ""),
		}
		zone, err := s.svc.DNS.CreateZone(ctx, createReq)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(zone), nil
	}

	updateReq := &model.UpdateDNSZoneRequest{Name: &name}
	if v := req.StringOr("network_id", ""); v != "" {
		updateReq.NetworkID = &v
	}
	if v, err := req.Bool("auto_sync"); err == nil {
		updateReq.AutoSync = &v
	}
	if v, err := req.Bool("create_ptr"); err == nil {
		updateReq.CreatePTR = &v
	}
	if v := req.StringOr("ptr_zone", ""); v != "" {
		updateReq.PTRZone = &v
	}
	if v := req.IntOr("ttl", 0); v > 0 {
		updateReq.TTL = &v
	}
	if v := req.StringOr("description", ""); v != "" {
		updateReq.Description = &v
	}

	zone, err := s.svc.DNS.UpdateZone(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(zone), nil
}

func (s *Server) handleDNSZoneDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.DNS.DeleteZone(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

func (s *Server) handleDNSZoneSync(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	result, err := s.svc.DNS.SyncZone(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(result), nil
}

func (s *Server) handleDNSZoneImport(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	result, err := s.svc.DNS.ImportFromDNS(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(result), nil
}

// Record handlers

func (s *Server) handleDNSRecordList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	pg := mcpPagination(req)
	zoneID, _ := req.String("zone_id")
	filter := &model.DNSRecordFilter{Pagination: pg, ZoneID: zoneID}
	if v := req.StringOr("type", ""); v != "" {
		filter.Type = v
	}
	if v := req.StringOr("device_id", ""); v != "" {
		filter.DeviceID = &v
	}
	if v := req.StringOr("sync_status", ""); v != "" {
		ss := model.RecordSyncStatus(v)
		filter.SyncStatus = &ss
	}
	records, err := s.svc.DNS.ListRecords(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(paginatedResponse(records, len(records), pg)), nil
}

func (s *Server) handleDNSRecordGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	record, err := s.svc.DNS.GetRecord(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(record), nil
}

func (s *Server) handleDNSRecordSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")
	recType, _ := req.String("type")
	value, _ := req.String("value")

	if id == "" {
		zoneID := req.StringOr("zone_id", "")
		var deviceID *string
		if v := req.StringOr("device_id", ""); v != "" {
			deviceID = &v
		}
		createReq := &model.CreateDNSRecordRequest{
			ZoneID:   zoneID,
			DeviceID: deviceID,
			Name:     name,
			Type:     recType,
			Value:    value,
			TTL:      req.IntOr("ttl", 0),
		}
		record, err := s.svc.DNS.CreateRecord(ctx, createReq)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(record), nil
	}

	updateReq := &model.UpdateDNSRecordRequest{
		Name:  &name,
		Type:  &recType,
		Value: &value,
	}
	if v := req.StringOr("device_id", ""); v != "" {
		updateReq.DeviceID = &v
	}
	if v := req.IntOr("ttl", 0); v > 0 {
		updateReq.TTL = &v
	}

	record, err := s.svc.DNS.UpdateRecord(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(record), nil
}

func (s *Server) handleDNSRecordDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.DNS.DeleteRecord(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

func (s *Server) handleDNSRecordLink(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	deviceID, _ := req.String("device_id")
	linkReq := &model.LinkDNSRecordRequest{
		DeviceID:     deviceID,
		AddToDomains: req.BoolOr("add_to_domains", false),
	}
	if v := req.StringOr("address_id", ""); v != "" {
		linkReq.AddressID = &v
	}
	record, err := s.svc.DNS.LinkRecord(ctx, id, linkReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(record), nil
}
