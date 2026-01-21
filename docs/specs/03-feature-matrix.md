# Feature Classification Matrix

This document outlines which features are available in the OSS edition versus the Enterprise/Enterprise edition.

## Feature Matrix

| Feature | OSS | Enterprise | Implementation Notes |
|---------|-----|---------|----------------------|
| **Core Features** ||||
| Device CRUD | ✅ | | Core tracking functionality |
| Datacenter CRUD | ✅ | | Physical location tracking |
| Network/IPAM | ✅ | | Subnet, pool, IP allocation |
| Network Pools | ✅ | | IP allocation within networks |
| Device Relationships | ✅ | | Dependency tracking (contains, connected_to, depends_on) |
| CLI Tool | ✅ | | Full command-line interface |
| Web UI | ✅ | | Complete web interface |
| MCP Server | ✅ | | All AI/automation tools |
| Search | ✅ | | Full-text search across entities |
| SQLite Storage | ✅ | | Default embedded database |
| **Discovery** ||||
| Basic IP Discovery | ✅ | | Ping scan, ARP table import |
| Discovery Scheduling | ✅ | | Configurable scan intervals |
| Discovered Device Promotion | ✅ | | Promote to full inventory |
| Advanced Discovery (SNMP) | | ✅ | SNMP polling, service detection |
| Continuous Discovery | | ✅ | Real-time network monitoring |
| **Visualization** ||||
| Visual Subnet Utilization | ✅ | | Heatmaps for IP usage |
| Network Topology | | ✅ | Visual network maps |
| **Network Features** ||||
| VLAN Management | ✅ | | Basic VLAN tracking |
| VRF Support | ✅ | | VRF-lite (network-scoped IPs) |
| DNS Integration | | ✅ | DNS server integration |
| DHCP Integration | | ✅ | DHCP server integration |
| **Enterprise** ||||
| Postgres Storage | | ✅ | Enterprise database backend |
| User Management | | ✅ | Multi-user support |
| SSO/OIDC | | ✅ | Single sign-on integration |
| RBAC | | ✅ | Role-based access control |
| Audit Logging | | ✅ | Compliance audit trail |
| Circuit Management | | ✅ | Provider circuit tracking |
| NAT Tracking | | ✅ | NAT mapping and tracking |
| Advanced Monitoring | | ✅ | Metrics, dashboards, alerts |

## Feature Interface Definitions (OSS)

These interfaces are defined in the OSS repository and implemented by Enterprise features:

```go
// ===== OSS REPO: internal/types/enterprise.go =====
package types

import (
    "context"
    "net/http"
    "time"
)

// AuthProvider defines authentication and authorization interfaces
type AuthProvider interface {
    AuthenticateRequest(r *http.Request) (context.Context, error)
    LoginURL(redirectURL string) string
    LogoutURL(redirectURL string) string
}

// User represents an authenticated user (for Enterprise RBAC)
type User struct {
    ID       string
    Username string
    Email    string
    Roles    []string
}

// RBACChecker defines role-based access control interface
type RBACChecker interface {
    CheckPermission(ctx context.Context, resource, action string) bool
    GetUserRoles(ctx context.Context, userID string) ([]string, error)
}

// AuditLogger defines audit logging interface
type AuditLogger interface {
    LogAction(ctx context.Context, entry AuditEntry) error
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
    UserID      string
    Action      string
    Resource    string
    ResourceID  string
    Details     map[string]interface{}
    IPAddress   string
    Timestamp   time.Time
}

// MonitoringBackend defines metrics and monitoring interface
type MonitoringBackend interface {
    RecordMetric(name string, value float64, tags map[string]string)
    RecordEvent(event string, details map[string]interface{})
    RecordHTTPRequest(method, path string, status int, duration time.Duration)
}

// AdvancedDiscoveryService defines extended discovery capabilities
type AdvancedDiscoveryService interface {
    ScanSubnetSNMP(ctx context.Context, subnet string, community string) ([]DiscoveredDevice, error)
    ScheduleScan(ctx context.Context, schedule ScanSchedule) error
    GetScanHistory(ctx context.Context, networkID string) ([]ScanResult, error)
}

// DNSProvider defines DNS integration interface
type DNSProvider interface {
    CreateARecord(ctx context.Context, name, ip string) error
    CreatePTRRecord(ctx context.Context, ip, name string) error
    DeleteRecord(ctx context.Context, name string) error
    ListRecords(ctx context.Context) ([]DNSRecord, error)
}

type DNSRecord struct {
    Name  string
    Type  string
    Value string
    TTL   int
}

// DHCPManager defines DHCP integration interface
type DHCPManager interface {
    CreateLease(ctx context.Context, lease DHCPLease) error
    GetLease(ctx context.Context, ip string) (*DHCPLease, error)
    ListLeases(ctx context.Context) ([]DHCPLease, error)
    DeleteLease(ctx context.Context, ip string) error
}

type DHCPLease struct {
    IP       string
    MAC      string
    Hostname string
    StartsAt time.Time
    EndsAt   time.Time
}

// Circuit represents a provider circuit (for Enterprise)
type Circuit struct {
    ID              string
    Provider        string
    CircuitID       string
    Type            string
    Capacity        int
    AEndpoint       string
    ZEndpoint       string
    Status          string
    InstallDate     time.Time
    TerminationDate *time.Time
}

// NATMapping represents a NAT translation (for Enterprise)
type NATMapping struct {
    ID           string
    ExternalIP   string
    ExternalPort int
    InternalIP   string
    InternalPort int
    Protocol     string
    DeviceID     string
}
```
