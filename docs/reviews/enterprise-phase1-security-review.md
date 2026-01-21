# Enterprise Phase 1 Security Review

**Date**: 2026-01-21  
**Reviewer**: Automated Security Review  
**Scope**: rackd-enterprise Enterprise Phase 1 Implementation  
**Status**: REVIEW COMPLETE

---

## Executive Summary

Enterprise Phase 1 implements repository setup and enterprise-specific data models. The implementation is **functionally complete** per specification but has **several security concerns** that should be addressed before production use.

| Category | Status | Risk Level |
|----------|--------|------------|
| Repository Structure | ✅ PASS | Low |
| Module Dependencies | ✅ PASS | Low |
| Credential Model | ⚠️ CONCERNS | High |
| Scan Profile Model | ✅ PASS | Low |
| Scheduled Scan Model | ✅ PASS | Low |
| .gitignore Configuration | ✅ PASS | Low |

---

## 1. Repository Structure Review

### 1.1 go.mod Analysis

**File**: `rackd-enterprise/go.mod`

```go
module github.com/martinsuchenak/rackd-enterprise

go 1.25.6

require (
	github.com/martinsuchenak/rackd v0.0.0
)

replace github.com/martinsuchenak/rackd => ../rackd
```

**Findings**:
- ✅ Module path correctly set to `github.com/martinsuchenak/rackd-enterprise`
- ✅ Imports OSS core as required by spec
- ✅ Local replace directive appropriate for development
- ⚠️ **Note**: Replace directive should be removed for production releases

**Compliance**: Meets spec requirements (E1-001)

### 1.2 Directory Structure

**Findings**:
- ✅ `internal/features/` - Present
- ✅ `internal/discovery/` - Present
- ✅ `internal/credentials/` - Present
- ✅ `cmd/rackd-enterprise/` - Present

**Compliance**: Meets spec requirements (E1-002)

### 1.3 .gitignore Review

**Findings**:
- ✅ Excludes `.env` files (prevents credential leakage)
- ✅ Excludes database files (`*.db`, `*.db-shm`, `*.db-wal`)
- ✅ Excludes IDE configurations
- ✅ Excludes binaries and build artifacts

**Compliance**: Follows security best practices

---

## 2. Enterprise Models Security Review

### 2.1 Credential Model (HIGH RISK)

**File**: `rackd-enterprise/internal/model/credential.go`

```go
type Credential struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"` // "snmp_v2c", "snmp_v3", "ssh_key", "ssh_password"
	SNMPCommunity string    `json:"snmp_community,omitempty"`
	SNMPV3User    string    `json:"snmp_v3_user,omitempty"`
	SNMPV3Auth    string    `json:"snmp_v3_auth,omitempty"`
	SNMPV3Priv    string    `json:"snmp_v3_priv,omitempty"`
	SSHUsername   string    `json:"ssh_username,omitempty"`
	SSHKeyID      string    `json:"ssh_key_id,omitempty"`
	DatacenterID  string    `json:"datacenter_id,omitempty"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
```

#### Security Concerns

| Issue | Severity | Description |
|-------|----------|-------------|
| SEC-001 | **HIGH** | `SNMPCommunity` stores plaintext SNMP community strings |
| SEC-002 | **HIGH** | `SNMPV3Auth` stores plaintext authentication passwords |
| SEC-003 | **HIGH** | `SNMPV3Priv` stores plaintext privacy passwords |
| SEC-004 | **MEDIUM** | No field-level encryption markers or handling |
| SEC-005 | **MEDIUM** | JSON serialization exposes secrets in API responses |
| SEC-006 | **LOW** | Missing validation constraints on Type field |

#### Recommendations

1. **Encrypt sensitive fields at rest**:
   ```go
   // Add encryption markers
   SNMPCommunity string `json:"-" db:"snmp_community_encrypted"`
   ```

2. **Implement separate DTO for API responses**:
   ```go
   type CredentialResponse struct {
       ID           string    `json:"id"`
       Name         string    `json:"name"`
       Type         string    `json:"type"`
       // Omit sensitive fields
       HasCommunity bool      `json:"has_community"`
       HasAuth      bool      `json:"has_auth"`
   }
   ```

3. **Add validation for Type field**:
   ```go
   var ValidCredentialTypes = []string{"snmp_v2c", "snmp_v3", "ssh_key", "ssh_password"}
   ```

4. **Consider using a secrets manager reference** instead of storing secrets directly:
   ```go
   SecretRef string `json:"secret_ref,omitempty"` // Reference to external secret store
   ```

### 2.2 Scan Profile Model (LOW RISK)

**File**: `rackd-enterprise/internal/model/scan_profile.go`

