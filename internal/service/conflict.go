package service

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"sort"
	"strings"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type ConflictService struct {
	store storage.ExtendedStorage
}

func NewConflictService(store storage.ExtendedStorage) *ConflictService {
	return &ConflictService{store: store}
}

// List returns all conflicts matching the filter
func (s *ConflictService) List(ctx context.Context, filter *model.ConflictFilter) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "list"); err != nil {
		return nil, err
	}

	s.reconcileOrLog(ctx)
	conflicts, err := s.store.ListConflicts(ctx, filter)
	if err != nil {
		return nil, err
	}

	return conflicts, nil
}

// Get returns a single conflict by ID
func (s *ConflictService) Get(ctx context.Context, id string) (*model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "read"); err != nil {
		return nil, err
	}

	conflict, err := s.store.GetConflict(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	return conflict, nil
}

// Resolve resolves a conflict by updating its status and recording who resolved it
func (s *ConflictService) Resolve(ctx context.Context, resolution *model.ConflictResolution) error {
	if err := requirePermission(ctx, s.store, "conflicts", "resolve"); err != nil {
		return err
	}

	if resolution.ConflictID == "" {
		return ValidationErrors{{Field: "conflict_id", Message: "Conflict ID is required"}}
	}

	caller := CallerFrom(ctx)
	resolvedBy := ""
	if caller != nil {
		resolvedBy = caller.UserID
	}

	// Get the conflict to determine its type
	conflict, err := s.store.GetConflict(ctx, resolution.ConflictID)
	if err != nil {
		return ErrNotFound
	}

	// For duplicate IP conflicts, update the addresses of devices that should lose the IP
	if conflict.Type == model.ConflictTypeDuplicateIP && resolution.KeepDeviceID != "" {
		// Taking the IP off the other devices changes them.
		if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
			return err
		}
		if err := s.resolveDuplicateIP(ctx, resolution, resolvedBy); err != nil {
			return err
		}
	}

	// Update conflict status to resolved
	if err := s.store.UpdateConflictStatus(enrichAuditCtx(ctx), resolution.ConflictID, model.ConflictStatusResolved, resolvedBy, resolution.Notes); err != nil {
		return err
	}
	if resolved, err := s.store.GetConflict(ctx, resolution.ConflictID); err == nil {
		publishEvent(model.EventTypeConflictResolved, resolved)
	}

	recordAudit(ctx, s.store, "resolve", "conflict", resolution.ConflictID, resolution)
	return nil
}

// resolveDuplicateIP removes the IP from devices that should not have it
func (s *ConflictService) resolveDuplicateIP(ctx context.Context, resolution *model.ConflictResolution, _ string) error {
	conflict, err := s.store.GetConflict(ctx, resolution.ConflictID)
	if err != nil {
		return err
	}

	// For each device that should lose the IP, remove the address
	for _, deviceID := range conflict.DeviceIDs {
		if deviceID == resolution.KeepDeviceID {
			continue // This device keeps the IP
		}

		// Get the device
		device, err := s.store.GetDevice(ctx, deviceID)
		if err != nil {
			continue // Skip if device not found
		}

		// Remove the conflicting IP from addresses
		var newAddresses []model.Address
		for _, addr := range device.Addresses {
			if addr.IP != conflict.IPAddress {
				newAddresses = append(newAddresses, addr)
			}
		}

		device.Addresses = newAddresses

		// Update the device
		if err := s.store.UpdateDevice(enrichAuditCtx(ctx), device); err != nil {
			return err
		}
	}

	return nil
}

// DetectDuplicateIPs scans for and creates conflict records for duplicate IPs
func (s *ConflictService) DetectDuplicateIPs(ctx context.Context) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "detect"); err != nil {
		return nil, err
	}

	conflicts, err := s.store.FindDuplicateIPs(ctx)
	if err != nil {
		return nil, err
	}
	// Recorded conflicts first catch up with the inventory, so those still
	// open are the ones that still apply.
	if err := s.reconcileWith(ctx, conflicts); err != nil {
		return nil, err
	}

	// Record the duplicates that have no open conflict yet.
	for _, conflict := range conflicts {
		existing, err := s.store.GetConflictsByIP(ctx, conflict.IPAddress)
		if err != nil {
			return nil, err
		}
		if !hasOpenConflict(existing) {
			// Create new conflict record, unless a concurrent write just did.
			if err := s.store.CreateConflict(ctx, &conflict); err != nil {
				if errors.Is(err, storage.ErrConflictRecorded) {
					continue
				}
				return nil, err
			}
			publishEvent(model.EventTypeConflictDetected, conflict)
		}
	}

	// The recorded conflicts, with their IDs, rather than the scan's rows.
	return s.store.ListConflicts(ctx, &model.ConflictFilter{Type: model.ConflictTypeDuplicateIP, Status: model.ConflictStatusActive})
}

