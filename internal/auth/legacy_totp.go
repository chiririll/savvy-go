package auth

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// UnwrapSecret returns a usable TOTP secret. Laravel-era rows store
// encrypt($secret); when APP_KEY is set those blobs are decrypted.
func UnwrapSecret(appKey, stored string) (string, bool) {
	if stored == "" {
		return "", false
	}
	if !LooksLikeLaravelEncrypted(stored) {
		return stored, false
	}
	if appKey == "" {
		return stored, false
	}
	plain, err := DecryptLaravel(appKey, stored)
	if err != nil || plain == "" {
		return stored, false
	}
	return plain, true
}

func (t TwoFactor) resolvedSecret(ctx context.Context, u *User) string {
	if u == nil || u.TwoFactorSecret == nil {
		return ""
	}
	plain, changed := UnwrapSecret(t.AppKey, *u.TwoFactorSecret)
	if changed {
		sec := plain
		u.TwoFactorSecret = &sec
		if t.Users.DB != nil {
			_ = t.Users.SetTwoFactor(ctx, u.ID, &sec, u.TwoFactorEnabled, u.TwoFactorConfirmed)
		}
	}
	return plain
}

func (t TwoFactor) verifyTOTP(ctx context.Context, u *User, code string) bool {
	return VerifyTOTP(t.resolvedSecret(ctx, u), code)
}

// UnwrapLegacyTOTPSecrets decrypts every Laravel-encrypted two_factor_secret
// and writes the plaintext secret back. Safe to run on every start.
func UnwrapLegacyTOTPSecrets(ctx context.Context, db *sql.DB, appKey string) error {
	if appKey == "" || db == nil {
		return nil
	}
	rows, err := db.QueryContext(ctx, `SELECT id, two_factor_secret FROM users WHERE two_factor_secret IS NOT NULL AND two_factor_secret != ''`)
	if err != nil {
		return err
	}
	type row struct {
		id     int64
		secret string
	}
	var found []row
	for rows.Next() {
		var r row
		if rows.Scan(&r.id, &r.secret) == nil {
			found = append(found, r)
		}
	}
	rows.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	n := 0
	for _, r := range found {
		plain, ok := UnwrapSecret(appKey, r.secret)
		if !ok {
			continue
		}
		if _, err := db.ExecContext(ctx, `UPDATE users SET two_factor_secret=?, updated_at=? WHERE id=?`, plain, now, r.id); err != nil {
			return err
		}
		n++
	}
	if n > 0 {
		slog.Info("unwrapped laravel two-factor secrets", "count", n)
	}
	return nil
}
