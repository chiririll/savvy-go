package auth

import (
	"context"
	"strconv"
	"strings"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"

	"savvy-go/internal/auth/totp"
)

const recoveryChars = "abcdefghjkmnpqrstuvwxyz23456789"

type TwoFactor struct {
	DB    store.DB
	Users Users
}

func (t TwoFactor) repo() repo.Repo { return repo.Repo{DB: t.DB} }

// verifyTOTP accepts an authenticator code once: the step it matched must be
// newer than the last one accepted, and recording it is atomic, so of two
// requests with the same code only one passes.
func (t TwoFactor) verifyTOTP(ctx context.Context, u *User, code string) bool {
	if u == nil || u.TwoFactorSecret == nil {
		return false
	}
	step, ok := totp.VerifyStep(*u.TwoFactorSecret, code, time.Now())
	if !ok {
		return false
	}
	fresh, err := t.repo().AdvanceTwoFactorStep(ctx, u.ID, step)
	return err == nil && fresh
}

func (t TwoFactor) Enable(ctx context.Context, u *User) (secret, uri string, err error) {
	if u.HasTwoFactor() {
		return "", "", fmtErr("already enabled")
	}
	secret, err = totp.NewSecret()
	if err != nil {
		return "", "", err
	}
	err = store.Tx(ctx, t.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		if err := r.SetTwoFactor(ctx, u.ID, &secret, true, false, time.Now().UTC()); err != nil {
			return err
		}
		// A new secret starts clean: no codes from an earlier pending setup.
		if err := r.DeleteRecoveryCodes(ctx, u.ID); err != nil {
			return err
		}
		return r.ClearTwoFactorStep(ctx, u.ID)
	})
	if err != nil {
		return "", "", err
	}
	return secret, totp.ProvisioningURI(u.Email, secret), nil
}

func (t TwoFactor) Confirm(ctx context.Context, u *User, code string) ([]string, error) {
	if !u.TwoFactorEnabled || u.TwoFactorConfirmed {
		return nil, fmtErr("not pending")
	}
	if !t.verifyTOTP(ctx, u, code) {
		return nil, fmtErr("invalid code")
	}
	var codes []string
	err := store.Tx(ctx, t.DB, func(tx store.DB) (err error) {
		r := repo.Repo{DB: tx}
		if err = r.SetTwoFactor(ctx, u.ID, u.TwoFactorSecret, true, true, time.Now().UTC()); err != nil {
			return err
		}
		// Codes are made once, here: drop any left from an earlier attempt.
		if err = r.DeleteRecoveryCodes(ctx, u.ID); err != nil {
			return err
		}
		codes, err = generateCodes(ctx, r, u.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

func (t TwoFactor) Disable(ctx context.Context, u *User, code string) error {
	if !u.HasTwoFactor() {
		return fmtErr("not enabled")
	}
	if !t.VerifyAny(ctx, u, code) {
		return fmtErr("invalid code")
	}
	return store.Tx(ctx, t.DB, func(tx store.DB) error {
		r := repo.Repo{DB: tx}
		if err := r.SetTwoFactor(ctx, u.ID, nil, false, false, time.Now().UTC()); err != nil {
			return err
		}
		if err := r.DeleteRecoveryCodes(ctx, u.ID); err != nil {
			return err
		}
		return r.ClearTwoFactorStep(ctx, u.ID)
	})
}

func (t TwoFactor) VerifyAny(ctx context.Context, u *User, code string) bool {
	if t.verifyTOTP(ctx, u, code) {
		return true
	}
	return t.ConsumeRecovery(ctx, u.ID, code)
}

func (t TwoFactor) ConsumeRecovery(ctx context.Context, userID int64, code string) bool {
	ok, err := t.repo().UseRecoveryCode(ctx, userID, recoveryHash(userID, code), time.Now().UTC())
	return err == nil && ok
}

// recoveryHash is what is stored for a code. A code has about 40 random bits and a person
// never picks it, so SHA-256 is enough and lets the lookup be an index hit instead of a
// bcrypt compare per row. The user id keeps equal codes of two users apart.
func recoveryHash(userID int64, code string) string {
	return HashToken(strconv.FormatInt(userID, 10) + ":" + strings.ToLower(strings.TrimSpace(code)))
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
	var codes []string
	err := store.Tx(ctx, t.DB, func(tx store.DB) (err error) {
		r := repo.Repo{DB: tx}
		if err = r.InvalidateUnusedRecoveryCodes(ctx, u.ID, time.Now().UTC()); err != nil {
			return err
		}
		codes, err = generateCodes(ctx, r, u.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// generateCodes stores 8 new codes through r, which must be a transaction so
// a failure leaves none of them behind.
func generateCodes(ctx context.Context, r repo.Repo, userID int64) ([]string, error) {
	var codes []string
	now := time.Now().UTC()
	for i := 0; i < 8; i++ {
		plain := recoveryCode()
		if err := r.InsertRecoveryCode(ctx, userID, recoveryHash(userID, plain), now); err != nil {
			return nil, err
		}
		codes = append(codes, plain)
	}
	return codes, nil
}

func recoveryCode() string {
	s := randomFrom(recoveryChars, 8)
	return s[:4] + "-" + s[4:]
}

type tfErr string

func (e tfErr) Error() string { return string(e) }

func fmtErr(s string) error { return tfErr(s) }