// DetectOverlappingSubnets scans for and creates conflict records for overlapping subnets
func (s *ConflictService) DetectOverlappingSubnets(ctx context.Context) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "detect"); err != nil {
		return nil, err
	}

	if err := s.reconcile(ctx); err != nil {
		return nil, err
	}
	// Read the recorded conflicts once, and skip them while searching, so the
	// per-run cap applies to new overlaps only.
	existing, err := s.store.ListConflicts(ctx, &model.ConflictFilter{
		Type:   model.ConflictTypeOverlappingSubnet,
		Status: model.ConflictStatusActive,
	})
	if err != nil {
		return nil, err
	}
	recorded := make(map[string]bool, len(existing))
	for _, ex := range existing {
		recorded[networkPairKey(ex.NetworkIDs)] = true
	}

	conflicts, err := s.store.FindOverlappingSubnets(ctx, func(ids []string) bool {
		return recorded[networkPairKey(ids)]
	})
	if err != nil {
		return nil, err
	}
	for _, conflict := range conflicts {
		if err := s.store.CreateConflict(ctx, &conflict); err != nil {
			return nil, err
		}
		publishEvent(model.EventTypeConflictDetected, conflict)
	}

	return conflicts, nil
}

// networkPairKey identifies a set of networks regardless of order.
func networkPairKey(ids []string) string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	return strings.Join(sorted, "\x00")
}

// GetConflictsByDevice returns all conflicts involving a specific device
func (s *ConflictService) GetConflictsByDevice(ctx context.Context, deviceID string) ([]model.Conflict, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "read"); err != nil {
		return nil, err
	}

	return s.store.GetConflictsByDeviceID(ctx, deviceID)
}

// Delete removes a conflict (requires conflict:delete permission)
func (s *ConflictService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "conflicts", "delete"); err != nil {
		return err
	}

	if id == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if err := s.store.DeleteConflict(ctx, id); err != nil {
		if errors.Is(err, storage.ErrConflictNotFound) {
			return ErrNotFound
		}
		return err
	}

	recordAudit(ctx, s.store, "delete", "conflict", id, nil)
	return nil
}

// GetSummary returns a summary of active conflicts
func (s *ConflictService) GetSummary(ctx context.Context) (map[string]int, error) {
	if err := requirePermission(ctx, s.store, "conflicts", "read"); err != nil {
		return nil, err
	}

	s.reconcileOrLog(ctx)
	conflicts, err := s.store.ListConflicts(ctx, &model.ConflictFilter{
		Status: model.ConflictStatusActive,
	})
	if err != nil {
		return nil, err
	}

	summary := make(map[string]int)
	for _, c := range conflicts {
		summary[string(c.Type)]++
	}

	return summary, nil
}

// hasOpenConflict reports whether any of an address's conflicts is still
// active or was ignored; a resolved one doesn't stop a new duplicate being
// recorded.
func hasOpenConflict(conflicts []model.Conflict) bool {
	for _, c := range conflicts {
		if c.Type == model.ConflictTypeDuplicateIP && c.Status != model.ConflictStatusResolved {
			return true
		}
	}
	return false
}

// reconcile brings the recorded open conflicts in line with the inventory as
// it is now. Devices and networks change and get deleted without touching
// conflicts, so a duplicate IP or an overlap that no longer exists is
// resolved, and one whose devices or networks changed lists the current
// ones. It writes only what changed, so reading conflicts can run it; it
// never records new conflicts, which detection does.
func (s *ConflictService) reconcile(ctx context.Context) error {
	return s.reconcileWith(ctx, nil)
}

