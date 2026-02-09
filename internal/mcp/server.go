package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Server struct {
	mcpServer   *mcp.Server
	svc         *service.Services
	store       storage.ExtendedStorage
	requireAuth bool
}

func NewServer(services *service.Services, store storage.ExtendedStorage, requireAuth bool) *Server {
	s := &Server{
		mcpServer:   mcp.NewServer("rackd", "1.0.0"),
		svc:         services,
		store:       store,
		requireAuth: requireAuth,
	}
	s.registerTools()
	return s
}

func (s *Server) Inner() *mcp.Server {
	return s.mcpServer
}

func (s *Server) registerTools() {
	// Device tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("device_save", "Create or update a device",
			mcp.String("id", "Device ID (omit for new device)"),
			mcp.String("name", "Device name", mcp.Required()),
			mcp.String("description", "Device description"),
			mcp.String("make_model", "Device make and model"),
			mcp.String("os", "Operating system"),
			mcp.String("datacenter_id", "Datacenter ID"),
			mcp.String("username", "Login username"),
			mcp.String("location", "Physical location"),
			mcp.StringArray("tags", "Device tags"),
			mcp.ObjectArray("addresses", "IP addresses", mcp.String("ip", "IP address"), mcp.String("type", "Address type")),
			mcp.StringArray("domains", "Domain names"),
		),
		s.handleDeviceSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_get", "Get a device by ID",
			mcp.String("id", "Device ID", mcp.Required()),
		),
		s.handleDeviceGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_list", "List devices with optional filters",
			mcp.String("query", "Search query"),
			mcp.StringArray("tags", "Filter by tags"),
			mcp.String("datacenter_id", "Filter by datacenter"),
		),
		s.handleDeviceList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_delete", "Delete a device",
			mcp.String("id", "Device ID", mcp.Required()),
		),
		s.handleDeviceDelete,
	)

	// Relationship tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("device_add_relationship", "Add a relationship between devices",
			mcp.String("parent_id", "Parent device ID", mcp.Required()),
			mcp.String("child_id", "Child device ID", mcp.Required()),
			mcp.String("type", "Relationship type (contains, connected_to, depends_on)", mcp.Required()),
		),
		s.handleAddRelationship,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("device_get_relationships", "Get all relationships for a device",
			mcp.String("id", "Device ID", mcp.Required()),
		),
		s.handleGetRelationships,
	)

	// Datacenter tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("datacenter_list", "List all datacenters"),
		s.handleDatacenterList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("datacenter_save", "Create or update a datacenter",
			mcp.String("id", "Datacenter ID (omit for new)"),
			mcp.String("name", "Datacenter name", mcp.Required()),
			mcp.String("location", "Physical location"),
			mcp.String("description", "Description"),
		),
		s.handleDatacenterSave,
	)

	// Network tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("network_list", "List all networks",
			mcp.String("datacenter_id", "Filter by datacenter"),
		),
		s.handleNetworkList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("network_save", "Create or update a network",
			mcp.String("id", "Network ID (omit for new)"),
			mcp.String("name", "Network name", mcp.Required()),
			mcp.String("subnet", "CIDR subnet (e.g., 192.168.1.0/24)", mcp.Required()),
			mcp.String("datacenter_id", "Datacenter ID"),
			mcp.Number("vlan_id", "VLAN ID"),
			mcp.String("description", "Description"),
		),
		s.handleNetworkSave,
	)

	// Pool tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("pool_get_next_ip", "Get the next available IP from a pool",
			mcp.String("pool_id", "Pool ID", mcp.Required()),
		),
		s.handleGetNextIP,
	)

	// Discovery tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("discovery_scan", "Start a network discovery scan",
			mcp.String("network_id", "Network ID to scan", mcp.Required()),
			mcp.String("scan_type", "Scan type: quick, full, deep"),
		),
		s.handleStartScan,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("discovery_list", "List discovered devices",
			mcp.String("network_id", "Network ID"),
		),
		s.handleListDiscovered,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("discovery_promote", "Promote a discovered device to inventory",
			mcp.String("discovered_id", "Discovered device ID", mcp.Required()),
			mcp.String("name", "Device name", mcp.Required()),
		),
		s.handlePromoteDevice,
	)
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	log.Debug("MCP request received", "remote_addr", r.RemoteAddr)

	if s.requireAuth {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			log.Debug("MCP auth failed: missing Bearer prefix")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")

		// Try API key authentication
		key, err := s.store.GetAPIKeyByKey(token)
		if err != nil {
			log.Debug("MCP auth failed: invalid API key")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check expiration
		if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
			log.Debug("MCP auth failed: expired API key", "key_name", key.Name)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Update last used (async)
		go func() {
			s.store.UpdateAPIKeyLastUsed(key.ID, time.Now())
		}()

		log.Trace("MCP auth successful (API key)", "key_name", key.Name)

		// Inject caller context - API keys bypass RBAC (no user association)
		caller := &service.Caller{
			Type:     service.CallerTypeAPIKey,
			UserID:   key.ID,
			Username: key.Name,
			Source:   "mcp",
		}
		r = r.WithContext(service.WithCaller(r.Context(), caller))
	} else {
		// When auth is not required, inject a system caller so service
		// layer calls succeed (requirePermission bypasses RBAC for system callers)
		r = r.WithContext(service.SystemContext(r.Context(), "mcp"))
	}

	s.mcpServer.HandleRequest(w, r)
}

