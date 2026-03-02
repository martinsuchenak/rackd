package service

import (
	"context"
	"net"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type NATService struct {
	store storage.ExtendedStorage
}

func NewNATService(store storage.ExtendedStorage) *NATService {
	return &NATService{store: store}
}

// List returns all NAT mappings with optional filtering
func (s *NATService) List(ctx context.Context, filter *model.NATFilter) ([]model.NATMapping, error) {
	if err := requirePermission(ctx, s.store, "nat", "list"); err != nil {
		return nil, err
	}

	return s.store.ListNATMappings(filter)
}

// Get returns a single NAT mapping by ID
func (s *NATService) Get(ctx context.Context, id string) (*model.NATMapping, error) {
	if err := requirePermission(ctx, s.store, "nat", "read"); err != nil {
		return nil, err
	}

	mapping, err := s.store.GetNATMapping(id)
	if err != nil {
		if err == storage.ErrNATNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return mapping, nil
}

// Create creates a new NAT mapping
func (s *NATService) Create(ctx context.Context, req *model.CreateNATRequest) (*model.NATMapping, error) {
	if err := requirePermission(ctx, s.store, "nat", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.ExternalIP == "" {
		return nil, ValidationErrors{{Field: "external_ip", Message: "External IP is required"}}
	}
	if net.ParseIP(req.ExternalIP) == nil {
		return nil, ValidationErrors{{Field: "external_ip", Message: "Invalid IP address"}}
	}
	if req.InternalIP == "" {
		return nil, ValidationErrors{{Field: "internal_ip", Message: "Internal IP is required"}}
	}
	if net.ParseIP(req.InternalIP) == nil {
		return nil, ValidationErrors{{Field: "internal_ip", Message: "Invalid IP address"}}
	}
	if req.ExternalPort < 0 || req.ExternalPort > 65535 {
		return nil, ValidationErrors{{Field: "external_port", Message: "External port must be between 0 and 65535"}}
	}
	if req.InternalPort < 0 || req.InternalPort > 65535 {
		return nil, ValidationErrors{{Field: "internal_port", Message: "Internal port must be between 0 and 65535"}}
	}

	// Validate protocol
	if req.Protocol != "" && !req.Protocol.IsValid() {
		return nil, ValidationErrors{{Field: "protocol", Message: "Invalid protocol: " + string(req.Protocol)}}
	}

	// Set defaults
	if req.Protocol == "" {
		req.Protocol = model.NATProtocolTCP
	}

	mapping := &model.NATMapping{
		Name:         req.Name,
		ExternalIP:   req.ExternalIP,
		ExternalPort: req.ExternalPort,
		InternalIP:   req.InternalIP,
		InternalPort: req.InternalPort,
		Protocol:     req.Protocol,
		DeviceID:     req.DeviceID,
		Description:  req.Description,
		Enabled:      req.Enabled,
		DatacenterID: req.DatacenterID,
		NetworkID:    req.NetworkID,
		Tags:         req.Tags,
	}

	if err := s.store.CreateNATMapping(ctx, mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

// Update updates an existing NAT mapping
func (s *NATService) Update(ctx context.Context, id string, req *model.UpdateNATRequest) (*model.NATMapping, error) {
	if err := requirePermission(ctx, s.store, "nat", "update"); err != nil {
		return nil, err
	}

	mapping, err := s.store.GetNATMapping(id)
	if err != nil {
		if err == storage.ErrNATNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ValidationErrors{{Field: "name", Message: "Name cannot be empty"}}
		}
		mapping.Name = *req.Name
	}
	if req.ExternalIP != nil {
		if net.ParseIP(*req.ExternalIP) == nil {
			return nil, ValidationErrors{{Field: "external_ip", Message: "Invalid IP address"}}
		}
		mapping.ExternalIP = *req.ExternalIP
	}
	if req.ExternalPort != nil {
		if *req.ExternalPort < 0 || *req.ExternalPort > 65535 {
			return nil, ValidationErrors{{Field: "external_port", Message: "External port must be between 0 and 65535"}}
		}
		mapping.ExternalPort = *req.ExternalPort
	}
	if req.InternalIP != nil {
		if net.ParseIP(*req.InternalIP) == nil {
			return nil, ValidationErrors{{Field: "internal_ip", Message: "Invalid IP address"}}
		}
		mapping.InternalIP = *req.InternalIP
	}
	if req.InternalPort != nil {
		if *req.InternalPort < 0 || *req.InternalPort > 65535 {
			return nil, ValidationErrors{{Field: "internal_port", Message: "Internal port must be between 0 and 65535"}}
		}
		mapping.InternalPort = *req.InternalPort
	}
	if req.Protocol != nil {
		if !req.Protocol.IsValid() {
			return nil, ValidationErrors{{Field: "protocol", Message: "Invalid protocol: " + string(*req.Protocol)}}
		}
		mapping.Protocol = *req.Protocol
	}
	if req.DeviceID != nil {
		mapping.DeviceID = *req.DeviceID
	}
	if req.Description != nil {
		mapping.Description = *req.Description
	}
	if req.Enabled != nil {
		mapping.Enabled = *req.Enabled
	}
	if req.DatacenterID != nil {
		mapping.DatacenterID = *req.DatacenterID
	}
	if req.NetworkID != nil {
		mapping.NetworkID = *req.NetworkID
	}
	if req.Tags != nil {
		mapping.Tags = *req.Tags
	}

	if err := s.store.UpdateNATMapping(ctx, mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

// Delete deletes a NAT mapping
func (s *NATService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "nat", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteNATMapping(ctx, id); err != nil {
		if err == storage.ErrNATNotFound {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// GetByDevice returns all NAT mappings for a device
func (s *NATService) GetByDevice(ctx context.Context, deviceID string) ([]model.NATMapping, error) {
	if err := requirePermission(ctx, s.store, "nat", "list"); err != nil {
		return nil, err
	}

	return s.store.GetNATMappingsByDevice(deviceID)
}

// GetByDatacenter returns all NAT mappings for a datacenter
func (s *NATService) GetByDatacenter(ctx context.Context, datacenterID string) ([]model.NATMapping, error) {
	if err := requirePermission(ctx, s.store, "nat", "list"); err != nil {
		return nil, err
	}

	return s.store.GetNATMappingsByDatacenter(datacenterID)
}
