package service

import (
	"context"
	"io"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func init() {
	log.Init("console", "error", io.Discard)
}

type serviceTestStorage struct {
	storage.ExtendedStorage

	permissions map[string]bool
	users       map[string]*model.User
	userRoles   map[string][]model.Role
	roles       map[string]*model.Role

	networks     []model.Network
	datacenters  []model.Datacenter
	customDefs   map[string]*model.CustomFieldDefinition
	devices      map[string]*model.Device
	reservations map[string]*model.Reservation
	natMappings  map[string]*model.NATMapping

	updatedUser    *model.User
	assignedUserID string
	assignedRoleID string

	natCreated      *model.NATMapping
	natUpdated      *model.NATMapping
	webhookCreated  *model.Webhook
	webhooks        map[string]*model.Webhook
	reservationMade *model.Reservation
	reservationSet  *model.Reservation
	deviceCreated   *model.Device
	deviceUpdated   *model.Device
	apiKeys         map[string]*model.APIKey
	apiKeyFilter    *model.APIKeyFilter
	deletedAPIKeyID string
	conflicts       map[string]*model.Conflict
	conflictStatus  struct {
		id         string
		status     model.ConflictStatus
		resolvedBy string
		notes      string
	}
	circuits         map[string]*model.Circuit
	circuitCreated   *model.Circuit
	circuitUpdated   *model.Circuit
	dashboardStaleDays int
	dashboardRecentLimit int
	utilTrendDays int
	bulkResult *storage.BulkResult
	lastBulkOp string
	auditLogs []model.AuditLog

	nextIPs               []string
	nextIPCalls           int
	createReservationErrs []error
	pools                 map[string]bool
	poolHeatmap           []storage.IPStatus

	removedParentID string
	removedChildID  string
	removedType     string
	removeErr       error
	addedParentID   string
	addedChildID    string
	addedType       string
	addedNotes      string

	setCustomFieldValues []model.CustomFieldValue
	deleteCustomFieldErr error
	deleteDeviceErr      error
	rules                map[string]*model.DiscoveryRule
	networkUtilization   *model.NetworkUtilization
	discoveryScans       map[string]*model.DiscoveryScan
	datacenterDevices    map[string][]model.Device
	networkDevices       map[string][]model.Device
	discoveredByNetwork  map[string][]model.DiscoveredDevice
}

func newServiceTestStorage() *serviceTestStorage {
	return &serviceTestStorage{
		permissions: make(map[string]bool),
		users:       make(map[string]*model.User),
		userRoles:   make(map[string][]model.Role),
		roles:       make(map[string]*model.Role),
		customDefs:  make(map[string]*model.CustomFieldDefinition),
		pools:       make(map[string]bool),
		devices:     make(map[string]*model.Device),
		reservations: make(map[string]*model.Reservation),
		natMappings: make(map[string]*model.NATMapping),
		webhooks:    make(map[string]*model.Webhook),
		apiKeys:     make(map[string]*model.APIKey),
		conflicts:   make(map[string]*model.Conflict),
		circuits:    make(map[string]*model.Circuit),
		rules:       make(map[string]*model.DiscoveryRule),
		discoveryScans: make(map[string]*model.DiscoveryScan),
		datacenterDevices: make(map[string][]model.Device),
		networkDevices: make(map[string][]model.Device),
		discoveredByNetwork: make(map[string][]model.DiscoveredDevice),
	}
}

func (s *serviceTestStorage) setPermission(userID, resource, action string, allowed bool) {
	s.permissions[userID+":"+resource+":"+action] = allowed
}

func (s *serviceTestStorage) HasPermission(_ context.Context, userID, resource, action string) (bool, error) {
	return s.permissions[userID+":"+resource+":"+action], nil
}

func (s *serviceTestStorage) GetUser(_ context.Context, id string) (*model.User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, storage.ErrUserNotFound
	}
	cloned := *user
	return &cloned, nil
}

func (s *serviceTestStorage) GetUserByUsername(_ context.Context, username string) (*model.User, error) {
	for _, user := range s.users {
		if user.Username == username {
			cloned := *user
			return &cloned, nil
		}
	}
	return nil, storage.ErrUserNotFound
}

