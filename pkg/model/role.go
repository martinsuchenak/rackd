package model

import "time"

type Permission struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RolePermission struct {
	RoleID       string    `json:"role_id"`
	PermissionID string    `json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserRole struct {
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type RoleFilter struct {
	Pagination
	Name     string
	IsSystem *bool
}

type PermissionFilter struct {
	Pagination
	Resource string
	Action   string
}

type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type UpdateRoleRequest struct {
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type AssignRoleRequest struct {
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
}

type RoleWithPermissions struct {
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions,omitempty"`
}

type UserWithRoles struct {
	User        UserResponse `json:"user"`
	Roles       []Role       `json:"roles,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

type RoleResponse struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	IsSystem    bool         `json:"is_system"`
	Permissions []Permission `json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (r *Role) ToResponse() RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func (r *Role) ToResponseWithPermissions(permissions []Permission) RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		Permissions: permissions,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

type GrantRoleRequest struct {
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
}

type RevokeRoleRequest struct {
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
}
