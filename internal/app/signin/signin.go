// Package signin holds the instance policy on how people may sign in.
package signin

import (
	"context"

	"savvy-go/internal/settings"
	"savvy-go/internal/sso"
)

// SettingRequireVerification is the instance setting that holds back users
// whose email address is not confirmed.
const SettingRequireVerification = "require_email_verification"

// Policy answers what the instance settings allow.
type Policy struct {
	Settings settings.Store
	SSO      *sso.Service
}

// PasswordLoginDisabled is true when an admin turned password sign-in off and
// at least one SSO provider is there to replace it.
func (p Policy) PasswordLoginDisabled(ctx context.Context) bool {
	if p.Settings.Bool(ctx, "password_login_enabled", true) {
		return false
	}
	return p.SSO.AnyEnabled(ctx)
}

// EmailRequired is true while the instance holds back unverified users.
func (p Policy) EmailRequired(ctx context.Context) bool {
	return p.Settings.Bool(ctx, SettingRequireVerification, false)
}
