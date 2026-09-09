package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"strings"
	"time"
)

const recoveryChars = "abcdefghjkmnpqrstuvwxyz23456789"

type TwoFactor struct {
	DB     *sql.DB
	Users  Users
	AppKey string
}

func (t TwoFactor) Enable(ctx context.Context, u *User) (secret, uri string, err error) {
	if u.HasTwoFactor() {
		return "", "", fmtErr("already enabled")
	}
	secret, err = NewTOTPSecret()
	if err != nil {
		return "", "", err
	}
	if err := t.Users.SetTwoFactor(ctx, u.ID, &secret, true, false); err != nil {
		return "", "", err
	}
	return secret, TOTPProvisioningURI(u.Email, secret), nil
}

func (t TwoFactor) Confirm(ctx context.Context, u *User, code string) ([]string, error) {
	if !u.TwoFactorEnabled || u.TwoFactorConfirmed {
		return nil, fmtErr("not pending")
	}
	if u.TwoFactorSecret == nil || !VerifyTOTP(*u.TwoFactorSecret, code) {
		return nil, fmtErr("invalid code")
	}
	if err := t.Users.SetTwoFactor(ctx, u.ID, u.TwoFactorSecret, true, true); err != nil {
		return nil, err
	}
	return t.generateCodes(ctx, u.ID)
}

func (t TwoFactor) Disable(ctx context.Context, u *User, code string) error {
	if !u.HasTwoFactor() {
		return fmtErr("not enabled")
	}
	if !t.VerifyAny(ctx, u, code) {
		return fmtErr("invalid code")
	}
	if err := t.Users.SetTwoFactor(ctx, u.ID, nil, false, false); err != nil {
		return err
	}
	_, _ = t.DB.ExecContext(ctx, `DELETE FROM two_factor_recovery_codes WHERE user_id = ?`, u.ID)
	return nil
}

func (t TwoFactor) VerifyAny(ctx context.Context, u *User, code string) bool {
	if t.verifyTOTP(ctx, u, code) {
		return true
	}
	return t.ConsumeRecovery(ctx, u.ID, code)
}

func (t TwoFactor) ConsumeRecovery(ctx context.Context, userID int64, code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	rows, err := t.DB.QueryContext(ctx, `SELECT id, code FROM two_factor_recovery_codes WHERE user_id = ? AND used_at IS NULL`, userID)
	if err != nil {
		return false
	}
	defer rows.Close()
	type row struct {
		id   int64
		hash string
	}
	var found []row
	for rows.Next() {
		var r row
		if rows.Scan(&r.id, &r.hash) == nil {
			found = append(found, r)
		}
	}
	rows.Close()
	for _, r := range found {
		if CheckPassword(r.hash, code) {
			now := time.Now().UTC().Format(time.RFC3339)
			_, _ = t.DB.ExecContext(ctx, `UPDATE two_factor_recovery_codes SET used_at=? WHERE id=?`, now, r.id)
			return true
		}
	}
	return false
}

func (t TwoFactor) Remaining(ctx context.Context, userID int64) int {
	var n int
	_ = t.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM two_factor_recovery_codes WHERE user_id = ? AND used_at IS NULL`, userID).Scan(&n)
	return n
}

func (t TwoFactor) Regenerate(ctx context.Context, u *User, code string) ([]string, error) {
	if !u.HasTwoFactor() {
		return nil, fmtErr("not enabled")
	}
	if !t.verifyTOTP(ctx, u, code) {
		return nil, fmtErr("invalid code")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = t.DB.ExecContext(ctx, `UPDATE two_factor_recovery_codes SET used_at=? WHERE user_id=? AND used_at IS NULL`, now, u.ID)
	return t.generateCodes(ctx, u.ID)
}

func (t TwoFactor) generateCodes(ctx context.Context, userID int64) ([]string, error) {
	var codes []string
	now := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < 8; i++ {
		plain := recoveryCode()
		h, err := HashPassword(plain)
		if err != nil {
			return nil, err
		}
		if _, err := t.DB.ExecContext(ctx, `INSERT INTO two_factor_recovery_codes (user_id, code, created_at) VALUES (?,?,?)`, userID, h, now); err != nil {
			return nil, err
		}
		codes = append(codes, plain)
	}
	return codes, nil
}

func recoveryCode() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	var p1, p2 strings.Builder
	for i := 0; i < 4; i++ {
		p1.WriteByte(recoveryChars[int(b[i])%len(recoveryChars)])
		p2.WriteByte(recoveryChars[int(b[i+4])%len(recoveryChars)])
	}
	return p1.String() + "-" + p2.String()
}

type tfErr string

func (e tfErr) Error() string { return string(e) }

func fmtErr(s string) error { return tfErr(s) }