// reconcileWith is reconcile with the current duplicate IPs, when the caller
// has already found them (nil reads them if needed).
func (s *ConflictService) reconcileWith(ctx context.Context, found []model.Conflict) error {
	var open []model.Conflict
	for _, status := range []model.ConflictStatus{model.ConflictStatusActive, model.ConflictStatusIgnored} {
		cs, err := s.store.ListConflicts(ctx, &model.ConflictFilter{Status: status})
		if err != nil {
			return err
		}
		open = append(open, cs...)
	}
	if len(open) == 0 {
		return nil
	}

	// The current duplicates and networks, each read once when first needed.
	var duplicates map[string]model.Conflict
	var networks map[string]*model.Network
	for i := range open {
		c := &open[i]
		switch c.Type {
		case model.ConflictTypeDuplicateIP:
			if duplicates == nil {
				if found == nil {
					var err error
					if found, err = s.store.FindDuplicateIPs(ctx); err != nil {
						return err
					}
				}
				duplicates = make(map[string]model.Conflict, len(found))
				for _, f := range found {
					duplicates[f.IPAddress] = f
				}
			}
			now, ok := duplicates[c.IPAddress]
			if !ok {
				if err := s.resolveStale(ctx, c, "No longer assigned to more than one device."); err != nil {
					return err
				}
				continue
			}
			if !slices.Equal(c.DeviceIDs, now.DeviceIDs) || !slices.Equal(c.DeviceNames, now.DeviceNames) {
				c.DeviceIDs, c.DeviceNames, c.Description = now.DeviceIDs, now.DeviceNames, now.Description
				if err := s.store.UpdateConflictParties(ctx, c); err != nil {
					return err
				}
			}
		case model.ConflictTypeOverlappingSubnet:
			if networks == nil {
				all, err := s.store.ListNetworks(ctx, &model.NetworkFilter{})
				if err != nil {
					return err
				}
				networks = make(map[string]*model.Network, len(all))
				for j := range all {
					networks[all[j].ID] = &all[j]
				}
			}
			if err := s.reconcileOverlap(ctx, c, networks); err != nil {
				return err
			}
		}
	}
	return nil
}

// reconcileOverlap resolves an overlap whose networks are gone or no longer
// overlap, and otherwise records their current names and subnets.
func (s *ConflictService) reconcileOverlap(ctx context.Context, c *model.Conflict, networks map[string]*model.Network) error {
	var names, subnets []string
	var prefixes []netip.Prefix
	for _, id := range c.NetworkIDs {
		n, ok := networks[id]
		if !ok {
			return s.resolveStale(ctx, c, "A network in this overlap was deleted.")
		}
		p, err := netip.ParsePrefix(n.Subnet)
		if err != nil {
			return s.resolveStale(ctx, c, "A network in this overlap no longer has a valid subnet.")
		}
		names, subnets, prefixes = append(names, n.Name), append(subnets, n.Subnet), append(prefixes, p.Masked())
	}
	if len(prefixes) != 2 || !prefixes[0].Overlaps(prefixes[1]) {
		return s.resolveStale(ctx, c, "The subnets no longer overlap.")
	}
	if slices.Equal(c.NetworkNames, names) && slices.Equal(c.Subnets, subnets) {
		return nil
	}
	c.NetworkNames, c.Subnets = names, subnets
	c.Description = fmt.Sprintf("Subnets %s and %s overlap", subnets[0], subnets[1])
	return s.store.UpdateConflictParties(ctx, c)
}

// resolveStale resolves a conflict whose condition no longer exists, keeping
// any notes on it.
func (s *ConflictService) resolveStale(ctx context.Context, c *model.Conflict, why string) error {
	notes := why
	if c.Notes != "" {
		notes = c.Notes + "\n\n" + why
	}
	// Only one of two reads reconciling at once resolves it and announces it.
	resolvedNow, err := s.store.ResolveOpenConflict(ctx, c.ID, "rackd", notes)
	if err != nil || !resolvedNow {
		return err
	}
	if resolved, err := s.store.GetConflict(ctx, c.ID); err == nil {
		publishEvent(model.EventTypeConflictResolved, resolved)
	}
	return nil
}

// reconcileOrLog runs reconcile for a read, which still answers if it fails.
func (s *ConflictService) reconcileOrLog(ctx context.Context) {
	if err := s.reconcile(ctx); err != nil {
		log.Warn("Failed to bring conflicts up to date", "error", err)
	}
}
