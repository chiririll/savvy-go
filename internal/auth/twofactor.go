package auth

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"
)

const recoveryChars = "abcdefghjkmnpqrstuvwxyz23456789"

type TwoFactor struct {
	DB    store.DB
	Users Users
}

func (t TwoFactor) repo() repo.Repo { return repo.Repo{DB: t.DB} }

func (t TwoFactor) verifyTOTP(_ context.Context, u *User, code string) bool {
	if u == nil || u.TwoFactorSecret == nil {
		return false
	}
	return VerifyTOTP(*u.TwoFactorSecret, code)
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
	_ = t.repo().DeleteRecoveryCodes(ctx, u.ID)
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
	unused, err := t.repo().UnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return false
	}
	for _, r := range unused {
		if CheckPassword(r.Hash, code) {
			_ = t.repo().MarkRecoveryCodeUsed(ctx, r.ID, time.Now().UTC())
			return true
		}
	}
	return false
}

func (t TwoFactor) Remaining(ctx context.Context, userID int64) int {
	n, _ := t.repo().CountUnusedRecoveryCodes(ctx, userID)
	return n
}

func (t TwoFactor) Regenerate(ctx context.Context, u *User, code string) ([]string, error) {
	if !u.HasTwoFactor() {
		return nil, fmtErr("not enabled")
	}
	if !t.verifyTOTP(ctx, u, code) {
		return nil, fmtErr("invalid code")
	}
	_ = t.repo().InvalidateUnusedRecoveryCodes(ctx, u.ID, time.Now().UTC())
	return t.generateCodes(ctx, u.ID)
}

func (t TwoFactor) generateCodes(ctx context.Context, userID int64) ([]string, error) {
	var codes []string
	now := time.Now().UTC()
	for i := 0; i < 8; i++ {
		plain := recoveryCode()
		h, err := HashPassword(plain)
		if err != nil {
			return nil, err
		}
		if err := t.repo().InsertRecoveryCode(ctx, userID, h, now); err != nil {
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
