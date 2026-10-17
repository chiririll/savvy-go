package httpserver

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
	"savvy-go/internal/mail/smtp"
	"savvy-go/internal/sso"
)

var linkRe = regexp.MustCompile(`https?://\S+`)

// linkIn is the one link of an email.
func linkIn(t *testing.T, m smtp.Message) *url.URL {
	t.Helper()
	raw := linkRe.FindString(m.Text)
	u, err := url.Parse(raw)
	if raw == "" || err != nil {
		t.Fatalf("no link in %q", m.Text)
	}
	return u
}

func (a *testApp) setting(key string) any {
	a.t.Helper()
	return a.svc.Settings.Get(context.Background(), key, nil)
}

func (a *testApp) verified(id int64) bool {
	a.t.Helper()
	u, err := a.svc.Users.ByID(context.Background(), id)
	if err != nil || u == nil {
		a.t.Fatalf("user %d: %v", id, err)
	}
	return u.EmailVerified
}

func TestRequiringVerificationNeedsAVerifiedAdmin(t *testing.T) {
	a, sess, admin := newMailApp(t, &recorder{on: true})
	on := map[string]any{"require_email_verification": true}

	// The admin has not proven their address: switching it on would lock them out.
	res := a.do("PATCH", "/api/settings", on, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 422 || body["error"] != "email_unverified" {
		t.Fatalf("unverified admin: %d %v", res.StatusCode, body)
	}
	if a.setting("require_email_verification") != false {
		t.Fatal("the setting changed anyway")
	}

	if err := a.svc.Users.SetEmailVerified(context.Background(), admin.ID, true); err != nil {
		t.Fatal(err)
	}
	res = a.do("PATCH", "/api/settings", on, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 200 || body["require_email_verification"] != true {
		t.Fatalf("verified admin: %d %v", res.StatusCode, body)
	}

	// Switching it off is always allowed.
	res = a.do("PATCH", "/api/settings", map[string]any{"require_email_verification": false}, sess.Token, sess.CSRF)
	if res.StatusCode != 200 || a.setting("require_email_verification") != false {
		t.Fatalf("switch off: %d", res.StatusCode)
	}
}

func TestRequiringVerificationNeedsMail(t *testing.T) {
	a, sess, admin := newMailApp(t, &recorder{on: false})
	if err := a.svc.Users.SetEmailVerified(context.Background(), admin.ID, true); err != nil {
		t.Fatal(err)
	}
	res := a.do("PATCH", "/api/settings", map[string]any{"require_email_verification": true}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 422 || body["error"] != "mail_required" {
		t.Fatalf("without mail: %d %v", res.StatusCode, body)
	}
	if a.setting("require_email_verification") != false {
		t.Fatal("the setting changed anyway")
	}
}

func TestUnverifiedUserIsHeldBackOnlyWhileVerificationIsRequired(t *testing.T) {
	a, sess, admin := newMailApp(t, &recorder{on: true})
	if err := a.svc.Users.SetEmailVerified(context.Background(), admin.ID, true); err != nil {
		t.Fatal(err)
	}
	user := a.createUser("u@test.com", "secret1", roleEditor)
	us := a.issue(user, false)

	// Not required: nothing changes.
	if res := a.do("GET", "/api/spaces", nil, us.Token, us.CSRF); res.StatusCode != 200 {
		t.Fatalf("not required: %d", res.StatusCode)
	}

	if err := a.svc.Settings.Set(context.Background(), "require_email_verification", true); err != nil {
		t.Fatal(err)
	}
	res := a.do("GET", "/api/spaces", nil, us.Token, us.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 403 || body["error"] != "email_unverified" {
		t.Fatalf("required: %d %v", res.StatusCode, body)
	}
	if res := a.do("GET", "/api/spaces", nil, sess.Token, sess.CSRF); res.StatusCode != 200 {
		t.Fatalf("verified admin: %d", res.StatusCode)
	}

	// Signing in still works, and the session says what is missing.
	res = a.do("POST", "/api/auth/login", map[string]any{"email": "u@test.com", "password": "secret1"}, "", "")
	if res.StatusCode != 200 {
		t.Fatalf("login: %d", res.StatusCode)
	}
	me := decodeJSON(t, a.do("GET", "/api/auth/me", nil, us.Token, us.CSRF))
	u := me["user"].(map[string]any)
	if u["emailVerified"] != false || me["email_verification_required"] != true {
		t.Fatalf("me = %v", me)
	}

	// Leaving stays possible.
	if res := a.do("POST", "/api/auth/logout", nil, us.Token, us.CSRF); res.StatusCode != 200 {
		t.Fatalf("logout: %d", res.StatusCode)
	}
}

func TestUnverifiedUserIsHeldBackOnAPITokensToo(t *testing.T) {
	a, _, admin := newMailApp(t, &recorder{on: true})
	if err := a.svc.Users.SetEmailVerified(context.Background(), admin.ID, true); err != nil {
		t.Fatal(err)
	}
	user := a.createUser("u@test.com", "secret1", roleEditor)
	us := a.issue(user, false)
	res := a.do("POST", "/api/auth/api-tokens", map[string]any{"name": "t", "permissions": allPerms(permissions.Read), "all_spaces": true}, us.Token, us.CSRF)
	raw, _ := decodeJSON(t, res)["data"].(map[string]any)["token"].(string)
	if res.StatusCode != 201 || raw == "" {
		t.Fatalf("mint a token: %d", res.StatusCode)
	}
	bearer := func() int {
		req := a.request("GET", "/api/spaces", nil, "", "")
		req.Header.Set("Authorization", "Bearer "+raw)
		res, err := a.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode
	}
	if got := bearer(); got != 200 {
		t.Fatalf("token while not required: %d", got)
	}
	if err := a.svc.Settings.Set(context.Background(), "require_email_verification", true); err != nil {
		t.Fatal(err)
	}
	if got := bearer(); got != 403 {
		t.Fatalf("token of an unverified user: %d", got)
	}
}

func TestVerifyEmailByLink(t *testing.T) {
	rec := &recorder{on: true}
	a, _, _ := newMailApp(t, rec)
	user := a.createUser("u@test.com", "secret1", roleEditor)
	us := a.issue(user, false)

	res := a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("resend: %d %v", res.StatusCode, decodeJSON(t, res))
	}
	msgs := rec.messages()
	if len(msgs) != 1 || msgs[0].To != "u@test.com" {
		t.Fatalf("messages = %+v", msgs)
	}
	link := linkIn(t, msgs[0])
	token := strings.TrimPrefix(link.Path, "/verify-email/")
	if token == link.Path || token == "" {
		t.Fatalf("link %s", link)
	}

	// The link is opened without a session, maybe on another device.
	if res := a.do("POST", "/api/auth/verify-email", map[string]any{"token": token + "x"}, "", ""); res.StatusCode != 422 {
		t.Fatalf("tampered token: %d", res.StatusCode)
	}
	if a.verified(user.ID) {
		t.Fatal("verified by a bad token")
	}
	if res := a.do("POST", "/api/auth/verify-email", map[string]any{"token": token}, "", ""); res.StatusCode != 200 {
		t.Fatalf("verify: %d", res.StatusCode)
	}
	if !a.verified(user.ID) {
		t.Fatal("not verified")
	}
	// Nothing left to send.
	if res := a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF); res.StatusCode != 422 {
		t.Fatalf("resend when verified: %d", res.StatusCode)
	}
}

func TestVerificationLinkIsOpaqueSingleUseAndReplaced(t *testing.T) {
	rec := &recorder{on: true}
	a, _, _ := newMailApp(t, rec)
	user := a.createUser("u@test.com", "secret1", roleEditor)
	us := a.issue(user, false)
	tokenOf := func(m smtp.Message) string {
		return strings.TrimPrefix(linkIn(t, m).Path, "/verify-email/")
	}
	verify := func(token string) int {
		res := a.do("POST", "/api/auth/verify-email", map[string]any{"token": token}, "", "")
		res.Body.Close()
		return res.StatusCode
	}

	a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF)
	a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF)
	msgs := rec.messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d", len(msgs))
	}
	first, second := tokenOf(msgs[0]), tokenOf(msgs[1])
	if link := linkIn(t, msgs[1]); link.RawQuery != "" || strings.ContainsAny(second, ".+/=") || len(second) < 40 {
		t.Fatalf("link %s: want /verify-email/<one opaque token>", link)
	}

	if got := verify(first); got != 422 {
		t.Fatalf("the replaced link: %d", got)
	}
	if got := verify(second); got != 200 || !a.verified(user.ID) {
		t.Fatalf("the newest link: %d", got)
	}
	if got := verify(second); got != 422 {
		t.Fatalf("a link used twice: %d", got)
	}
}

