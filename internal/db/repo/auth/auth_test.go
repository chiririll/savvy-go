package auth_test

import (
	"context"
	"testing"
	"time"

	authrepo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store/sqlite"
)

func newRepo(t *testing.T) (authrepo.Repo, int64) {
	t.Helper()
	st, err := sqlite.OpenApp(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	r := authrepo.Repo{DB: st.Server()}
	id, err := r.InsertUser(context.Background(), "Ann", "ann@test.com", "hash", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return r, id
}

func TestMissingRowsAreNilNotErrors(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	if u, err := r.UserByID(ctx, 999); u != nil || err != nil {
		t.Fatalf("user: %v, %v", u, err)
	}
	if u, err := r.UserByEmail(ctx, "nobody@test.com"); u != nil || err != nil {
		t.Fatalf("user by email: %v, %v", u, err)
	}
	if s, err := r.SessionByTokenHash(ctx, "x"); s != nil || err != nil {
		t.Fatalf("session: %v, %v", s, err)
	}
	if tok, err := r.APITokenByHash(ctx, "x"); tok != nil || err != nil {
		t.Fatalf("api token: %v, %v", tok, err)
	}
	if id, err := r.PasskeyUserID(ctx, "x"); id != nil || err != nil {
		t.Fatalf("passkey owner: %v, %v", id, err)
	}
}

func TestUserRoundTripAndPasswordOptional(t *testing.T) {
	r, id := newRepo(t)
	ctx := context.Background()
	u, _ := r.UserByEmail(ctx, "ann@test.com")
	if u == nil || u.ID != id || u.Password == nil || *u.Password != "hash" || u.CreatedAt == nil {
		t.Fatalf("user = %+v", u)
	}

	noPass, err := r.InsertUser(ctx, "Sso", "sso@test.com", "", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := r.UserByID(ctx, noPass); u.Password != nil {
		t.Fatalf("an empty hash must store no password, got %q", *u.Password)
	}

	// An empty hash on update keeps the password; a non-empty one replaces it.
	if err := r.UpdateUser(ctx, id, "Ann B", "ann@test.com", "admin", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.UserByID(ctx, id); u.Name != "Ann B" || u.Role != "admin" || *u.Password != "hash" {
		t.Fatalf("after profile update: %+v", u)
	}
	if err := r.UpdateUser(ctx, id, "Ann B", "ann@test.com", "admin", "new", time.Now()); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.UserByID(ctx, id); *u.Password != "new" {
		t.Fatalf("password = %q", *u.Password)
	}
	if n, _ := r.CountAdmins(ctx, "admin"); n != 1 {
		t.Fatalf("admins = %d", n)
	}
}

func TestTombstoneKeepsTheRowAndDropsAccess(t *testing.T) {
	r, id := newRepo(t)
	ctx := context.Background()
	if _, err := r.InsertAPIToken(ctx, authrepo.NewAPIToken{UserID: id, Name: "t", Hash: "h", Prefix: "svy_", AllSpaces: true, Now: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := r.InsertRecoveryCode(ctx, id, "c", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := r.Tombstone(ctx, id, time.Now()); err != nil {
		t.Fatal(err)
	}
	u, _ := r.UserByID(ctx, id)
	if u == nil || !u.Deleted || u.Password != nil {
		t.Fatalf("tombstone = %+v", u)
	}
	if tokens, _ := r.APITokens(ctx, id); len(tokens) != 0 {
		t.Fatalf("%d tokens survived", len(tokens))
	}
	if n, _ := r.CountUnusedRecoveryCodes(ctx, id); n != 0 {
		t.Fatalf("%d recovery codes survived", n)
	}
}

func TestSessionLifecycle(t *testing.T) {
	r, id := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	newSess := func(hash string) int64 {
		sid, err := r.InsertSession(ctx, authrepo.NewSession{UserID: id, TokenHash: hash, CSRF: "c", IP: "1.2.3.4",
			UserAgent: "ua", RememberMe: true, Expires: now.Add(time.Hour), Now: now})
		if err != nil {
			t.Fatal(err)
		}
		return sid
	}
	keep, other := newSess("a"), newSess("b")

	s, _ := r.SessionByTokenHash(ctx, "a")
	if s == nil || s.ID != keep || !s.RememberMe || s.RevokedAt != nil || !s.AbsoluteExpiresAt.Equal(now.Add(time.Hour)) ||
		s.RefreshedAt == nil || !s.CreatedAt.Equal(now) {
		t.Fatalf("session = %+v", s)
	}

	later := now.Add(10 * time.Minute)
	if err := r.RotateSession(ctx, keep, "a2", "c2", later, later.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if old, _ := r.SessionByTokenHash(ctx, "a"); old != nil {
		t.Fatal("the old token must stop resolving after a rotation")
	}
	if s, _ := r.SessionByTokenHash(ctx, "a2"); s == nil || s.CSRF != "c2" || !s.RefreshedAt.Equal(later) {
		t.Fatalf("rotated = %+v", s)
	}

	if n, err := r.RevokeOtherSessions(ctx, id, keep, later); err != nil || n != 1 {
		t.Fatalf("revoked %d, %v; want 1", n, err)
	}
	if s, _ := r.SessionByTokenHash(ctx, "b"); s.RevokedAt == nil || s.ID != other {
		t.Fatalf("other session = %+v", s)
	}
	if s, _ := r.SessionByTokenHash(ctx, "a2"); s.RevokedAt != nil {
		t.Fatal("the kept session was revoked")
	}
}

func TestOneTimeTokensWorkOnceAndExpire(t *testing.T) {
	r, id := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := r.InsertPasswordToken(ctx, id, "live", now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if err := r.InsertPasswordToken(ctx, id, "dead", now.Add(-time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if tok, _ := r.ValidPasswordToken(ctx, "live", now); tok == nil || tok.UserID != id {
		t.Fatalf("live token = %+v", tok)
	}
	if tok, _ := r.ValidPasswordToken(ctx, "dead", now); tok != nil {
		t.Fatal("an expired token is not valid")
	}
	if ok, _ := r.ConsumePasswordToken(ctx, "dead", now); ok {
		t.Fatal("an expired token must not be consumed")
	}
	if ok, err := r.ConsumePasswordToken(ctx, "live", now); !ok || err != nil {
		t.Fatalf("first consume = %v, %v", ok, err)
	}
	if ok, _ := r.ConsumePasswordToken(ctx, "live", now); ok {
		t.Fatal("a token must work only once")
	}
	if tok, _ := r.PasswordTokenByHash(ctx, "live"); tok == nil {
		t.Fatal("a used token is still found by its hash")
	}
}

func TestChallengesWorkOnce(t *testing.T) {
	r, id := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := r.InsertChallenge(ctx, id, "ch", now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if uid, _ := r.ChallengeUserID(ctx, "ch", now); uid == nil || *uid != id {
		t.Fatalf("challenge owner = %v", uid)
	}
	if uid, _ := r.ChallengeUserID(ctx, "ch", now.Add(2*time.Minute)); uid != nil {
		t.Fatal("an expired challenge must not resolve")
	}
	if ok, _ := r.ConsumeChallenge(ctx, "ch", now); !ok {
		t.Fatal("first consume failed")
	}
	if ok, _ := r.ConsumeChallenge(ctx, "ch", now); ok {
		t.Fatal("a challenge must work only once")
	}
	if got, _ := r.AnyChallengeUserID(ctx, "ch"); got != id {
		t.Fatalf("owner after use = %d", got)
	}
}

func TestPasskeysBelongToTheirOwner(t *testing.T) {
	r, id := newRepo(t)
	ctx := context.Background()
	other, _ := r.InsertUser(ctx, "Bob", "bob@test.com", "h", "user", time.Now())
	pid, err := r.InsertPasskey(ctx, authrepo.NewPasskey{UserID: id, CredentialID: "cred", Name: "Laptop", AAGUID: "ab", Record: "{}", Now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if owner, _ := r.PasskeyUserID(ctx, "cred"); owner == nil || *owner != id {
		t.Fatalf("owner = %v", owner)
	}
	if ok, _ := r.RenamePasskey(ctx, other, pid, "Mine now", time.Now()); ok {
		t.Fatal("another user renamed a passkey")
	}
	if ok, _ := r.RenamePasskey(ctx, id, pid, "Phone", time.Now()); !ok {
		t.Fatal("the owner could not rename it")
	}
	list, _ := r.Passkeys(ctx, id)
	if len(list) != 1 || list[0].Name == nil || *list[0].Name != "Phone" || list[0].LastUsedAt != nil {
		t.Fatalf("passkeys = %+v", list)
	}
	if ok, _ := r.DeletePasskey(ctx, other, pid); ok {
		t.Fatal("another user deleted a passkey")
	}
	if ok, _ := r.DeletePasskey(ctx, id, pid); !ok {
		t.Fatal("the owner could not delete it")
	}
	if n, _ := r.CountPasskeys(ctx, id); n != 0 {
		t.Fatalf("count = %d", n)
	}
}

func TestPasskeyChallengeIsTakenOnceAndPerType(t *testing.T) {
	r, id := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := r.InsertPasskeyChallenge(ctx, &id, "tok", "registration", `{"x":1}`, now.Add(5*time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if c, _ := r.TakePasskeyChallenge(ctx, "tok", "authentication", now); c != nil {
		t.Fatal("a challenge of another type must not be taken")
	}
	c, _ := r.TakePasskeyChallenge(ctx, "tok", "registration", now)
	if c == nil || c.UserID != id || c.Options != `{"x":1}` {
		t.Fatalf("challenge = %+v", c)
	}
	if again, _ := r.TakePasskeyChallenge(ctx, "tok", "registration", now); again != nil {
		t.Fatal("a challenge must be taken only once")
	}

	// A login that does not name the user has no owner.
	if err := r.InsertPasskeyChallenge(ctx, nil, "anon", "authentication", "{}", now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if c, _ := r.TakePasskeyChallenge(ctx, "anon", "authentication", now); c == nil || c.UserID != 0 {
		t.Fatalf("anonymous challenge = %+v", c)
	}
}
