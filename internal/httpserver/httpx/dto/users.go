package dto

import (
	"time"

	"savvy-go/internal/auth"
)

// SessionUser is the signed-in user as the SPA keeps it.
type SessionUser struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	IsSSOOnly     bool   `json:"isSsoOnly"`
	EmailVerified bool   `json:"emailVerified"`
}

func NewSessionUser(u auth.User) SessionUser {
	return SessionUser{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, IsSSOOnly: u.IsSSOOnly, EmailVerified: u.EmailVerified}
}

// User is a user as the server admin lists them.
type User struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	Role          string     `json:"role"`
	IsInactive    bool       `json:"isInactive"`
	IsSSOOnly     bool       `json:"isSsoOnly"`
	EmailVerified bool       `json:"emailVerified"`
	CreatedAt     *time.Time `json:"createdAt"`
}

func NewUser(u auth.User) User {
	return User{
		ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, IsInactive: u.IsInactive(), IsSSOOnly: u.IsSSOOnly,
		EmailVerified: u.EmailVerified, CreatedAt: utc(u.CreatedAt),
	}
}

// SetupLink is the set-password link of a user without a password: handed back
// as Token, or mailed (EmailSent), or mailing failed (EmailError).
type SetupLink struct {
	ExpiresAt  time.Time `json:"expiresAt"`
	Token      string    `json:"token,omitempty"`
	EmailSent  bool      `json:"emailSent,omitempty"`
	EmailError string    `json:"emailError,omitempty"`
}

// UserWithLink is a user together with the link that lets them set a password.
type UserWithLink struct {
	User
	*SetupLink
}

// SessionPayload answers sign-in and "who am I". User, ExpiresAt and RefreshAt
// are null without a session; RefreshAt also when the session does not rotate.
type SessionPayload struct {
	User                      *SessionUser `json:"user"`
	ExpiresAt                 *time.Time   `json:"expires_at"`
	RefreshAt                 *time.Time   `json:"refresh_at"`
	EmailVerificationRequired bool         `json:"email_verification_required"`
}

// PasswordLink is what the set-password page shows before the user chooses one.
type PasswordLink struct {
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	IsInactive bool      `json:"isInactive"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// BootstrapHint says where to find the first-run password; the password itself
// never leaves the server.
type BootstrapHint struct {
	Environment string `json:"environment"`
	File        string `json:"file"`
}

// SetupStatus tells the sign-in page what to offer.
type SetupStatus struct {
	NeedsRegistration    bool           `json:"needs_registration"`
	PasswordLoginEnabled bool           `json:"password_login_enabled"`
	BootstrapHint        *BootstrapHint `json:"bootstrap_hint"`
}

// SignedOutOthers reports how many other sessions were ended.
type SignedOutOthers struct {
	Message string `json:"message"`
	Revoked int    `json:"revoked"`
}

// TwoFactorSetup starts enrolment: the secret and the QR code to scan.
type TwoFactorSetup struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
}

// RecoveryCodes are shown once, when two-factor is enabled or the codes are regenerated.
type RecoveryCodes struct {
	Message string   `json:"message"`
	Codes   []string `json:"recovery_codes"`
}

type RecoveryCodesRemaining struct {
	Remaining int `json:"remaining_count"`
}

type TwoFactorStatus struct {
	Enabled                bool `json:"enabled"`
	PendingConfirmation    bool `json:"pending_confirmation"`
	RecoveryCodesRemaining *int `json:"recovery_codes_remaining"`
}

type PasskeyList struct {
	Credentials []WebAuthnCred `json:"credentials"`
}

type PasskeyRegistered struct {
	Message    string       `json:"message"`
	Credential WebAuthnCred `json:"credential"`
}
