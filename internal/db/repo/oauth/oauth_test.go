package oauth_test

import (
	"context"
	"testing"
	"time"

	authrepo "savvy-go/internal/db/repo/auth"
	repo "savvy-go/internal/db/repo/oauth"
	"savvy-go/internal/store/sqlite/sqlitetest"
)

func newRepos(t *testing.T) (repo.Repo, authrepo.Repo, int64) {
	t.Helper()
	st := sqlitetest.Open(t)
	a := authrepo.Repo{DB: st.Server()}
	uid, err := a.InsertUser(context.Background(), "Ann", "ann@test.com", "hash", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return repo.Repo{DB: st.Server()}, a, uid
}

func token(t *testing.T, a authrepo.Repo, uid int64, hash string) int64 {
	t.Helper()
	id, err := a.InsertAPIToken(
		context.Background(),
		authrepo.NewAPIToken{UserID: uid, Name: "t", Hash: hash, Prefix: "svy_", AllSpaces: true, Now: time.Now()},
	)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMissingRowsAreNil(t *testing.T) {
	r, _, _ := newRepos(t)
	ctx := context.Background()
	if c, err := r.Client(ctx, "nope"); c != nil || err != nil {
		t.Fatalf("client: %v, %v", c, err)
	}
	if rt, err := r.RefreshByHash(ctx, "nope"); rt != nil || err != nil {
		t.Fatalf("refresh: %v, %v", rt, err)
	}
}

func TestClientRoundTripKeepsRedirectUR(t *testing.T) {
	r, _, _ := newRepos(t)
	ctx := context.Background()
	uris := []string{"https://claude.ai/api/mcp/auth_callback", "http://localhost:6274/cb"}
	if err := r.InsertClient(ctx, "c1", "Claude", uris, repo.RequestInfo{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	c, _ := r.Client(ctx, "c1")
	if c == nil || c.Name != "Claude" || len(c.RedirectURIs) != 2 || c.RedirectURIs[1] != uris[1] {
		t.Fatalf("client = %+v", c)
	}
	if n, _ := r.CountClients(ctx); n != 1 {
		t.Fatalf("count = %d", n)
	}
}

func TestPruneKeepsClientsHoldRefreshToken(t *testing.T) {
	r, a, uid := newRepos(t)
	ctx := context.Background()
	old := time.Now().Add(-48 * time.Hour)
	_ = r.InsertClient(ctx, "idle", "Idle", []string{"https://x/cb"}, repo.RequestInfo{}, old)
	_ = r.InsertClient(ctx, "used", "Used", []string{"https://x/cb"}, repo.RequestInfo{}, old)
	_ = r.InsertClient(ctx, "fresh", "Fresh", []string{"https://x/cb"}, repo.RequestInfo{}, time.Now())
	tok := token(t, a, uid, "h")
	if err := r.InsertRefresh(ctx,
		repo.NewRefresh{Hash: "r1", ClientID: "used", UserID: uid, APITokenID: tok, ExpiresAt: time.Now().Add(time.Hour),
			Now: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := r.PruneClients(ctx, time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]bool{"idle": false, "used": true, "fresh": true} {
		if c, _ := r.Client(ctx, id); (c != nil) != want {
			t.Errorf("client %s present = %v, want %v", id, c != nil, want)
		}
	}
}

func TestRotateNeedsTheCurrentHash(t *testing.T) {
	r, a, uid := newRepos(t)
	ctx := context.Background()
	_ = r.InsertClient(ctx, "c", "C", []string{"https://x/cb"}, repo.RequestInfo{}, time.Now())
	tok := token(t, a, uid, "h1")
	exp := time.Now().Add(time.Hour)
	if err := r.InsertRefresh(ctx,
		repo.NewRefresh{Hash: "r1", ClientID: "c", UserID: uid, APITokenID: tok, ExpiresAt: exp,
			Now: time.Now()}); err != nil {
		t.Fatal(err)
	}
	row, _ := r.RefreshByHash(ctx, "r1")
	if row == nil || row.UserID != uid || row.APITokenID != tok || row.ClientID != "c" {
		t.Fatalf("row = %+v", row)
	}
	tok2 := token(t, a, uid, "h2")
	ok, err := r.RotateRefresh(ctx, row.ID, "r1", "r2", tok2, exp)
	if err != nil || !ok {
		t.Fatalf("rotate = %v, %v", ok, err)
	}
	if ok, _ := r.RotateRefresh(ctx, row.ID, "r1", "r3", tok2, exp); ok {
		t.Fatal("rotating with a stale hash must fail")
	}
	if old, _ := r.RefreshByHash(ctx, "r1"); old != nil {
		t.Fatal("old hash still resolves")
	}
}

func TestRevokingAccessTokenRemovesRefreshToken(t *testing.T) {
	r, a, uid := newRepos(t)
	ctx := context.Background()
	_ = r.InsertClient(ctx, "c", "C", []string{"https://x/cb"}, repo.RequestInfo{}, time.Now())
	tok := token(t, a, uid, "h1")
	_ = r.InsertRefresh(
		ctx,
		repo.NewRefresh{
			Hash:       "r1",
			ClientID:   "c",
			UserID:     uid,
			APITokenID: tok,
			ExpiresAt:  time.Now().Add(time.Hour),
			Now:        time.Now(),
		},
	)
	if ok, _ := a.DeleteAPIToken(ctx, uid, tok); !ok {
		t.Fatal("delete token")
	}
	if row, _ := r.RefreshByHash(ctx, "r1"); row != nil {
		t.Fatal("refresh token must die with its access token")
	}
}

func TestDeleteClientRevokesOwnTokensOnly(t *testing.T) {
	r, a, uid := newRepos(t)
	ctx := context.Background()
	_ = r.InsertClient(ctx, "c", "C", []string{"https://x/cb"}, repo.RequestInfo{}, time.Now())
	oauthTok, plain := token(t, a, uid, "h1"), token(t, a, uid, "h2")
	_ = r.InsertRefresh(
		ctx,
		repo.NewRefresh{
			Hash:       "r1",
			ClientID:   "c",
			UserID:     uid,
			APITokenID: oauthTok,
			ExpiresAt:  time.Now().Add(time.Hour),
			Now:        time.Now(),
		},
	)
	if ok, err := r.DeleteClient(ctx, "c"); !ok || err != nil {
		t.Fatalf("delete = %v, %v", ok, err)
	}
	if row, _ := a.APITokenByHash(ctx, "h1"); row != nil {
		t.Fatal("the app's access token survived")
	}
	if row, _ := a.APITokenByHash(ctx, "h2"); row == nil {
		t.Fatalf("an unrelated token %d was deleted", plain)
	}
	if ok, _ := r.DeleteClient(ctx, "c"); ok {
		t.Fatal("unknown client")
	}
}

func TestClientKeepsRequestInfo(t *testing.T) {
	r, _, _ := newRepos(t)
	ctx := context.Background()
	info := repo.RequestInfo{IP: "203.0.113.7", UserAgent: "UA/1", Origin: "https://claude.ai"}
	_ = r.InsertClient(ctx, "c", "C", []string{"https://x/cb"}, info, time.Now())
	c, _ := r.Client(ctx, "c")
	if c == nil || c.Request != info || c.CreatedAt.IsZero() {
		t.Fatalf("client = %+v", c)
	}
	list, err := r.Clients(ctx)
	if err != nil || len(list) != 1 || list[0].ID != "c" || list[0].Request != info {
		t.Fatalf("clients = %+v, %v", list, err)
	}
}
