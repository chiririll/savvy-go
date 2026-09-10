package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"strings"
	"time"

	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/db/sqlc"
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
	_ = db.Q(t.DB).DeleteRecoveryCodes(ctx, u.ID)
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
	found, err := db.Q(t.DB).ListUnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return false
	}
	for _, r := range found {
		if CheckPassword(r.Code, code) {
			now := time.Now().UTC().Format(time.RFC3339)
			_ = db.Q(t.DB).MarkRecoveryCodeUsed(ctx, sqlc.MarkRecoveryCodeUsedParams{UsedAt: db.NS(now), ID: r.ID})
			return true
		}
	}
	return false
}

func (t TwoFactor) Remaining(ctx context.Context, userID int64) int {
	n, _ := db.Q(t.DB).CountUnusedRecoveryCodes(ctx, userID)
	return int(n)
}

func (t TwoFactor) Regenerate(ctx context.Context, u *User, code string) ([]string, error) {
	if !u.HasTwoFactor() {
		return nil, fmtErr("not enabled")
	}
	if !t.verifyTOTP(ctx, u, code) {
		return nil, fmtErr("invalid code")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_ = db.Q(t.DB).InvalidateUnusedRecoveryCodes(ctx, sqlc.InvalidateUnusedRecoveryCodesParams{UsedAt: db.NS(now), UserID: u.ID})
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
		if err := db.Q(t.DB).InsertRecoveryCode(ctx, sqlc.InsertRecoveryCodeParams{UserID: userID, Code: h, CreatedAt: db.NS(now)}); err != nil {
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