func TestVerificationLinkDiesWithTheAddress(t *testing.T) {
	rec := &recorder{on: true}
	a, admin, _ := newMailApp(t, rec)
	user := a.createUser("u@test.com", "secret1", roleEditor)
	us := a.issue(user, false)
	a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF)
	token := strings.TrimPrefix(linkIn(t, rec.messages()[0]).Path, "/verify-email/")

	if res := a.do("PATCH", "/api/users/"+itoa(user.ID), map[string]any{"email": "new@test.com"}, admin.Token, admin.CSRF); res.StatusCode != 200 {
		t.Fatalf("change address: %d", res.StatusCode)
	}
	if res := a.do("POST", "/api/auth/verify-email", map[string]any{"token": token}, "", ""); res.StatusCode != 422 {
		t.Fatalf("link for the old address: %d", res.StatusCode)
	}
	if a.verified(user.ID) {
		t.Fatal("verified through an old link")
	}
}

func TestVerificationLinkExpires(t *testing.T) {
	rec := &recorder{on: true}
	a, _, _ := newMailApp(t, rec)
	user := a.createUser("u@test.com", "secret1", roleEditor)
	us := a.issue(user, false)
	a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF).Body.Close()
	token := strings.TrimPrefix(linkIn(t, rec.messages()[0]).Path, "/verify-email/")

	if _, err := a.db.Exec(`UPDATE email_verification_tokens SET expires_at = ?`, time.Now().Add(-time.Minute).UTC()); err != nil {
		t.Fatal(err)
	}
	if res := a.do("POST", "/api/auth/verify-email", map[string]any{"token": token}, "", ""); res.StatusCode != 422 {
		t.Fatalf("an expired link: %d", res.StatusCode)
	}
	if a.verified(user.ID) {
		t.Fatal("verified through an expired link")
	}

	// The user asks again and the new link works.
	a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF).Body.Close()
	fresh := strings.TrimPrefix(linkIn(t, rec.messages()[1]).Path, "/verify-email/")
	if res := a.do("POST", "/api/auth/verify-email", map[string]any{"token": fresh}, "", ""); res.StatusCode != 200 || !a.verified(user.ID) {
		t.Fatalf("the new link: %d", res.StatusCode)
	}
}

