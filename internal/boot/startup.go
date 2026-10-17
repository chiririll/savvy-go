package boot

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/firstrun"
	"savvy-go/internal/refsync"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store/sqlite"
	"savvy-go/internal/transfers"
)

// prepare brings the databases to a servable state: loads the signing key,
// checks the spaces and finishes interrupted transfers between them.
func prepare(ctx context.Context, cfg appconfig.Config, st *sqlite.Store) (*signing.Holder, signing.KeyRing, error) {
	keys, err := loadSigningKey(ctx, cfg.DataDir, st)
	if err != nil {
		return nil, signing.KeyRing{}, fmt.Errorf("signing key: %w", err)
	}
	if missing, orphans, err := (spaces.Service{Store: st}).Reconcile(ctx); err != nil {
		slog.Warn("check the spaces against their database files", "err", err)
	} else {
		for _, id := range missing {
			slog.Error("space has no database file", "space", id)
		}
		for _, id := range orphans {
			slog.Warn("space database file without a registered space; left untouched", "space", id)
		}
	}
	// Finish transfers a crash left half-written and review spaces restored
	// just before it; an unavailable space is merged on a later start.
	ring := signing.KeyRing{Holder: keys, Trusted: transfers.TrustedKeys(st.Server())}
	if err := (transfers.Service{Spaces: spaces.Service{Store: st}, Keys: ring}).SyncAll(ctx); err != nil {
		slog.Warn("merge transfers between spaces", "err", err)
	}
	// Bring the spaces of sync groups back together if they drifted apart.
	if err := (refsync.Service{Spaces: spaces.Service{Store: st}}).Heal(ctx); err != nil {
		slog.Warn("heal sync groups", "err", err)
	}
	for id, why := range st.Status().Unavailable {
		slog.Error("space unavailable", "space", id, "reason", why)
	}
	return keys, ring, nil
}

// startBootstrap creates the first-run password when the server has no users
// and prints it. It returns nil, which keeps registration closed, otherwise;
// that includes a demo-seeded server. A password file left from an earlier run
// is removed once users exist.
func startBootstrap(
	ctx context.Context,
	cfg appconfig.Config,
	listen string,
	st *sqlite.Store,
) (*firstrun.Guard, error) {
	n, err := (auth.Users{DB: st.Server()}).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		if err := os.Remove(filepath.Join(cfg.DataDir, firstrun.FileName)); err != nil && !os.IsNotExist(err) {
			slog.Warn("could not remove the stale bootstrap password file", "err", err)
		}
		return nil, nil
	}
	guard := firstrun.New(cfg.DataDir, firstrun.DetectEnv())
	guard.Announce(openURL(listen, cfg.AppURL()))
	return guard, nil
}

// openURL is where the operator opens the app: the public URL when set,
// otherwise the listen address on localhost.
func openURL(listenAddr, publicURL string) string {
	if publicURL != "" {
		return publicURL
	}
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "http://localhost"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
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
