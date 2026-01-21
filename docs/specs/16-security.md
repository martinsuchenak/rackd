# Security Specification

This document outlines the security considerations, policies, and practices for Rackd. It aims to ensure the confidentiality, integrity, and availability of the system and the data it manages.

## 1. Threat Model

A foundational threat model identifies key assets, potential threats, vulnerabilities, and counter-measures.

**Key Assets:**
- IPAM data (IP addresses, network configurations, device details)
- User credentials (Enterprise edition)
- Application configuration (API tokens, database credentials)
- System availability

**Threats & Vulnerabilities:**
- Unauthorized access to data or system functions.
- Data tampering or corruption.
- Denial of Service (DoS).
- Code injection (SQLi, XSS, Command Injection).
- Exposure of sensitive information (secrets, PII).

## 2. Authentication and Authorization

### 2.1. API Authentication (OSS)

- **Mechanism**: Bearer token (`API_AUTH_TOKEN`).
- **Policy**: If `API_AUTH_TOKEN` is set, all API endpoints require a valid token. If not set, API is open (suitable for internal networks).
- **Token Management**: Tokens should be strong, secret, and managed securely (e.g., environment variables, secret management systems). Rotation mechanisms are recommended.

### 2.2. MCP Authentication (OSS)

- **Mechanism**: Bearer token (`MCP_AUTH_TOKEN`).
- **Policy**: Similar to API authentication, the MCP endpoint requires a valid token if `MCP_AUTH_TOKEN` is set.
- **Purpose**: Controls access for AI/automation tools that interact with the MCP server.

### 2.3. Enterprise Authentication (SSO/OIDC)

- **Mechanism**: SSO/OIDC integration as defined in `03-feature-matrix.md`.
- **Policy**: Delegates authentication to an external identity provider.
- **User Information**: User roles and permissions are retrieved from the identity provider or mapped within Rackd.

### 2.4. Role-Based Access Control (RBAC - Enterprise)

- **Mechanism**: `RBACChecker` interface as defined in `03-feature-matrix.md`.
- **Policy**: Granular control over resource access and actions based on user roles.
- **Implementation**: Authorization checks must be performed at the API handler level and potentially within business logic.

## 3. Data Protection

### 3.1. Data at Rest

- **OSS (SQLite)**: Data is stored in a local file. Users are responsible for filesystem-level security (permissions, encryption-at-rest solutions like LUKS, BitLocker).
- **Enterprise (PostgreSQL)**: Database server should be configured with encryption-at-rest. Network isolation for the database is highly recommended.

### 3.2. Data in Transit

- **HTTPS**: All external-facing HTTP API and Web UI traffic **MUST** be served over HTTPS. `SecurityHeaders` middleware includes HSTS.
- **Internal Communication**: Inter-service communication (if any) should also use TLS.

## 4. Input Validation and Output Encoding

- **Input Validation**: All user-supplied input (API parameters, CLI arguments, UI forms) **MUST** be rigorously validated against expected types, formats, and lengths. This prevents common injection vulnerabilities.
- **Output Encoding**: Data rendered in the Web UI **MUST** be properly HTML-encoded to prevent Cross-Site Scripting (XSS) attacks. Database query parameters **MUST** use parameterized queries or ORM methods to prevent SQL Injection. System command arguments **MUST** be properly escaped to prevent Command Injection.

## 5. Secret Management

- **Configuration**: Sensitive configuration values (e.g., API tokens, database connection strings) should be supplied via environment variables or a secure configuration management system (e.g., HashiCorp Vault, Kubernetes Secrets). They **MUST NOT** be hardcoded in source control.
- **Logging**: Secrets **MUST NOT** be logged or displayed in error messages. Redaction mechanisms should be in place for sensitive data in logs.

## 6. Dependency Security

- **Vulnerability Scanning**: Regularly scan project dependencies for known vulnerabilities (e.g., using `go list -m all | grep -v replace | xargs -L1 go vuln`).
- **Updates**: Keep dependencies up-to-date to benefit from security patches.
- **Review**: Carefully review new dependencies before adoption.

## 7. Logging and Monitoring

- **Security Logging**: Implement comprehensive logging of security-relevant events, such as authentication attempts (success/failure), authorization failures, and critical configuration changes.
- **Audit Logging (Enterprise)**: The `AuditLogger` interface (see `03-feature-matrix.md`) provides a mechanism for detailed, immutable audit trails.

## 8. Deployment Security

- **Principle of Least Privilege**: Deploy the application with the minimum necessary permissions.
- **Network Segmentation**: Isolate the application and database servers within the network.
- **Container Security**: Follow Docker best practices (e.g., non-root user, minimal base image, regularly updated images).

## 9. Error Handling

- **Fail Securely**: Error messages returned to users **MUST NOT** expose sensitive system information (e.g., stack traces, internal paths, database query details). Generic error messages should be provided to the user, while detailed errors are logged internally.

## 10. Code Review

- All security-sensitive code paths (authentication, authorization, data handling) **MUST** undergo rigorous peer code review.
