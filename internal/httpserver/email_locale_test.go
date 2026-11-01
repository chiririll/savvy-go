//go:build integration

package httpserver

import (
	"strings"
	"testing"
)

// The language of an email is the one the sender picked for the invitee, sent as `locale`.

func TestNewUserEmailIsInTheChosenLanguage(t *testing.T) {
	for _, c := range []struct{ locale, want string }{
		{"ru", "Здравствуйте"},
		{"ru-RU", "Здравствуйте"},
		{"en", "Hello"},
		{"", "Hello"},
		{"xx", "Hello"},
	} {
		rec := &recorder{on: true}
		a, sess, _ := newMailApp(t, rec)
		body := map[string]any{"name": "New", "email": "new@test.com", "role": "user", "send": true}
		if c.locale != "" {
			body["locale"] = c.locale
		}
		if res := a.do("POST", "/api/users", body, sess.Token, sess.CSRF); res.StatusCode != 201 {
			t.Fatalf("%q: create %d %v", c.locale, res.StatusCode, decodeJSON(t, res))
		}
		got := rec.messages()
		if len(got) != 1 || !strings.Contains(got[0].Text, c.want) {
			t.Errorf("locale %q: want %q in %+v", c.locale, c.want, got)
		}
	}
}

func TestResetLinkEmailIsInTheChosenLanguage(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)
	target := a.createUser("target@test.com", "oldpass12", roleEditor)

	res := a.do("POST", "/api/users/"+itoa(target.ID)+"/password-token",
		map[string]any{"send": true, "locale": "ru"}, sess.Token, sess.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("send %d %v", res.StatusCode, decodeJSON(t, res))
	}
	if got := rec.messages(); len(got) != 1 || !strings.Contains(got[0].Text, "Здравствуйте") {
		t.Errorf("sent %+v", got)
	}
}

func TestSpaceInvitationEmailIsInTheChosenLanguage(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)

	res := a.do("POST", a.invitePath(a.space.ID),
		map[string]any{"role": "editor", "email": "new@example.org", "locale": "ru"}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		t.Fatalf("invite %d %v", res.StatusCode, decodeJSON(t, res))
	}
	got := rec.messages()
	if len(got) != 1 || !strings.Contains(got[0].Text, "приглашает") || !strings.Contains(got[0].Text, "редактор") {
		t.Errorf("sent %+v", got)
	}
	if !strings.Contains(got[0].HTML, `lang="ru"`) {
		t.Errorf("no html part in russian: %q", got[0].HTML)
	}
}

func TestVerificationEmailFollowsInterfaceLanguage(t *testing.T) {
	rec := &recorder{on: true}
	a, _, _ := newMailApp(t, rec)
	sess := a.issue(a.createUser("u@test.com", "secret1", roleEditor), false)
	req := a.request("POST", "/api/auth/verify-email/resend", nil, sess.Token, sess.CSRF)
	req.Header.Set("X-Locale", "ru")
	res, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("resend %d %v", res.StatusCode, decodeJSON(t, res))
	}
	if got := rec.messages(); len(got) != 1 || !strings.Contains(got[0].Text, "Здравствуйте") {
		t.Errorf("sent %+v", got)
	}
}
