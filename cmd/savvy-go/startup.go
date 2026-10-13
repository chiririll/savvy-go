package main

import (
	"context"
	"fmt"
	"log/slog"

	"savvy-go/internal/config"
	"savvy-go/internal/refsync"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store/sqlite"
	"savvy-go/internal/transfers"
)

// prepare brings the databases to a servable state: loads the signing key,
// checks the spaces and finishes interrupted transfers between them.
func prepare(ctx context.Context, cfg config.Config, st *sqlite.Store) (*signing.Holder, transfers.Service, error) {
	keys, err := loadSigningKey(ctx, cfg.DataDir, st)
	if err != nil {
		return nil, transfers.Service{}, fmt.Errorf("signing key: %w", err)
	}
	if missing, orphans, err := (spaces.Service{Store: st}).Reconcile(ctx); err == nil {
		for _, id := range missing {
			slog.Error("space has no database file", "space", id)
		}
		for _, id := range orphans {
			slog.Warn("space database file without a registered space; left untouched", "space", id)
		}
	}
	// Finish transfers a crash left half-written and review spaces restored
	// just before it; an unavailable space is merged on a later start.
	transfers := transfers.Service{
		Spaces: spaces.Service{Store: st},
		Keys:   signing.KeyRing{Holder: keys, Trusted: transfers.TrustedKeys(st.Server())},
	}
	if err := transfers.SyncAll(ctx); err != nil {
		slog.Warn("merge transfers between spaces", "err", err)
	}
	// Bring the spaces of sync groups back together if they drifted apart.
	if err := (refsync.Service{Spaces: spaces.Service{Store: st}}).Heal(ctx); err != nil {
		slog.Warn("heal sync groups", "err", err)
	}
	for id, why := range st.Status().Unavailable {
		slog.Error("space unavailable", "space", id, "reason", why)
	}
	return keys, transfers, nil
}

// signingKeyUsedKey records the key id once the server has a key, so a key
// file that disappears later is noticed instead of silently replaced.
const signingKeyUsedKey = "signing_kid"

func loadSigningKey(ctx context.Context, dataDir string, st *sqlite.Store) (*signing.Holder, error) {
	server := settings.Store{DB: st.Server()}
	kid, _ := server.Get(ctx, signingKeyUsedKey, "").(string)
	key, err := signing.Load(dataDir, kid != "")
	if err != nil {
		return nil, err
	}
	if kid == "" {
		if err := server.Set(ctx, signingKeyUsedKey, key.KID()); err != nil {
			return nil, err
		}
	} else if kid != key.KID() {
		slog.Warn("the signing key file is not the key this server used before", "recorded", kid, "file", key.KID())
	}
	return signing.NewHolder(key), nil
}
