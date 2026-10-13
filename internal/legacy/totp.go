package legacy

import (
	"context"
	"database/sql"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

// UnwrapSecret returns a usable TOTP secret. Laravel-era rows store
// encrypt($secret); with the Laravel APP_KEY those blobs are decrypted.
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

// UpgradeLegacyTOTPSecrets makes the two-factor secrets of a Laravel database
// usable. A Laravel-encrypted secret is decrypted with appKey and written back
// in plain form; one that cannot be decrypted (no key, or the wrong one) is
// reset: two-factor is switched off for that user, so they can still sign in
// with their password. It returns how many secrets were decrypted and reset.
func UpgradeLegacyTOTPSecrets(ctx context.Context, sqlDB *sql.DB, appKey string) (unwrapped, reset int, err error) {
	if sqlDB == nil {
		return 0, 0, nil
	}
	q := db.Q(sqlDB)
	found, err := q.ListUsersWithTwoFactorSecret(ctx)
	if err != nil {
		return 0, 0, err
	}
	now := db.NS(time.Now().UTC().Format(time.RFC3339))
	for _, r := range found {
		if !LooksLikeLaravelEncrypted(r.TwoFactorSecret.String) {
			continue
		}
		if plain, ok := UnwrapSecret(appKey, r.TwoFactorSecret.String); ok {
			err = q.UpdateUserTwoFactorSecret(ctx, sqlc.UpdateUserTwoFactorSecretParams{TwoFactorSecret: db.NS(plain), UpdatedAt: now, ID: r.ID})
			unwrapped++
		} else {
			err = q.SetUserTwoFactor(ctx, sqlc.SetUserTwoFactorParams{UpdatedAt: now, ID: r.ID})
			if err == nil {
				err = q.DeleteRecoveryCodes(ctx, r.ID)
			}
			reset++
		}
		if err != nil {
			return unwrapped, reset, err
		}
	}
	return unwrapped, reset, nil
}
