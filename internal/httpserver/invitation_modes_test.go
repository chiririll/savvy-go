package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/settings"
)

// invitePath is the invitations collection of the app's space.
func (a *testApp) invitePath(spaceID int64) string {
	return "/api/spaces/" + itoa(spaceID) + "/invitations"
}

// postInvite creates an invitation and returns the status and the data object.
func (a *testApp) postInvite(sess *auth.Issued, spaceID int64, email string) (int, map[string]any, map[string]any) {
	a.t.Helper()
	body := map[string]any{"role": "editor"}
	if email != "" {
		body["email"] = email
	}
	res := a.do("POST", a.invitePath(spaceID), body, sess.Token, sess.CSRF)
	raw := decodeJSON(a.t, res)
	data, _ := raw["data"].(map[string]any)
	return res.StatusCode, data, raw
}

// requireVerification switches the instance requirement on, as an admin who
// has confirmed their own address.
func (a *testApp) requireVerification(admin *auth.User) {
	a.t.Helper()
	if err := a.svc.Users.SetEmailVerified(context.Background(), admin.ID, true); err != nil {
		a.t.Fatal(err)
	}
	if err := a.svc.Settings.Set(context.Background(), "require_email_verification", true); err != nil {
		a.t.Fatal(err)
	}
}

func (a *testApp) invitationRows() int {
	a.t.Helper()
	var n int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM space_invitations`).Scan(&n); err != nil {
		a.t.Fatal(err)
	}
	return n
}

func (a *testApp) secondSpace() int64 {
	a.t.Helper()
	admin, err := a.svc.Users.ByEmail(context.Background(), "admin@test.com")
	if err != nil || admin == nil {
		a.t.Fatalf("admin: %v", err)
	}
	sp, err := a.svc.Spaces.Create(context.Background(), "Other", admin)
	if err != nil {
		a.t.Fatal(err)
	}
	return sp.ID
}

func TestInvitationMethodFollowsTheServerSetup(t *testing.T) {
	t.Run("mail on, verification off: the inviter picks", func(t *testing.T) {
		rec := &recorder{on: true}
		a, sess, _ := newMailApp(t, rec)

		status, data, raw := a.postInvite(sess, a.space.ID, "new@example.org")
		if status != 201 || data["emailSent"] != true {
			t.Fatalf("email invitation: %d %v", status, raw)
		}
		if _, has := data["token"]; has {
			t.Fatalf("the link of an emailed invitation was handed to the inviter: %v", data)
		}
		link := linkIn(t, rec.messages()[0])
		if !strings.HasPrefix(link.Path, "/invite/") || link.RawQuery != "" {
			t.Errorf("emailed link %s: want /invite/<token> with no query", link)
		}

		status, data, raw = a.postInvite(sess, a.space.ID, "")
		if status != 201 || data["token"] == nil || data["token"] == "" {
			t.Fatalf("link invitation: %d %v", status, raw)
		}
		if n := len(rec.messages()); n != 1 {
			t.Errorf("a link invitation sent mail (%d messages)", n)
		}
	})

	t.Run("verification required: email only", func(t *testing.T) {
		rec := &recorder{on: true}
		a, sess, admin := newMailApp(t, rec)
		a.requireVerification(admin)

		if status, _, raw := a.postInvite(sess, a.space.ID, ""); status != 422 {
			t.Errorf("link invitation with verification required: %d %v", status, raw)
		}
		if status, data, raw := a.postInvite(sess, a.space.ID, "new@example.org"); status != 201 ||
			data["emailSent"] != true {
			t.Errorf("email invitation: %d %v", status, raw)
		}
	})

	t.Run("no mail: link only", func(t *testing.T) {
		a, sess, _ := newMailApp(t, &recorder{})
		if status, _, raw := a.postInvite(sess, a.space.ID, "new@example.org"); status != 422 {
			t.Errorf("email invitation without mail: %d %v", status, raw)
		}
		if status, data, raw := a.postInvite(sess, a.space.ID, ""); status != 201 || data["token"] == "" {
			t.Errorf("link invitation: %d %v", status, raw)
		}
		if a.invitationRows() != 1 {
			t.Errorf("the refused email invitation left a row behind")
		}
	})
}

func TestInvitationMethodsTellTheFrontendWhatApplies(t *testing.T) {
	cases := []struct {
		name         string
		appURL       string
		mail, verify bool
		email, link  bool
		reason       string // what the reason for "no email" must contain; "" when email works
	}{
		{"mail and choice", testURL, true, false, true, true, ""},
		{"verification required", testURL, true, true, true, false, ""},
		{"no mail", testURL, false, false, false, true, "mail"},
		{"mail but no public URL", "", true, false, false, true, "public URL"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, sess, admin := newMailAppAt(t, c.appURL, &recorder{on: c.mail})
			if c.verify {
				a.requireVerification(admin)
			}
			res := a.do("GET", "/api/invitation-methods", nil, sess.Token, sess.CSRF)
			data, _ := decodeJSON(t, res)["data"].(map[string]any)
			if data["email"] != c.email || data["link"] != c.link {
				t.Errorf("methods %v, want email=%v link=%v", data, c.email, c.link)
			}
			reason, _ := data["emailReason"].(string)
			if c.email && reason != "" || !c.email && !strings.Contains(strings.ToLower(reason), strings.ToLower(c.reason)) {
				t.Errorf("emailReason = %q, want one mentioning %q", reason, c.reason)
			}
		})
	}
}

func TestRegisteringThroughAnEmailedInvitationVerifiesTheAddress(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, admin := newMailApp(t, rec)
	a.requireVerification(admin)

	register := func(token, email string) *http.Response {
		return a.do("POST", "/api/invitations/"+token+"/register",
			map[string]any{"name": "New", "email": email, "password": "secret12"}, "", "")
	}
	a.postInvite(sess, a.space.ID, "bob@example.org")
	token := strings.TrimPrefix(linkIn(t, rec.messages()[0]).Path, "/invite/")

	// The invitation names an address; the link cannot be used for another.
	status(t, register(token, "mallory@example.org"), 422, "another address")

	res := register(token, "bob@example.org")
	if res.StatusCode != 201 {
		t.Fatalf("register: %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	u, _ := a.svc.Users.ByEmail(context.Background(), "bob@example.org")
	if u == nil || !a.verified(u.ID) {
		t.Fatal("registering from the emailed link left the address unverified")
	}
	if n := len(rec.messages()); n != 1 {
		t.Errorf("a verified address was sent another email (%d messages)", n)
	}
}

func TestLinkInvitationKeepsNoteEmailedDoesNot(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)
	post := func(body map[string]any) (int, map[string]any) {
		res := a.do("POST", a.invitePath(a.space.ID), body, sess.Token, sess.CSRF)
		return res.StatusCode, decodeJSON(t, res)
	}

	if status, raw := post(map[string]any{"role": "editor", "note": "  Kim from work "}); status != 201 {
		t.Fatalf("link with note: %d %v", status, raw)
	}
	if status, raw := post(map[string]any{"role": "editor", "email": "new@example.org", "note": "ignored"}); status != 201 {
		t.Fatalf("email with note: %d %v", status, raw)
	}
	if status, _ := post(map[string]any{"role": "editor", "note": strings.Repeat("x", 101)}); status != 422 {
		t.Fatalf("a note over the limit = %d, want 422", status)
	}

	res := a.do("GET", a.invitePath(a.space.ID), nil, sess.Token, sess.CSRF)
	list, _ := decodeJSON(t, res)["data"].([]any)
	if len(list) != 2 {
		t.Fatalf("invitations = %v", list)
	}
	// Newest first: the emailed one, then the link.
	if emailed := list[0].(map[string]any); emailed["note"] != nil {
		t.Fatalf("an emailed invitation has a note: %v", emailed)
	}
	if link := list[1].(map[string]any); link["note"] != "Kim from work" {
		t.Fatalf("link note = %v", link["note"])
	}
}

// A link nobody mailed proves nothing about the address typed at registration.
// If the instance wants verified addresses by then, the new account is mailed a
// confirmation link; if not, nothing is sent.
func TestLinkInvitationRegistrationAsksEmailConfirmation(t *testing.T) {
	register := func(a *testApp, token, email string) {
		t.Helper()
		res := a.do("POST", "/api/invitations/"+token+"/register",
			map[string]any{"name": "New", "email": email, "password": "secret12"}, "", "")
		if res.StatusCode != 201 {
			t.Fatalf("register: %d %v", res.StatusCode, decodeJSON(t, res))
		}
		res.Body.Close()
	}

	t.Run("verification required later", func(t *testing.T) {
		rec := &recorder{on: true}
		a, sess, admin := newMailApp(t, rec)
		_, data, _ := a.postInvite(sess, a.space.ID, "")
		a.requireVerification(admin) // the link was made while it was not required

		register(a, data["token"].(string), "carol@example.org")
		u, _ := a.svc.Users.ByEmail(context.Background(), "carol@example.org")
		if u == nil || a.verified(u.ID) {
			t.Fatalf("a link nobody mailed proved an address: %+v", u)
		}
		msgs := rec.messages()
		if len(msgs) != 1 || msgs[0].To != "carol@example.org" ||
			!strings.HasPrefix(linkIn(t, msgs[0]).Path, "/verify-email/") {
			t.Fatalf("want one confirmation email to carol, got %+v", msgs)
		}
	})

	t.Run("verification not required", func(t *testing.T) {
		rec := &recorder{on: true}
		a, sess, _ := newMailApp(t, rec)
		_, data, _ := a.postInvite(sess, a.space.ID, "")

		register(a, data["token"].(string), "dan@example.org")
		if n := len(rec.messages()); n != 0 {
			t.Errorf("%d emails sent although verification is not required", n)
		}
	})
}

func TestInvitationRegistrationIsClosedWhenPasswordSignInIsOff(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{})
	_, data, _ := a.postInvite(sess, a.space.ID, "")
	insertSSO(t, a.db, true)
	if err := (settings.Store{DB: a.db}).Set(context.Background(), "password_login_enabled", false); err != nil {
		t.Fatal(err)
	}

	res := a.do("POST", "/api/invitations/"+data["token"].(string)+"/register",
		map[string]any{"name": "New", "email": "eve@example.org", "password": "secret12"}, "", "")
	status(t, res, http.StatusForbidden, "register without password sign-in")
	if u, _ := a.svc.Users.ByEmail(context.Background(), "eve@example.org"); u != nil {
		t.Fatal("an account was created")
	}
}

func TestAFailedSendLeavesNoInvitationAndNoCooldown(t *testing.T) {
	rec := &recorder{on: true, sendErr: errors.New("connection refused")}
	a, sess, _ := newMailApp(t, rec)

	status, _, raw := a.postInvite(sess, a.space.ID, "new@example.org")
	if status < 400 || !strings.Contains(toString(raw), "connection refused") {
		t.Fatalf("failed send: %d %v", status, raw)
	}
	if a.invitationRows() != 0 {
		t.Fatal("an invitation nobody received was kept")
	}

	rec.mu.Lock()
	rec.sendErr = nil
	rec.mu.Unlock()
	if status, _, raw := a.postInvite(sess, a.space.ID, "new@example.org"); status != 201 {
		t.Errorf("retry after a failed send: %d %v", status, raw)
	}
}

func TestAnAddressCannotBeInvitedTwiceToTheSameSpace(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	if status, _, raw := a.postInvite(sess, a.space.ID, "new@example.org"); status != 201 {
		t.Fatalf("first: %d %v", status, raw)
	}
	status, _, raw := a.postInvite(sess, a.space.ID, "NEW@example.org")
	if status != 422 || !strings.Contains(strings.ToLower(toString(raw)), "already invited") {
		t.Errorf("second invitation: %d %v", status, raw)
	}
	if a.invitationRows() != 1 {
		t.Errorf("rows %d, want 1", a.invitationRows())
	}

	// A pending invitation to another space is no obstacle.
	if status, _, raw := a.postInvite(sess, a.secondSpace(), "new@example.org"); status != 201 {
		t.Errorf("other space: %d %v", status, raw)
	}
}

func TestAMemberCannotBeInvitedByEmail(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	a.createUser("member@test.com", "secret1", roleEditor)
	status, _, raw := a.postInvite(sess, a.space.ID, "member@test.com")
	if status != 422 || !strings.Contains(strings.ToLower(toString(raw)), "member") {
		t.Errorf("inviting a member: %d %v", status, raw)
	}
}

func TestRevokingAnInvitationKillsTheLinkButNotTheCooldown(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)
	_, data, _ := a.postInvite(sess, a.space.ID, "new@example.org")
	token := strings.TrimPrefix(linkIn(t, rec.messages()[0]).Path, "/invite/")
	id := int64(data["id"].(float64))

	status(t, a.do("DELETE", a.invitePath(a.space.ID)+"/"+itoa(id), nil, sess.Token, sess.CSRF), 204, "revoke")
	status(t, a.do("GET", "/api/invitations/"+token, nil, "", ""), 404, "revoked link")

	list := decodeJSON(t, a.do("GET", a.invitePath(a.space.ID), nil, sess.Token, sess.CSRF))["data"].([]any)
	if len(list) != 0 {
		t.Errorf("a revoked invitation is still listed: %v", list)
	}
	if a.invitationRows() != 1 {
		t.Fatal("the revoked row is the cooldown's record and must stay")
	}

	// Revoke, invite, revoke, invite is how an admin would flood a mailbox.
	before := len(rec.messages())
	for _, space := range []int64{a.space.ID, a.secondSpace()} {
		status, _, raw := a.postInvite(sess, space, "new@example.org")
		if status != http.StatusTooManyRequests {
			t.Errorf("space %d right after revoking: %d %v", space, status, raw)
		}
	}
	if len(rec.messages()) != before {
		t.Error("a mail went out during the cooldown")
	}

	// Revoking it again must not erase the record either.
	status(t, a.do("DELETE", a.invitePath(a.space.ID)+"/"+itoa(id), nil, sess.Token, sess.CSRF), 404, "revoke twice")
	if a.invitationRows() != 1 {
		t.Fatal("revoking twice deleted the cooldown record")
	}
}

func TestPruningEndsTheCooldownOfOldRowsOnly(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	ctx := context.Background()
	now := time.Now().UTC()
	stamp := func(d time.Duration) string { return now.Add(d).Format("2006-01-02T15:04:05Z") }

	for _, email := range []string{"old@example.org", "recent@example.org", "pending@example.org"} {
		if status, _, raw := a.postInvite(sess, a.space.ID, email); status != 201 {
			t.Fatalf("%s: %d %v", email, status, raw)
		}
	}
	set := func(email, at string) {
		if _, err := a.db.Exec(`UPDATE space_invitations SET expires_at = ? WHERE email = ?`, at, email); err != nil {
			t.Fatal(err)
		}
	}
	set("old@example.org", stamp(-2*time.Hour))
	set("recent@example.org", stamp(-10*time.Minute))
	// An accepted invitation is history and stays however old.
	if _, err := a.db.Exec(`INSERT INTO space_invitations (space_id, role, token_hash, expires_at, accepted_at)
		VALUES (?, 'viewer', 'accepted-hash', ?, ?)`, a.space.ID, stamp(-48*time.Hour), stamp(-72*time.Hour)); err != nil {
		t.Fatal(err)
	}

	n, err := a.svc.Spaces.PruneInvitations(ctx)
	if err != nil || n != 1 {
		t.Fatalf("pruned %d, %v; want only the row that expired two hours ago", n, err)
	}
	if status, _, raw := a.postInvite(sess, a.space.ID, "old@example.org"); status != 201 {
		t.Errorf("after pruning: %d %v", status, raw)
	}
	if status, _, _ := a.postInvite(sess, a.space.ID, "recent@example.org"); status != http.StatusTooManyRequests {
		t.Errorf("recent revocation: %d", status)
	}
	if status, _, _ := a.postInvite(sess, a.space.ID, "pending@example.org"); status != 422 {
		t.Errorf("pending invitation: %d", status)
	}
}

func TestOnlyAServerAdminClearsInvitationCooldowns(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	for _, email := range []string{"a@example.org", "b@example.org"} {
		_, data, _ := a.postInvite(sess, a.space.ID, email)
		status(
			t,
			a.do("DELETE", a.invitePath(a.space.ID)+"/"+itoa(int64(data["id"].(float64))), nil, sess.Token, sess.CSRF),
			204,
			"revoke",
		)
	}
	a.postInvite(sess, a.space.ID, "c@example.org")

	spaceAdmin := a.createUser("spaceadmin@test.com", "secret1", roleEditor)
	u := a.issue(spaceAdmin, false)
	status(t, a.do("POST", "/api/admin/invitations/clear", map[string]any{}, u.Token, u.CSRF), 403, "plain user")
	status(t, a.do("POST", "/api/admin/invitations/clear", map[string]any{}, "", ""), 401, "signed out")

	// One address first.
	res := a.do("POST", "/api/admin/invitations/clear", map[string]any{"email": "A@example.org"}, sess.Token, sess.CSRF)
	data := decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 200 || data["cleared"] != float64(1) {
		t.Fatalf("clear one: %d %v", res.StatusCode, data)
	}
	if status, _, raw := a.postInvite(sess, a.space.ID, "a@example.org"); status != 201 {
		t.Errorf("cleared address: %d %v", status, raw)
	}
	if status, _, _ := a.postInvite(sess, a.space.ID, "b@example.org"); status != http.StatusTooManyRequests {
		t.Errorf("other address stays blocked: %d", status)
	}

	// Then everything: the pending invitation is not a cooldown and stays.
	res = a.do("POST", "/api/admin/invitations/clear", map[string]any{}, sess.Token, sess.CSRF)
	data = decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 200 || data["cleared"] != float64(1) {
		t.Fatalf("clear all: %d %v", res.StatusCode, data)
	}
	if status, _, _ := a.postInvite(sess, a.space.ID, "c@example.org"); status != 422 {
		t.Errorf("pending invitation survived clearing: %d", status)
	}

	var audited int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM admin_audit WHERE action = 'invitations.cleared'`).Scan(&audited); err != nil ||
		audited != 2 {
		t.Errorf("audit rows %d, %v; want one per clearing", audited, err)
	}
}

func TestAcceptedInvitationsCanStillBeRemovedFromTheList(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{})
	_, data, _ := a.postInvite(sess, a.space.ID, "")
	token := data["token"].(string)
	id := int64(data["id"].(float64))

	bob := a.createUser("bob@test.com", "secret1", roleEditor)
	b := a.issue(bob, false)
	status(t, a.do("POST", "/api/invitations/"+token+"/accept", nil, b.Token, b.CSRF), 200, "accept")

	list := decodeJSON(t, a.do("GET", a.invitePath(a.space.ID), nil, sess.Token, sess.CSRF))["data"].([]any)
	if len(list) != 1 {
		t.Fatalf("accepted invitation not listed: %v", list)
	}
	status(t, a.do("DELETE", a.invitePath(a.space.ID)+"/"+itoa(id), nil, sess.Token, sess.CSRF), 204, "remove accepted")
	if a.invitationRows() != 0 {
		t.Error("the accepted row should be deleted, not kept as a cooldown")
	}
}
