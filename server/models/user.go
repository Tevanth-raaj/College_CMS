package models

import "time"

type User struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Password  string     `json:"-"` // Never send password to frontend
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type LoginResponse struct {
	Success     bool    `json:"success"`
	Message     string  `json:"message"`
	User        *User   `json:"user,omitempty"`
	Token       string  `json:"token,omitempty"`
	TeacherID   *string `json:"teacher_id,omitempty"`
	TeacherName *string `json:"teacher_name,omitempty"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type UpdateUserRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type ChangePasswordRequest struct {
	Password string `json:"password"`
}
