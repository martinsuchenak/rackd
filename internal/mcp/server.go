package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Server struct {
	mcpServer   *mcp.Server
	storage     storage.ExtendedStorage
	bearerToken string
}

func NewServer(store storage.ExtendedStorage, bearerToken string) *Server {
	s := &Server{
		mcpServer:   mcp.NewServer("rackd", "1.0.0"),
		storage:     store,
		bearerToken: bearerToken,
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
	if s.bearerToken != "" {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.bearerToken)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
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
		device.ID = uuid.Must(uuid.NewV7()).String()
		if err := s.storage.CreateDevice(device); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	} else {
		if err := s.storage.UpdateDevice(device); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	}

	return mcp.NewToolResponseJSON(device), nil
}

func (s *Server) handleDeviceGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	device, err := s.storage.GetDevice(id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(device), nil
}

func (s *Server) handleDeviceList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	query := req.StringOr("query", "")
	if query != "" {
		devices, err := s.storage.SearchDevices(query)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(devices), nil
	}

	filter := &model.DeviceFilter{
		Tags:         req.StringSliceOr("tags", nil),
		DatacenterID: req.StringOr("datacenter_id", ""),
	}
	devices, err := s.storage.ListDevices(filter)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(devices), nil
}

func (s *Server) handleDeviceDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.storage.DeleteDevice(id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

// Relationship handlers

func (s *Server) handleAddRelationship(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	parentID, _ := req.String("parent_id")
	childID, _ := req.String("child_id")
	relType, _ := req.String("type")

	if relType != model.RelationshipContains && relType != model.RelationshipConnectedTo && relType != model.RelationshipDependsOn {
		return nil, mcp.NewToolErrorInvalidParams("type must be one of: contains, connected_to, depends_on")
	}

	if err := s.storage.AddRelationship(parentID, childID, relType); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "created"}), nil
}

func (s *Server) handleGetRelationships(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	rels, err := s.storage.GetRelationships(id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(rels), nil
}

// Datacenter handlers

func (s *Server) handleDatacenterList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	dcs, err := s.storage.ListDatacenters(nil)
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
		dc.ID = uuid.Must(uuid.NewV7()).String()
		if err := s.storage.CreateDatacenter(dc); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	} else {
		if err := s.storage.UpdateDatacenter(dc); err != nil {
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
	networks, err := s.storage.ListNetworks(filter)
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
		network.ID = uuid.Must(uuid.NewV7()).String()
		if err := s.storage.CreateNetwork(network); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	} else {
		if err := s.storage.UpdateNetwork(network); err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
	}

	return mcp.NewToolResponseJSON(network), nil
}

// Pool handlers

func (s *Server) handleGetNextIP(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	poolID, _ := req.String("pool_id")
	ip, err := s.storage.GetNextAvailableIP(poolID)
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

	now := time.Now()
	scan := &model.DiscoveryScan{
		ID:        uuid.Must(uuid.NewV7()).String(),
		NetworkID: networkID,
		Status:    model.ScanStatusPending,
		ScanType:  scanType,
		StartedAt: &now,
	}

	if err := s.storage.CreateDiscoveryScan(scan); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}

	return mcp.NewToolResponseJSON(scan), nil
}

func (s *Server) handleListDiscovered(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	networkID := req.StringOr("network_id", "")
	devices, err := s.storage.ListDiscoveredDevices(networkID)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(devices), nil
}

func (s *Server) handlePromoteDevice(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	discoveredID, _ := req.String("discovered_id")
	name, _ := req.String("name")

	discovered, err := s.storage.GetDiscoveredDevice(discoveredID)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}

	device := &model.Device{
		ID:   uuid.Must(uuid.NewV7()).String(),
		Name: name,
		Addresses: []model.Address{
			{IP: discovered.IP, Type: "ipv4"},
		},
		Tags:    []string{},
		Domains: []string{},
	}
	if discovered.Hostname != "" {
		device.Domains = append(device.Domains, discovered.Hostname)
	}

	if err := s.storage.CreateDevice(device); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}

	if err := s.storage.PromoteDiscoveredDevice(discoveredID, device.ID); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}

	return mcp.NewToolResponseJSON(device), nil
}

// toJSON is a helper for JSON serialization
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
