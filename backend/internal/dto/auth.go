package dto

import "time"

type LoginRequest struct {
	Email    string `json:"email" example:"ada@coloplus.example"`
	Password string `json:"password" example:"correct-horse-battery-9"`
	// TOTPCode is the six-digit authenticator code, needed when two-step
	// verification is on.
	TOTPCode string `json:"totp_code,omitempty" example:"123456"`
}

type LoginResponse struct {
	// Status is "ok" or "mfa_required". With mfa_required no token is issued;
	// send the same request again with totp_code.
	Status      string     `json:"status" example:"ok"`
	AccessToken string     `json:"access_token,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	User        *UserDTO   `json:"user,omitempty"`
}

type MeResponse struct {
	User        UserDTO  `json:"user"`
	Permissions []string `json:"permissions"`
	AllSites    bool     `json:"all_sites"`
	SiteIDs     []string `json:"site_ids"`
}

type MFASetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

type EnableMFARequest struct {
	Code string `json:"code" example:"123456"`
}

type UserDTO struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Role        string     `json:"role" example:"reporter"`
	Active      bool       `json:"is_active"`
	MFAEnabled  bool       `json:"mfa_enabled"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type UserDetailDTO struct {
	UserDTO
	SiteIDs []string `json:"site_ids"`
}

type CreateUserRequest struct {
	Email    string   `json:"email"`
	FullName string   `json:"full_name"`
	Role     string   `json:"role" example:"reporter"`
	Password string   `json:"password"`
	SiteIDs  []string `json:"site_ids"`
}

// UpdateUserRequest changes only the fields that are present.
type UpdateUserRequest struct {
	FullName *string   `json:"full_name,omitempty"`
	Role     *string   `json:"role,omitempty"`
	Active   *bool     `json:"is_active,omitempty"`
	Password *string   `json:"password,omitempty"`
	SiteIDs  *[]string `json:"site_ids,omitempty"`
}

type UserListResponse struct {
	Users []UserDTO `json:"users"`
}
