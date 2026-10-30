package dto

import "time"

type LoginRequest struct {
	Email    string `json:"email" example:"ada@coloplus.example"`
	Password string `json:"password" example:"correct-horse-battery-9"`
	// TOTPCode is the six-digit authenticator code, or a recovery code, needed
	// when two-step verification is on.
	TOTPCode string `json:"totp_code,omitempty" example:"123456"`
}

type LoginResponse struct {
	// Status is "ok" or "mfa_required". With mfa_required no token is issued;
	// send the same request again with totp_code.
	Status string `json:"status" example:"ok"`
	// MFASetupRequired is true when the person is signed in but their role
	// needs two-step verification they have not set up: everything else is
	// refused until they have.
	MFASetupRequired bool       `json:"mfa_setup_required"`
	AccessToken      string     `json:"access_token,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	User             *UserDTO   `json:"user,omitempty"`
}

type MeResponse struct {
	User        UserDTO  `json:"user"`
	Permissions []string `json:"permissions"`
	AllSites    bool     `json:"all_sites"`
	SiteIDs     []string `json:"site_ids"`
	// MFASetupRequired: the role needs two-step verification that is not on
	// yet, so permissions are empty until it is.
	MFASetupRequired bool `json:"mfa_setup_required"`
	// RecoveryCodesLeft is how many unused recovery codes remain.
	RecoveryCodesLeft int `json:"recovery_codes_left"`
}

type MFASetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

type EnableMFARequest struct {
	Code string `json:"code" example:"123456"`
}

// RecoveryCodesResponse carries the codes once. They cannot be shown again.
type RecoveryCodesResponse struct {
	RecoveryCodes []string `json:"recovery_codes" example:"K7QM-4XWP-92TD"`
}

type UserDTO struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FullName   string `json:"full_name"`
	Role       string `json:"role" example:"reporter"`
	Active     bool   `json:"is_active"`
	MFAEnabled bool   `json:"mfa_enabled"`
	// Invited is true until the person has chosen a password.
	Invited     bool       `json:"invited"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type UserDetailDTO struct {
	UserDTO
	SiteIDs []string `json:"site_ids"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role" example:"reporter"`
	// Password is optional. Leave it out and the person is emailed a link to
	// choose their own.
	Password string   `json:"password,omitempty"`
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

type ForgotPasswordRequest struct {
	Email string `json:"email" example:"ada@coloplus.example"`
}

type SetPasswordRequest struct {
	// Token is the value of the link's token parameter (invitation or reset).
	Token    string `json:"token"`
	Password string `json:"password"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type NotificationDTO struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind" example:"incident.overdue"`
	Recipient string     `json:"recipient"`
	Subject   string     `json:"subject"`
	Status    string     `json:"status" example:"failed"`
	Attempts  int        `json:"attempts"`
	LastError string     `json:"last_error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
}

type NotificationListResponse struct {
	Notifications []NotificationDTO `json:"notifications"`
	Total         int               `json:"total"`
	Limit         int               `json:"limit"`
	Offset        int               `json:"offset"`
}