func TestResendNeedsMail(t *testing.T) {
	a, _, _ := newMailApp(t, &recorder{on: false})
	user := a.createUser("u@test.com", "secret1", roleEditor)
	us := a.issue(user, false)
	if res := a.do("POST", "/api/auth/verify-email/resend", nil, us.Token, us.CSRF); res.StatusCode != 422 {
		t.Fatalf("without mail: %d", res.StatusCode)
	}
}

func TestAdminSetsVerifiedByHandAndAddressChangeClearsIt(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	create := a.do("POST", "/api/users", map[string]any{"name": "N", "email": "n@test.com", "password": "secret12", "role": "user", "emailVerified": true}, sess.Token, sess.CSRF)
	data := decodeJSON(t, create)["data"].(map[string]any)
	id := int64(data["id"].(float64))
	if create.StatusCode != 201 || data["emailVerified"] != true {
		t.Fatalf("create: %d %v", create.StatusCode, data)
	}

	res := a.do("PATCH", "/api/users/"+itoa(id), map[string]any{"emailVerified": false}, sess.Token, sess.CSRF)
	if got := decodeJSON(t, res)["data"].(map[string]any); got["emailVerified"] != false {
		t.Fatalf("unmark: %v", got)
	}
	res = a.do("PATCH", "/api/users/"+itoa(id), map[string]any{"emailVerified": true}, sess.Token, sess.CSRF)
	if got := decodeJSON(t, res)["data"].(map[string]any); got["emailVerified"] != true {
		t.Fatalf("mark: %v", got)
	}

	// Renaming keeps it; a new address does not.
	a.do("PATCH", "/api/users/"+itoa(id), map[string]any{"name": "Other"}, sess.Token, sess.CSRF)
	if !a.verified(id) {
		t.Fatal("a rename cleared the flag")
	}
	a.do("PATCH", "/api/users/"+itoa(id), map[string]any{"email": "changed@test.com"}, sess.Token, sess.CSRF)
	if a.verified(id) {
		t.Fatal("a new address stayed verified")
	}
	// Changing the address and marking it verified in one request is the admin's word.
	a.do("PATCH", "/api/users/"+itoa(id), map[string]any{"email": "again@test.com", "emailVerified": true}, sess.Token, sess.CSRF)
	if !a.verified(id) {
		t.Fatal("the admin's explicit mark was lost")
	}
}

