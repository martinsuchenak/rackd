package service

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type CircuitService struct {
	store storage.ExtendedStorage
}

func NewCircuitService(store storage.ExtendedStorage) *CircuitService {
	return &CircuitService{store: store}
}

// List returns all circuits with optional filtering
func (s *CircuitService) List(ctx context.Context, filter *model.CircuitFilter) ([]model.Circuit, error) {
	if err := requirePermission(ctx, s.store, "circuits", "list"); err != nil {
		return nil, err
	}

	return s.store.ListCircuits(filter)
}

// Get returns a single circuit by ID
func (s *CircuitService) Get(ctx context.Context, id string) (*model.Circuit, error) {
	if err := requirePermission(ctx, s.store, "circuits", "read"); err != nil {
		return nil, err
	}

	circuit, err := s.store.GetCircuit(id)
	if err != nil {
		if err == storage.ErrCircuitNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return circuit, nil
}

// GetByCircuitID returns a single circuit by provider's circuit ID
func (s *CircuitService) GetByCircuitID(ctx context.Context, circuitID string) (*model.Circuit, error) {
	if err := requirePermission(ctx, s.store, "circuits", "read"); err != nil {
		return nil, err
	}

	circuit, err := s.store.GetCircuitByCircuitID(circuitID)
	if err != nil {
		if err == storage.ErrCircuitNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return circuit, nil
}

// Create creates a new circuit
func (s *CircuitService) Create(ctx context.Context, req *model.CreateCircuitRequest) (*model.Circuit, error) {
	if err := requirePermission(ctx, s.store, "circuits", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.CircuitID == "" {
		return nil, ValidationErrors{{Field: "circuit_id", Message: "Circuit ID is required"}}
	}
	if req.Provider == "" {
		return nil, ValidationErrors{{Field: "provider", Message: "Provider is required"}}
	}

	// Validate status
	if req.Status != "" && !req.Status.IsValid() {
		return nil, ValidationErrors{{Field: "status", Message: "Invalid status: " + string(req.Status)}}
	}

	// Set default status
	if req.Status == "" {
		req.Status = model.CircuitStatusActive
	}

	circuit := &model.Circuit{
		Name:           req.Name,
		CircuitID:      req.CircuitID,
		Provider:       req.Provider,
		Type:           req.Type,
		Status:         req.Status,
		CapacityMbps:   req.CapacityMbps,
		DatacenterAID:  req.DatacenterAID,
		DatacenterBID:  req.DatacenterBID,
		DeviceAID:      req.DeviceAID,
		DeviceBID:      req.DeviceBID,
		PortA:          req.PortA,
		PortB:          req.PortB,
		IPAddressA:     req.IPAddressA,
		IPAddressB:     req.IPAddressB,
		VLANID:         req.VLANID,
		Description:    req.Description,
		InstallDate:    req.InstallDate,
		MonthlyCost:    req.MonthlyCost,
		ContractNumber: req.ContractNumber,
		ContactName:    req.ContactName,
		ContactPhone:   req.ContactPhone,
		ContactEmail:   req.ContactEmail,
		Tags:           req.Tags,
	}

	if err := s.store.CreateCircuit(ctx, circuit); err != nil {
		return nil, err
	}

	return circuit, nil
}

// Update updates an existing circuit
func (s *CircuitService) Update(ctx context.Context, id string, req *model.UpdateCircuitRequest) (*model.Circuit, error) {
	if err := requirePermission(ctx, s.store, "circuits", "update"); err != nil {
		return nil, err
	}

	circuit, err := s.store.GetCircuit(id)
	if err != nil {
		if err == storage.ErrCircuitNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ValidationErrors{{Field: "name", Message: "Name cannot be empty"}}
		}
		circuit.Name = *req.Name
	}
	if req.CircuitID != nil {
		if *req.CircuitID == "" {
			return nil, ValidationErrors{{Field: "circuit_id", Message: "Circuit ID cannot be empty"}}
		}
		circuit.CircuitID = *req.CircuitID
	}
	if req.Provider != nil {
		circuit.Provider = *req.Provider
	}
	if req.Type != nil {
		circuit.Type = *req.Type
	}
	if req.Status != nil {
		if !req.Status.IsValid() {
			return nil, ValidationErrors{{Field: "status", Message: "Invalid status: " + string(*req.Status)}}
		}
		circuit.Status = *req.Status
	}
	if req.CapacityMbps != nil {
		circuit.CapacityMbps = *req.CapacityMbps
	}
	if req.DatacenterAID != nil {
		circuit.DatacenterAID = *req.DatacenterAID
	}
	if req.DatacenterBID != nil {
		circuit.DatacenterBID = *req.DatacenterBID
	}
	if req.DeviceAID != nil {
		circuit.DeviceAID = *req.DeviceAID
	}
	if req.DeviceBID != nil {
		circuit.DeviceBID = *req.DeviceBID
	}
	if req.PortA != nil {
		circuit.PortA = *req.PortA
	}
	if req.PortB != nil {
		circuit.PortB = *req.PortB
	}
	if req.IPAddressA != nil {
		circuit.IPAddressA = *req.IPAddressA
	}
	if req.IPAddressB != nil {
		circuit.IPAddressB = *req.IPAddressB
	}
	if req.VLANID != nil {
		circuit.VLANID = *req.VLANID
	}
	if req.Description != nil {
		circuit.Description = *req.Description
	}
	if req.InstallDate != nil {
		circuit.InstallDate = req.InstallDate
	}
	if req.TerminateDate != nil {
		circuit.TerminateDate = req.TerminateDate
	}
	if req.MonthlyCost != nil {
		circuit.MonthlyCost = *req.MonthlyCost
	}
	if req.ContractNumber != nil {
		circuit.ContractNumber = *req.ContractNumber
	}
	if req.ContactName != nil {
		circuit.ContactName = *req.ContactName
	}
	if req.ContactPhone != nil {
		circuit.ContactPhone = *req.ContactPhone
	}
	if req.ContactEmail != nil {
		circuit.ContactEmail = *req.ContactEmail
	}
	if req.Tags != nil {
		circuit.Tags = *req.Tags
	}

	if err := s.store.UpdateCircuit(ctx, circuit); err != nil {
		return nil, err
	}

	return circuit, nil
}

// Delete deletes a circuit
func (s *CircuitService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "circuits", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteCircuit(ctx, id); err != nil {
		if err == storage.ErrCircuitNotFound {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// GetByDatacenter returns all circuits for a datacenter
func (s *CircuitService) GetByDatacenter(ctx context.Context, datacenterID string) ([]model.Circuit, error) {
	if err := requirePermission(ctx, s.store, "circuits", "list"); err != nil {
		return nil, err
	}

	return s.store.GetCircuitsByDatacenter(datacenterID)
}

// GetByDevice returns all circuits linked to a device
func (s *CircuitService) GetByDevice(ctx context.Context, deviceID string) ([]model.Circuit, error) {
	if err := requirePermission(ctx, s.store, "circuits", "list"); err != nil {
		return nil, err
	}

	return s.store.GetCircuitsByDevice(deviceID)
}