// Device handlers

func (s *Server) handleDeviceSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")

	device := &model.Device{
		ID:           id,
		Name:         name,
		Description:  req.StringOr("description", ""),
		MakeModel:    req.StringOr("make_model", ""),
		OS:           req.StringOr("os", ""),
		DatacenterID: req.StringOr("datacenter_id", ""),
		Username:     req.StringOr("username", ""),
		Location:     req.StringOr("location", ""),
		Tags:         req.StringSliceOr("tags", []string{}),
		Domains:      req.StringSliceOr("domains", []string{}),
	}

	// Parse addresses
	addrs := req.ObjectSliceOr("addresses", nil)
	for _, addr := range addrs {
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

	return mcp.NewToolResponseJSON(device), nil
}

func (s *Server) handleDeviceGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	device, err := s.svc.Devices.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(device), nil
}

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
	}
	devices, err := s.svc.Devices.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(devices), nil
}

func (s *Server) handleDeviceDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Devices.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

// Relationship handlers

func (s *Server) handleAddRelationship(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	parentID, _ := req.String("parent_id")
	childID, _ := req.String("child_id")
	relType, _ := req.String("type")
	notes, _ := req.String("notes")

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

// Datacenter handlers

func (s *Server) handleDatacenterList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	dcs, err := s.svc.Datacenters.List(ctx, nil)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(dcs), nil
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

// Network handlers

func (s *Server) handleNetworkList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	filter := &model.NetworkFilter{
		DatacenterID: req.StringOr("datacenter_id", ""),
	}
	networks, err := s.svc.Networks.List(ctx, filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(networks), nil
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

// Pool handlers

func (s *Server) handleGetNextIP(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	poolID, _ := req.String("pool_id")
	ip, err := s.svc.Pools.GetNextIP(ctx, poolID)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"ip": ip}), nil
}

// Discovery handlers

func (s *Server) handleStartScan(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	networkID, _ := req.String("network_id")
	scanType := req.StringOr("scan_type", model.ScanTypeQuick)

	if scanType != model.ScanTypeQuick && scanType != model.ScanTypeFull && scanType != model.ScanTypeDeep {
		scanType = model.ScanTypeQuick
	}

	scan, err := s.svc.Discovery.StartScan(ctx, networkID, scanType)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}

	return mcp.NewToolResponseJSON(scan), nil
}

func (s *Server) handleListDiscovered(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	networkID := req.StringOr("network_id", "")
	devices, err := s.svc.Discovery.ListDevices(ctx, networkID)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(devices), nil
}

func (s *Server) handlePromoteDevice(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	discoveredID, _ := req.String("discovered_id")
	name, _ := req.String("name")

	device := &model.Device{
		Name:    name,
		Tags:    []string{},
		Domains: []string{},
	}

	promoted, err := s.svc.Discovery.PromoteDevice(ctx, discoveredID, device)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}

	return mcp.NewToolResponseJSON(promoted), nil
}

// toJSON is a helper for JSON serialization
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
