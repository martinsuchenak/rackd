package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Server struct {
	mcpServer    *mcp.Server
	svc          *service.Services
	store        storage.ExtendedStorage
	requireAuth  bool
	oauthService *service.OAuthService
	oauthEnabled bool
}

func (s *Server) SetOAuthService(svc *service.OAuthService) {
	s.oauthService = svc
	s.oauthEnabled = svc != nil
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
	// Search tool
	s.mcpServer.RegisterTool(
		mcp.NewTool("search", "Search across devices, networks, and datacenters",
			mcp.String("query", "Search query", mcp.Required()),
		),
		s.handleSearch,
	)

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
		mcp.NewTool("datacenter_get", "Get a datacenter by ID",
			mcp.String("id", "Datacenter ID", mcp.Required()),
		),
		s.handleDatacenterGet,
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

	s.mcpServer.RegisterTool(
		mcp.NewTool("datacenter_delete", "Delete a datacenter",
			mcp.String("id", "Datacenter ID", mcp.Required()),
		),
		s.handleDatacenterDelete,
	)

	// Network tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("network_list", "List all networks",
			mcp.String("datacenter_id", "Filter by datacenter"),
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

	s.mcpServer.RegisterTool(
		mcp.NewTool("network_delete", "Delete a network",
			mcp.String("id", "Network ID", mcp.Required()),
		),
		s.handleNetworkDelete,
	)

	// Pool tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("pool_list", "List IP pools for a network",
			mcp.String("network_id", "Network ID", mcp.Required()),
		),
		s.handlePoolList,
	)

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
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Debug("MCP auth failed: missing Bearer prefix")
			s.writeUnauthorized(w)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		var caller *service.Caller

		// Strategy 1: Try OAuth token validation (if OAuth enabled)
		if s.oauthService != nil {
			oauthToken, err := s.oauthService.ValidateAccessToken(token)
			if err == nil {
				caller, err = s.oauthService.ResolveCallerFromOAuthToken(oauthToken, r.RemoteAddr)
				if err != nil {
					log.Debug("MCP OAuth auth failed: could not resolve caller", "error", err)
					s.writeUnauthorized(w)
					return
				}
				log.Trace("MCP auth successful (OAuth)", "user_id", caller.UserID)
			}
		}

		// Strategy 2: Fall back to API key authentication
		if caller == nil {
			hash := auth.HashToken(token)
			key, err := s.store.GetAPIKeyByKey(r.Context(), hash)

			if err != nil || subtle.ConstantTimeCompare([]byte(hash), []byte(key.Key)) != 1 {
				log.Debug("MCP auth failed: invalid token")
				s.writeUnauthorized(w)
				return
			}

			// Check expiration
			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				log.Debug("MCP auth failed: expired API key", "key_name", key.Name)
				s.writeUnauthorized(w)
				return
			}

			// Update last used (async)
			go func() {
				s.store.UpdateAPIKeyLastUsed(context.Background(), key.ID, time.Now())
			}()

			log.Trace("MCP auth successful (API key)", "key_name", key.Name)

			// Resolve API key owner: if the key has a UserID, use the owner's
			// identity so RBAC is enforced using their roles.
			if key.UserID != "" {
				user, err := s.store.GetUser(r.Context(), key.UserID)
				if err == nil && user.IsActive {
					caller = &service.Caller{
						Type:      service.CallerTypeUser,
						UserID:    user.ID,
						Username:  user.Username,
						IPAddress: r.RemoteAddr,
						Source:    "mcp",
					}
				}
			}
			if caller == nil {
				// Legacy key (no user association)
				caller = &service.Caller{
					Type:     service.CallerTypeAPIKey,
					UserID:   key.ID,
					Username: key.Name,
					Source:   "mcp",
				}
			}
		}

		r = r.WithContext(service.WithCaller(r.Context(), caller))
	} else {
		// When auth is not required, inject a system caller so service
		// layer calls succeed (requirePermission bypasses RBAC for system callers)
		r = r.WithContext(service.SystemContext(r.Context(), "mcp"))
	}

	s.mcpServer.HandleRequest(w, r)
}

func (s *Server) writeUnauthorized(w http.ResponseWriter) {
	if s.oauthEnabled {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="/.well-known/oauth-protected-resource"`)
	}
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

func (s *Server) handleNetworkGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	network, err := s.svc.Networks.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
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