func (s *serviceTestStorage) UpdateUser(_ context.Context, user *model.User) error {
	cloned := *user
	s.users[user.ID] = &cloned
	s.updatedUser = &cloned
	return nil
}

func (s *serviceTestStorage) GetUserRoles(_ context.Context, userID string) ([]model.Role, error) {
	return append([]model.Role(nil), s.userRoles[userID]...), nil
}

func (s *serviceTestStorage) GetRole(_ context.Context, id string) (*model.Role, error) {
	role, ok := s.roles[id]
	if !ok {
		return nil, storage.ErrRoleNotFound
	}
	cloned := *role
	return &cloned, nil
}

func (s *serviceTestStorage) CreateRole(_ context.Context, role *model.Role) error {
	cloned := *role
	s.roles[role.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) DeleteRole(_ context.Context, id string) error {
	if _, ok := s.roles[id]; !ok {
		return storage.ErrRoleNotFound
	}
	delete(s.roles, id)
	return nil
}

func (s *serviceTestStorage) AssignRoleToUser(_ context.Context, userID, roleID string) error {
	s.assignedUserID = userID
	s.assignedRoleID = roleID
	return nil
}

func (s *serviceTestStorage) SearchNetworks(_ context.Context, query string) ([]model.Network, error) {
	var results []model.Network
	for _, network := range s.networks {
		if network.Name == query {
			results = append(results, network)
		}
	}
	return results, nil
}

func (s *serviceTestStorage) SearchDatacenters(_ context.Context, query string) ([]model.Datacenter, error) {
	var results []model.Datacenter
	for _, dc := range s.datacenters {
		if dc.Name == query {
			results = append(results, dc)
		}
	}
	return results, nil
}

func (s *serviceTestStorage) ListDatacenters(_ context.Context, _ *model.DatacenterFilter) ([]model.Datacenter, error) {
	return append([]model.Datacenter(nil), s.datacenters...), nil
}

func (s *serviceTestStorage) CreateDatacenter(_ context.Context, dc *model.Datacenter) error {
	cloned := *dc
	s.datacenters = append(s.datacenters, cloned)
	return nil
}

func (s *serviceTestStorage) GetDatacenter(_ context.Context, id string) (*model.Datacenter, error) {
	for _, dc := range s.datacenters {
		if dc.ID == id {
			cloned := dc
			return &cloned, nil
		}
	}
	return nil, storage.ErrDatacenterNotFound
}

func (s *serviceTestStorage) UpdateDatacenter(_ context.Context, dc *model.Datacenter) error {
	for i := range s.datacenters {
		if s.datacenters[i].ID == dc.ID {
			s.datacenters[i] = *dc
			return nil
		}
	}
	return storage.ErrDatacenterNotFound
}

func (s *serviceTestStorage) DeleteDatacenter(_ context.Context, id string) error {
	for i := range s.datacenters {
		if s.datacenters[i].ID == id {
			s.datacenters = append(s.datacenters[:i], s.datacenters[i+1:]...)
			return nil
		}
	}
	return storage.ErrDatacenterNotFound
}

func (s *serviceTestStorage) GetDatacenterDevices(_ context.Context, datacenterID string) ([]model.Device, error) {
	return append([]model.Device(nil), s.datacenterDevices[datacenterID]...), nil
}

func (s *serviceTestStorage) GetCustomFieldDefinition(_ context.Context, id string) (*model.CustomFieldDefinition, error) {
	def, ok := s.customDefs[id]
	if !ok {
		return nil, storage.ErrCustomFieldNotFound
	}
	cloned := *def
	return &cloned, nil
}

func (s *serviceTestStorage) CreateCustomFieldDefinition(_ context.Context, def *model.CustomFieldDefinition) error {
	cloned := *def
	if cloned.ID == "" {
		cloned.ID = "field-created"
	}
	s.customDefs[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) GetNextAvailableIP(_ context.Context, poolID string) (string, error) {
	s.nextIPCalls++
	if !s.pools[poolID] {
		return "", storage.ErrPoolNotFound
	}
	if len(s.nextIPs) == 0 {
		return "", storage.ErrIPNotAvailable
	}
	ip := s.nextIPs[0]
	s.nextIPs = s.nextIPs[1:]
	return ip, nil
}

func (s *serviceTestStorage) CreateReservation(_ context.Context, reservation *model.Reservation) error {
	if len(s.createReservationErrs) > 0 {
		err := s.createReservationErrs[0]
		s.createReservationErrs = s.createReservationErrs[1:]
		if err != nil {
			return err
		}
	}
	cloned := *reservation
	s.reservationMade = &cloned
	s.reservations[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) GetNetwork(_ context.Context, id string) (*model.Network, error) {
	for _, network := range s.networks {
		if network.ID == id {
			cloned := network
			return &cloned, nil
		}
	}
	return nil, storage.ErrNetworkNotFound
}

func (s *serviceTestStorage) CreateNATMapping(_ context.Context, mapping *model.NATMapping) error {
	cloned := *mapping
	s.natCreated = &cloned
	s.natMappings[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) GetReservation(_ context.Context, id string) (*model.Reservation, error) {
	r, ok := s.reservations[id]
	if !ok {
		return nil, storage.ErrReservationNotFound
	}
	cloned := *r
	return &cloned, nil
}

func (s *serviceTestStorage) GetReservationByIP(_ context.Context, poolID, ip string) (*model.Reservation, error) {
	for _, r := range s.reservations {
		if r.PoolID == poolID && r.IPAddress == ip {
			cloned := *r
			return &cloned, nil
		}
	}
	return nil, storage.ErrReservationNotFound
}

func (s *serviceTestStorage) UpdateReservation(_ context.Context, reservation *model.Reservation) error {
	cloned := *reservation
	s.reservations[cloned.ID] = &cloned
	s.reservationSet = &cloned
	return nil
}

func (s *serviceTestStorage) GetNetworkPool(_ context.Context, id string) (*model.NetworkPool, error) {
	if !s.pools[id] {
		return nil, storage.ErrPoolNotFound
	}
	return &model.NetworkPool{ID: id}, nil
}

func (s *serviceTestStorage) GetPoolHeatmap(_ context.Context, poolID string) ([]storage.IPStatus, error) {
	if !s.pools[poolID] {
		return nil, storage.ErrPoolNotFound
	}
	return append([]storage.IPStatus(nil), s.poolHeatmap...), nil
}

func (s *serviceTestStorage) GetNATMapping(_ context.Context, id string) (*model.NATMapping, error) {
	m, ok := s.natMappings[id]
	if !ok {
		return nil, storage.ErrNATNotFound
	}
	cloned := *m
	return &cloned, nil
}

func (s *serviceTestStorage) UpdateNATMapping(_ context.Context, mapping *model.NATMapping) error {
	cloned := *mapping
	s.natMappings[cloned.ID] = &cloned
	s.natUpdated = &cloned
	return nil
}

func (s *serviceTestStorage) DeleteNATMapping(_ context.Context, id string) error {
	if _, ok := s.natMappings[id]; !ok {
		return storage.ErrNATNotFound
	}
	delete(s.natMappings, id)
	return nil
}

func (s *serviceTestStorage) CreateWebhook(_ context.Context, webhook *model.Webhook) error {
	cloned := *webhook
	s.webhookCreated = &cloned
	if cloned.ID == "" {
		cloned.ID = "webhook-created"
	}
	s.webhooks[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) GetWebhook(_ context.Context, id string) (*model.Webhook, error) {
	webhook, ok := s.webhooks[id]
	if !ok {
		return nil, storage.ErrWebhookNotFound
	}
	cloned := *webhook
	return &cloned, nil
}

func (s *serviceTestStorage) UpdateWebhook(_ context.Context, webhook *model.Webhook) error {
	cloned := *webhook
	s.webhooks[cloned.ID] = &cloned
	s.webhookCreated = &cloned
	return nil
}

func (s *serviceTestStorage) DeleteWebhook(_ context.Context, id string) error {
	if _, ok := s.webhooks[id]; !ok {
		return storage.ErrWebhookNotFound
	}
	delete(s.webhooks, id)
	return nil
}

func (s *serviceTestStorage) ListCustomFieldDefinitions(_ context.Context, _ *model.CustomFieldDefinitionFilter) ([]model.CustomFieldDefinition, error) {
	defs := make([]model.CustomFieldDefinition, 0, len(s.customDefs))
	for _, def := range s.customDefs {
		defs = append(defs, *def)
	}
	return defs, nil
}

func (s *serviceTestStorage) SetCustomFieldValue(_ context.Context, value *model.CustomFieldValue) error {
	cloned := *value
	s.setCustomFieldValues = append(s.setCustomFieldValues, cloned)
	return nil
}

func (s *serviceTestStorage) DeleteCustomFieldValue(_ context.Context, _, _ string) error {
	return s.deleteCustomFieldErr
}

func (s *serviceTestStorage) CreateDevice(_ context.Context, device *model.Device) error {
	cloned := *device
	s.devices[cloned.ID] = &cloned
	s.deviceCreated = &cloned
	return nil
}

func (s *serviceTestStorage) UpdateDevice(_ context.Context, device *model.Device) error {
	cloned := *device
	s.devices[cloned.ID] = &cloned
	s.deviceUpdated = &cloned
	return nil
}

func (s *serviceTestStorage) SearchDevices(_ context.Context, query string) ([]model.Device, error) {
	var results []model.Device
	for _, device := range s.devices {
		if device.Name == query {
			results = append(results, *device)
		}
	}
	return results, nil
}

func (s *serviceTestStorage) ListNetworks(_ context.Context, _ *model.NetworkFilter) ([]model.Network, error) {
	return append([]model.Network(nil), s.networks...), nil
}

func (s *serviceTestStorage) CreateNetwork(_ context.Context, network *model.Network) error {
	cloned := *network
	s.networks = append(s.networks, cloned)
	return nil
}

func (s *serviceTestStorage) UpdateNetwork(_ context.Context, network *model.Network) error {
	for i := range s.networks {
		if s.networks[i].ID == network.ID {
			s.networks[i] = *network
			return nil
		}
	}
	return storage.ErrNetworkNotFound
}

func (s *serviceTestStorage) DeleteNetwork(_ context.Context, id string) error {
	for i := range s.networks {
		if s.networks[i].ID == id {
			s.networks = append(s.networks[:i], s.networks[i+1:]...)
			return nil
		}
	}
	return storage.ErrNetworkNotFound
}

func (s *serviceTestStorage) GetNetworkDevices(_ context.Context, networkID string) ([]model.Device, error) {
	return append([]model.Device(nil), s.networkDevices[networkID]...), nil
}

func (s *serviceTestStorage) GetNetworkUtilization(_ context.Context, networkID string) (*model.NetworkUtilization, error) {
	if !s.pools[networkID] && s.networkUtilization == nil {
		return nil, storage.ErrNetworkNotFound
	}
	if s.networkUtilization == nil {
		return &model.NetworkUtilization{}, nil
	}
	return s.networkUtilization, nil
}

func (s *serviceTestStorage) ListDiscoveryScans(_ context.Context, networkID string) ([]model.DiscoveryScan, error) {
	var scans []model.DiscoveryScan
	for _, scan := range s.discoveryScans {
		if networkID != "" && scan.NetworkID != networkID {
			continue
		}
		scans = append(scans, *scan)
	}
	return scans, nil
}

func (s *serviceTestStorage) GetDiscoveryScan(_ context.Context, id string) (*model.DiscoveryScan, error) {
	scan, ok := s.discoveryScans[id]
	if !ok {
		return nil, storage.ErrScanNotFound
	}
	cloned := *scan
	return &cloned, nil
}

func (s *serviceTestStorage) DeleteDiscoveryScan(_ context.Context, id string) error {
	if _, ok := s.discoveryScans[id]; !ok {
		return storage.ErrScanNotFound
	}
	delete(s.discoveryScans, id)
	return nil
}

func (s *serviceTestStorage) ListDiscoveredDevices(_ context.Context, networkID string) ([]model.DiscoveredDevice, error) {
	return append([]model.DiscoveredDevice(nil), s.discoveredByNetwork[networkID]...), nil
}

func (s *serviceTestStorage) GetDiscoveredDevice(_ context.Context, id string) (*model.DiscoveredDevice, error) {
	for _, devices := range s.discoveredByNetwork {
		for i := range devices {
			if devices[i].ID == id {
				cloned := devices[i]
				return &cloned, nil
			}
		}
	}
	return nil, storage.ErrDiscoveryNotFound
}

func (s *serviceTestStorage) DeleteDiscoveredDevicesByNetwork(_ context.Context, networkID string) error {
	delete(s.discoveredByNetwork, networkID)
	return nil
}

func (s *serviceTestStorage) DeleteDiscoveredDevice(_ context.Context, id string) error {
	for networkID, devices := range s.discoveredByNetwork {
		for i := range devices {
			if devices[i].ID == id {
				s.discoveredByNetwork[networkID] = append(devices[:i], devices[i+1:]...)
				return nil
			}
		}
	}
	return storage.ErrDiscoveryNotFound
}

func (s *serviceTestStorage) ListDiscoveryRules(_ context.Context) ([]model.DiscoveryRule, error) {
	var rules []model.DiscoveryRule
	for _, rule := range s.rules {
		rules = append(rules, *rule)
	}
	return rules, nil
}

func (s *serviceTestStorage) GetDiscoveryRule(_ context.Context, id string) (*model.DiscoveryRule, error) {
	rule, ok := s.rules[id]
	if !ok {
		return nil, storage.ErrRuleNotFound
	}
	cloned := *rule
	return &cloned, nil
}

func (s *serviceTestStorage) DeleteDevice(_ context.Context, id string) error {
	if s.deleteDeviceErr != nil {
		return s.deleteDeviceErr
	}
	if _, ok := s.devices[id]; !ok {
		return storage.ErrDeviceNotFound
	}
	delete(s.devices, id)
	return nil
}

func (s *serviceTestStorage) SaveDiscoveryRule(_ context.Context, rule *model.DiscoveryRule) error {
	cloned := *rule
	if cloned.ID == "" {
		cloned.ID = "rule-created"
	}
	s.rules[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) DeleteDiscoveryRule(_ context.Context, id string) error {
	if _, ok := s.rules[id]; !ok {
		return storage.ErrRuleNotFound
	}
	delete(s.rules, id)
	return nil
}

func (s *serviceTestStorage) ListAPIKeys(_ context.Context, filter *model.APIKeyFilter) ([]model.APIKey, error) {
	if filter != nil {
		cloned := *filter
		s.apiKeyFilter = &cloned
	}
	var results []model.APIKey
	for _, key := range s.apiKeys {
		if filter != nil && filter.UserID != "" && key.UserID != filter.UserID {
			continue
		}
		results = append(results, *key)
	}
	return results, nil
}

func (s *serviceTestStorage) CreateAPIKey(_ context.Context, key *model.APIKey) error {
	cloned := *key
	s.apiKeys[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) GetAPIKey(_ context.Context, id string) (*model.APIKey, error) {
	key, ok := s.apiKeys[id]
	if !ok {
		return nil, storage.ErrAPIKeyNotFound
	}
	cloned := *key
	return &cloned, nil
}

func (s *serviceTestStorage) DeleteAPIKey(_ context.Context, id string) error {
	s.deletedAPIKeyID = id
	if _, ok := s.apiKeys[id]; !ok {
		return storage.ErrAPIKeyNotFound
	}
	delete(s.apiKeys, id)
	return nil
}

func (s *serviceTestStorage) GetUserPermissions(_ context.Context, _ string) ([]model.Permission, error) {
	return nil, nil
}

func (s *serviceTestStorage) GetConflict(_ context.Context, id string) (*model.Conflict, error) {
	conflict, ok := s.conflicts[id]
	if !ok {
		return nil, storage.ErrConflictNotFound
	}
	cloned := *conflict
	return &cloned, nil
}

func (s *serviceTestStorage) UpdateConflictStatus(_ context.Context, id string, status model.ConflictStatus, resolvedBy, notes string) error {
	s.conflictStatus.id = id
	s.conflictStatus.status = status
	s.conflictStatus.resolvedBy = resolvedBy
	s.conflictStatus.notes = notes
	return nil
}

func (s *serviceTestStorage) CreateCircuit(_ context.Context, circuit *model.Circuit) error {
	cloned := *circuit
	if cloned.ID == "" {
		cloned.ID = "circuit-created"
	}
	s.circuitCreated = &cloned
	s.circuits[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) GetCircuit(_ context.Context, id string) (*model.Circuit, error) {
	circuit, ok := s.circuits[id]
	if !ok {
		return nil, storage.ErrCircuitNotFound
	}
	cloned := *circuit
	return &cloned, nil
}

func (s *serviceTestStorage) UpdateCircuit(_ context.Context, circuit *model.Circuit) error {
	cloned := *circuit
	s.circuitUpdated = &cloned
	s.circuits[cloned.ID] = &cloned
	return nil
}

func (s *serviceTestStorage) GetDashboardStats(_ context.Context, staleDays, recentLimit int) (*model.DashboardStats, error) {
	s.dashboardStaleDays = staleDays
	s.dashboardRecentLimit = recentLimit
	return &model.DashboardStats{}, nil
}

func (s *serviceTestStorage) GetUtilizationTrend(_ context.Context, _ model.SnapshotType, _ string, days int) ([]model.UtilizationTrendPoint, error) {
	s.utilTrendDays = days
	return []model.UtilizationTrendPoint{}, nil
}

func (s *serviceTestStorage) BulkCreateDevices(_ context.Context, _ []*model.Device) (*storage.BulkResult, error) {
	s.lastBulkOp = "create-devices"
	return s.bulkResult, nil
}

func (s *serviceTestStorage) BulkUpdateDevices(_ context.Context, _ []*model.Device) (*storage.BulkResult, error) {
	s.lastBulkOp = "update-devices"
	return s.bulkResult, nil
}

func (s *serviceTestStorage) BulkDeleteDevices(_ context.Context, _ []string) (*storage.BulkResult, error) {
	s.lastBulkOp = "delete-devices"
	return s.bulkResult, nil
}

func (s *serviceTestStorage) BulkAddTags(_ context.Context, _ []string, _ []string) (*storage.BulkResult, error) {
	s.lastBulkOp = "add-tags"
	return s.bulkResult, nil
}

func (s *serviceTestStorage) BulkRemoveTags(_ context.Context, _ []string, _ []string) (*storage.BulkResult, error) {
	s.lastBulkOp = "remove-tags"
	return s.bulkResult, nil
}

func (s *serviceTestStorage) BulkCreateNetworks(_ context.Context, _ []*model.Network) (*storage.BulkResult, error) {
	s.lastBulkOp = "create-networks"
	return s.bulkResult, nil
}

func (s *serviceTestStorage) BulkDeleteNetworks(_ context.Context, _ []string) (*storage.BulkResult, error) {
	s.lastBulkOp = "delete-networks"
	return s.bulkResult, nil
}

func (s *serviceTestStorage) ListAuditLogs(_ context.Context, _ *model.AuditFilter) ([]model.AuditLog, error) {
	return append([]model.AuditLog(nil), s.auditLogs...), nil
}

func (s *serviceTestStorage) GetAuditLog(_ context.Context, id string) (*model.AuditLog, error) {
	for _, log := range s.auditLogs {
		if log.ID == id {
			cloned := log
			return &cloned, nil
		}
	}
	return nil, storage.ErrAuditLogNotFound
}

func (s *serviceTestStorage) AddRelationship(_ context.Context, parentID, childID, relationshipType, notes string) error {
	s.addedParentID = parentID
	s.addedChildID = childID
	s.addedType = relationshipType
	s.addedNotes = notes
	return nil
}

func (s *serviceTestStorage) RemoveRelationship(_ context.Context, parentID, childID, relationshipType string) error {
	s.removedParentID = parentID
	s.removedChildID = childID
	s.removedType = relationshipType
	return s.removeErr
}

type stubSessionInvalidator struct {
	invalidated []string
}

func (s *stubSessionInvalidator) InvalidateUserSessions(userID string) {
	s.invalidated = append(s.invalidated, userID)
}

func userContext(userID string) context.Context {
	return WithCaller(context.Background(), &Caller{Type: CallerTypeUser, UserID: userID})
}
