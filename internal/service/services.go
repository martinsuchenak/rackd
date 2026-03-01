package service

import (
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Services struct {
	Devices        *DeviceService
	Datacenters    *DatacenterService
	Networks       *NetworkService
	Pools          *PoolService
	Relationships  *RelationshipService
	Discovery      *DiscoveryService
	Users          *UserService
	Roles          *RoleService
	Auth           *AuthService
	Audit          *AuditService
	APIKeys        *APIKeyService
	Bulk           *BulkService
	Credentials    *CredentialService
	ScanProfiles   *ScanProfileService
	ScheduledScans *ScheduledScanService
	OAuth          *OAuthService
	Conflicts      *ConflictService
	Reservations   *ReservationService
	Dashboard      *DashboardService
	Webhooks       *WebhookService
	CustomFields   *CustomFieldService
	Circuits       *CircuitService
	NAT            *NATService
	DNS            *DNSService
}

func NewServices(store storage.ExtendedStorage, sessionManager *auth.SessionManager, scanner discovery.Scanner) *Services {
	return &Services{
		Devices:       NewDeviceService(store),
		Datacenters:   NewDatacenterService(store),
		Networks:      NewNetworkService(store),
		Pools:         NewPoolService(store),
		Relationships: NewRelationshipService(store),
		Discovery:     NewDiscoveryService(store, scanner),
		Users:         NewUserService(store, sessionManager),
		Roles:         NewRoleService(store),
		Auth:          NewAuthService(store, sessionManager),
		Audit:         NewAuditService(store),
		APIKeys:       NewAPIKeyService(store),
		Bulk:          NewBulkService(store),
		Conflicts:     NewConflictService(store),
		Reservations:  NewReservationService(store),
		Dashboard:     NewDashboardService(store),
		Webhooks:      NewWebhookService(store),
		CustomFields:  NewCustomFieldService(store),
		Circuits:      NewCircuitService(store),
		NAT:           NewNATService(store),
	}
}

func (s *Services) SetCredentialsStorage(store credentials.Storage) {
	s.Credentials = NewCredentialService(store, s.Users.store)
}

func (s *Services) SetProfileStorage(store storage.ProfileStorage) {
	s.ScanProfiles = NewScanProfileService(store, s.Users.store)
}

func (s *Services) SetScheduledScanStorage(store storage.ScheduledScanStorage) {
	s.ScheduledScans = NewScheduledScanService(store, s.Users.store)
}

func (s *Services) SetDNSService(store storage.ExtendedStorage, encryptor *credentials.Encryptor) {
	s.DNS = NewDNSService(store, encryptor)
	// Set DNS service on DeviceService for automatic DNS record creation/updates
	s.Devices.setDNSService(s.DNS)
}