func TestFirstAdminStartsUnverified(t *testing.T) {
	a := newTestApp(t)
	if res := a.register(a.bootstrap.Password(), nil); res.StatusCode != 201 {
		t.Fatalf("register: %d", res.StatusCode)
	}
	u, _ := a.svc.Users.ByEmail(context.Background(), "a@a.com")
	if u == nil || u.EmailVerified {
		t.Fatalf("the first admin must verify like everyone else: %+v", u)
	}
}

func TestSSOAddressVerifiedOnlyWhenProviderVouches(t *testing.T) {
	a := newTestApp(t)
	a.createUser("root@test.com", "secret1", auth.RoleAdmin)
	existing := a.createUser("jane@test.com", "secret1", roleEditor)
	p, err := a.svc.SSO.Create(context.Background(), sso.IdPWrite{
		Name: "IdP", Slug: "idp", Preset: "custom_oidc", Enabled: boolPtr(true),
		Fields: map[string]any{
			"discovery_url": "https://idp.test/.well-known/openid-configuration",
			"client_id":     "c", "client_secret": "s",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	provision := func(subject, email string, verified bool) *auth.User {
		t.Helper()
		u, _, err := a.svc.SSO.Provision(context.Background(), *p, sso.NormalizedIdentity{
			Subject: subject, Email: email, EmailVerified: verified, Name: "N",
		})
		if err != nil {
			t.Fatalf("provision %s: %v", email, err)
		}
		return u
	}

	if u := provision("s1", "vouched@test.com", true); !u.EmailVerified || !a.verified(u.ID) {
		t.Fatal("an address the provider vouches for must be verified")
	}
	if u := provision("s2", "unvouched@test.com", false); u.EmailVerified || a.verified(u.ID) {
		t.Fatal("an address the provider does not vouch for must stay unverified")
	}
	// Linking an existing account by a vouched address proves the address too.
	if u := provision("s3", "jane@test.com", true); u.ID != existing.ID || !a.verified(existing.ID) {
		t.Fatalf("link by email: %+v", u)
	}
}
