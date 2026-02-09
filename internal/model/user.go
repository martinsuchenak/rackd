package model

import "time"

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email,omitempty"`
	FullName     string     `json:"full_name,omitempty"`
	PasswordHash string     `json:"-" db:"password_hash"`
	IsActive     bool       `json:"is_active"`
	IsAdmin      bool       `json:"is_admin"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

type UserFilter struct {
	Username string
	Email    string
	IsActive *bool
	IsAdmin  *bool
}

type UserResponse struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email,omitempty"`
	FullName    string     `json:"full_name,omitempty"`
	IsActive    bool       `json:"is_active"`
	IsAdmin     bool       `json:"is_admin"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	Roles       []Role     `json:"roles,omitempty"`
}

type CurrentUserResponse struct {
	ID          string       `json:"id"`
	Username    string       `json:"username"`
	Email       string       `json:"email,omitempty"`
	FullName    string       `json:"full_name,omitempty"`
	IsActive    bool         `json:"is_active"`
	IsAdmin     bool         `json:"is_admin"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	LastLoginAt *time.Time   `json:"last_login_at,omitempty"`
	Permissions []Permission `json:"permissions"`
	Roles       []Role       `json:"roles"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User      UserResponse `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	FullName string `json:"full_name,omitempty"`
	IsAdmin  bool   `json:"is_admin,omitempty"`
	RoleID   string `json:"role_id,omitempty"`
}

type UpdateUserRequest struct {
	Email    string `json:"email,omitempty"`
	FullName string `json:"full_name,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
	IsAdmin  *bool  `json:"is_admin,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		FullName:    u.FullName,
		IsActive:    u.IsActive,
		IsAdmin:     u.IsAdmin,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}