```go
type ScanProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ScanType    string    `json:"scan_type"` // "quick", "full", "deep", "custom"
	Ports       []int     `json:"ports,omitempty"`
	EnableSNMP  bool      `json:"enable_snmp"`
	EnableSSH   bool      `json:"enable_ssh"`
	TimeoutSec  int       `json:"timeout_sec"`
	MaxWorkers  int       `json:"max_workers"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

**Findings**:
- ✅ No sensitive data stored
- ✅ JSON tags match API expectations per spec
- ⚠️ **Minor**: Missing validation for `Ports` range (should be 1-65535)
- ⚠️ **Minor**: Missing validation for `MaxWorkers` (should have upper bound to prevent DoS)
- ⚠️ **Minor**: Missing validation for `ScanType` enum values

**Recommendations**:
1. Add port range validation (1-65535)
2. Add MaxWorkers upper bound (e.g., 100)
3. Add ScanType enum validation

### 2.3 Scheduled Scan Model (LOW RISK)

**File**: `rackd-enterprise/internal/model/scheduled_scan.go`

```go
type ScheduledScan struct {
	ID             string     `json:"id"`
	NetworkID      string     `json:"network_id"`
	ProfileID      string     `json:"profile_id"`
	Name           string     `json:"name"`
	CronExpression string     `json:"cron_expression"`
	Enabled        bool       `json:"enabled"`
	Description    string     `json:"description,omitempty"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
```

**Findings**:
- ✅ No sensitive data stored
- ✅ JSON tags match API expectations per spec
- ✅ Proper use of pointer types for optional timestamps
- ⚠️ **Minor**: `CronExpression` should be validated to prevent malicious schedules

**Recommendations**:
1. Validate cron expression syntax
2. Consider minimum interval enforcement to prevent resource exhaustion

---

## 3. Specification Compliance Matrix

| Requirement | Spec Reference | Status | Notes |
|-------------|----------------|--------|-------|
| go.mod imports OSS | E1-001 | ✅ PASS | |
| Module path correct | E1-001 | ✅ PASS | |
| Can import OSS types | E1-001 | ✅ PASS | |
| Directory structure | E1-002 | ✅ PASS | |
| Credential struct | E1-003 | ✅ PASS | Fields present, security concerns noted |
| ScanProfile struct | E1-003 | ✅ PASS | |
| ScheduledScan struct | E1-003 | ✅ PASS | |
| JSON tags match API | E1-003 | ✅ PASS | |

---

## 4. Security Specification Compliance

Per `docs/specs/16-security.md`:

| Requirement | Status | Notes |
|-------------|--------|-------|
| Secret Management (§5) | ⚠️ PARTIAL | Credential model stores secrets in plaintext |
| Input Validation (§4) | ⚠️ MISSING | No validation on model fields |
| Data at Rest (§3.1) | ⚠️ MISSING | No encryption for sensitive credential fields |
| Logging (§7) | N/A | Not applicable to data models |

---

## 5. Risk Summary

### High Priority (Address Before Production)

1. **SEC-001/002/003**: Credential secrets stored in plaintext
   - Impact: Credential exposure if database is compromised
   - Mitigation: Implement field-level encryption or external secret store

2. **SEC-005**: JSON serialization exposes secrets
   - Impact: API responses may leak credentials
   - Mitigation: Use separate DTOs for API responses

### Medium Priority (Address Before Beta)

3. **SEC-004**: No encryption markers
   - Impact: Developers may inadvertently expose secrets
   - Mitigation: Add `json:"-"` tags to sensitive fields

4. **Missing validation**: Type enums and numeric bounds
   - Impact: Invalid data may cause runtime errors
   - Mitigation: Add validation methods to models

### Low Priority (Address in Future Phases)

5. Cron expression validation
6. Port range validation
7. MaxWorkers bounds

---

## 6. Recommendations Summary

### Immediate Actions

1. Add `json:"-"` to sensitive Credential fields
2. Create CredentialResponse DTO for API serialization
3. Document encryption requirements for storage layer

### Phase 2 Actions

1. Implement field-level encryption in storage layer
2. Add validation methods to all models
3. Consider integration with external secret managers

---

## 7. Conclusion

Enterprise Phase 1 implementation is **functionally complete** and meets the structural requirements of the specification. However, the Credential model presents **significant security risks** due to plaintext storage of sensitive authentication data.

**Recommendation**: Address HIGH severity issues (SEC-001, SEC-002, SEC-003, SEC-005) before proceeding to production deployment. These can be deferred for development/testing phases but must be resolved before handling real credentials.

---

*Review generated: 2026-01-21*
