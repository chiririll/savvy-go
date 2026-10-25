//go:build integration

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/mail/smtp"
	"savvy-go/internal/settings"
)

// recorder is a mailer that keeps what it is asked to send.
type recorder struct {
	mu       sync.Mutex
	on       bool
	sent     []smtp.Message
	sendErr  error
	checkErr error
}

func (r *recorder) Enabled() bool { return r.on }

func (r *recorder) Send(_ context.Context, m smtp.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.on {
		return smtp.ErrNotConfigured
	}
	if r.sendErr != nil {
		return r.sendErr
	}
	r.sent = append(r.sent, m)
	return nil
}

func (r *recorder) Check(context.Context) error {
	if !r.on {
		return smtp.ErrNotConfigured
	}
	return r.checkErr
}

func (r *recorder) messages() []smtp.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]smtp.Message(nil), r.sent...)
}

// mailFile is a config file with the [mailer] section of newMailApp; host replaces the mail
// server's host.
func mailFile(on bool, host string) *config.Config {
	return testFile(fmt.Sprintf(`[mailer]
enabled = %t
host = %q
port = 587
protocol = "starttls"
user = "savvy"
password = "s3cret-smtp-password"
from = "Go Savvy <savvy@example.com>"
timeout = 5
`, on, host))
}

// newMailApp is newTestApp with mail configured (a recorder stands in for the
// SMTP server) and a signed-in server admin. opts come last, so they can replace the
// section.
func newMailApp(t *testing.T, rec *recorder, opts ...Option) (*testApp, *auth.Issued, *auth.User) {
	t.Helper()
	return newMailAppAt(t, testURL, rec, opts...)
}

// newMailAppAt is newMailApp with the given public URL; "" is a server that has none.
func newMailAppAt(t *testing.T, appURL string, rec *recorder, opts ...Option) (*testApp, *auth.Issued, *auth.User) {
	t.Helper()
	cfg, _ := testConfig(t)
	st := openTestStore(t, cfg)
	opts = append(
		[]Option{WithRates(currencydata.NewOffline()), WithConfig(mailFile(rec.on, "smtp.example.com")), WithMailer(rec)},
		opts...)
	s := New(cfg.WithAppURL(appURL), st, testKeys(t, cfg), opts...)
	space, err := servicesOf(s).Spaces.Create(context.Background(), "Test", nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	a := &testApp{t: t, db: st.SQL(), store: st, space: space, srv: ts, s: s, svc: servicesOf(s), client: ts.Client()}
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	return a, a.issue(admin, false), admin
}

func TestMailStatusAdminOnlyHidesPassword(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})

	res := a.do("GET", "/api/admin/mail", nil, sess.Token, sess.CSRF)
	raw := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("status %d %v", res.StatusCode, raw)
	}
	data := raw["data"].(map[string]any)
	if data["enabled"] != true || data["host"] != "smtp.example.com" || data["port"] != float64(587) ||
		data["protocol"] != "starttls" || data["user"] != "savvy" || data["from"] != "Go Savvy <savvy@example.com>" ||
		data["passwordSet"] != true || data["skipVerify"] != false {
		t.Errorf("status %v", data)
	}
	if strings.Contains(toString(raw), "s3cret-smtp-password") {
		t.Error("the SMTP password is in the response")
	}

	user := a.createUser("plain@test.com", "secret1", roleEditor)
	u := a.issue(user, false)
	for _, call := range [][2]string{{"GET", "/api/admin/mail"}, {"POST", "/api/admin/mail/test"}} {
		res := a.do(call[0], call[1], map[string]any{}, u.Token, u.CSRF)
		if res.StatusCode != 403 {
			t.Errorf("%s %s as a plain user: %d", call[0], call[1], res.StatusCode)
		}
		res.Body.Close()
	}
	res = a.do("GET", "/api/admin/mail", nil, "", "")
	if res.StatusCode != 401 {
		t.Errorf("signed out: %d", res.StatusCode)
	}
	res.Body.Close()
}

// newWritableMailApp is newMailApp over a config file on disk that the application can save,
// as in production once startup has finished.
func newWritableMailApp(t *testing.T, rec *recorder) (*testApp, *auth.Issued, *config.Config) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(fmt.Sprintf(`[mailer]
enabled = true
host = "smtp.example.com"
port = 587
protocol = "starttls"
user = "savvy"
password = "s3cret-smtp-password"
from = "Go Savvy <savvy@example.com>"
timeout = 5
`)), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := config.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a, sess, _ := newMailApp(t, rec, WithConfig(file))
	if err := file.Finish(); err != nil {
		t.Fatal(err)
	}
	return a, sess, file
}

func mailBody(over map[string]any) map[string]any {
	body := map[string]any{
		"enabled": true, "host": "mail.example.org", "port": 465, "protocol": "smtps",
		"user": "savvy", "from": "savvy@example.org", "skipVerify": false, "timeout": 8,
	}
	for k, v := range over {
		body[k] = v
	}
	return body
}

func TestMailSettingsAreSavedFromThePanel(t *testing.T) {
	a, sess, file := newWritableMailApp(t, &recorder{on: true})

	data := decodeJSON(t, a.do("GET", "/api/admin/mail", nil, sess.Token, sess.CSRF))["data"].(map[string]any)
	if data["writable"] != true {
		t.Fatalf("a config file that can be saved is reported read-only: %v", data)
	}

	res := a.do("PUT", "/api/admin/mail", mailBody(nil), sess.Token, sess.CSRF)
	raw := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("save %d %v", res.StatusCode, raw)
	}
	data = raw["data"].(map[string]any)
	if data["host"] != "mail.example.org" || data["port"] != float64(465) || data["protocol"] != "smtps" ||
		data["from"] != "savvy@example.org" || data["timeout"] != float64(8) || data["passwordSet"] != true {
		t.Errorf("response %v", data)
	}
	if strings.Contains(toString(raw), "s3cret-smtp-password") {
		t.Error("the SMTP password is in the response")
	}
	if conf, _ := a.svc.Mail.Config(); conf.Host != "mail.example.org" || conf.Password != "s3cret-smtp-password" {
		t.Errorf("not applied, or the stored password was lost: %+v", conf)
	}
	saved, err := os.ReadFile(file.Path())
	if err != nil || !strings.Contains(string(saved), "mail.example.org") {
		t.Errorf("not written to the config file: %v\n%s", err, saved)
	}

	res = a.do("PUT", "/api/admin/mail", mailBody(map[string]any{"user": "", "password": ""}), sess.Token, sess.CSRF)
	res.Body.Close()
	if conf, _ := a.svc.Mail.Config(); conf.Password != "" {
		t.Errorf("an empty password did not clear it: %q", conf.Password)
	}
}

func TestMailSettingsRejectedWhenInvalidOrNotAllowed(t *testing.T) {
	a, sess, _ := newWritableMailApp(t, &recorder{on: true})

	res := a.do("PUT", "/api/admin/mail", mailBody(map[string]any{"host": ""}), sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 422 || !strings.Contains(toString(body["errors"]), "host") {
		t.Fatalf("empty host: %d %v", res.StatusCode, body)
	}
	if conf, _ := a.svc.Mail.Config(); conf.Host != "smtp.example.com" {
		t.Errorf("a rejected save changed the mailer: %q", conf.Host)
	}

	user := a.createUser("plain@test.com", "secret1", roleEditor)
	u := a.issue(user, false)
	res = a.do("PUT", "/api/admin/mail", mailBody(nil), u.Token, u.CSRF)
	if res.StatusCode != 403 {
		t.Errorf("plain user: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestMailSettingsRefusedOnReadOnlyFile(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	res := a.do("PUT", "/api/admin/mail", mailBody(nil), sess.Token, sess.CSRF)
	if res.StatusCode != 409 {
		t.Fatalf("read-only file: %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
}

func TestMailStatusReportsAConfigProblem(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{}, WithConfig(mailFile(true, "")))
	data := decodeJSON(t, a.do("GET", "/api/admin/mail", nil, sess.Token, sess.CSRF))["data"].(map[string]any)
	if data["enabled"] != false || data["problem"] != "mailer.host is empty" {
		t.Errorf("status %v", data)
	}
}

func TestMailTestSendsAdminGivenAddress(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)

	res := a.do("POST", "/api/admin/mail/test", map[string]any{}, sess.Token, sess.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("test %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	res = a.do("POST", "/api/admin/mail/test", map[string]any{"to": "other@example.org"}, sess.Token, sess.CSRF)
	res.Body.Close()
	got := rec.messages()
	if len(got) != 2 || got[0].To != "admin@test.com" || got[1].To != "other@example.org" {
		t.Fatalf("sent %+v", got)
	}

	res = a.do("POST", "/api/admin/mail/test", map[string]any{"to": "not-an-address"}, sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Errorf("bad recipient: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestMailTestShowsWhyItFailed(t *testing.T) {
	rec := &recorder{on: true, checkErr: errors.New("535 authentication failed")}
	a, sess, _ := newMailApp(t, rec)
	res := a.do("POST", "/api/admin/mail/test", map[string]any{}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 422 || !strings.Contains(toString(body), "535 authentication failed") {
		t.Fatalf("connect failure: %d %v", res.StatusCode, body)
	}
	if len(rec.messages()) != 0 {
		t.Error("a message was sent although the connection check failed")
	}

	rec.checkErr, rec.sendErr = nil, errors.New("550 mailbox unavailable")
	res = a.do("POST", "/api/admin/mail/test", map[string]any{}, sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 422 || !strings.Contains(toString(body), "550 mailbox unavailable") {
		t.Fatalf("send failure: %d %v", res.StatusCode, body)
	}

	off := &recorder{}
	a, sess, _ = newMailApp(t, off)
	res = a.do("POST", "/api/admin/mail/test", map[string]any{}, sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Errorf("mail off: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestMailCheckLeavesSettingsUnsaved(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, file := newWritableMailApp(t, rec)
	before, _ := os.ReadFile(file.Path())

	res := a.do("POST", "/api/admin/mail/check", mailBody(nil), sess.Token, sess.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("check %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	if conf, _ := a.svc.Mail.Config(); conf.Host != "smtp.example.com" {
		t.Errorf("a check changed the mailer: %q", conf.Host)
	}
	if after, _ := os.ReadFile(file.Path()); string(after) != string(before) {
		t.Error("a check wrote the config file")
	}
	if len(rec.messages()) != 0 {
		t.Error("a check sent a message")
	}
}

func TestMailCheckReportsWhyItFailed(t *testing.T) {
	rec := &recorder{on: true, checkErr: errors.New("535 authentication failed")}
	a, sess, _ := newWritableMailApp(t, rec)

	res := a.do("POST", "/api/admin/mail/check", mailBody(nil), sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 422 || !strings.Contains(toString(body), "535 authentication failed") {
		t.Fatalf("connect failure: %d %v", res.StatusCode, body)
	}

	res = a.do("POST", "/api/admin/mail/check", mailBody(map[string]any{"host": ""}), sess.Token, sess.CSRF)
	body = decodeJSON(t, res)
	if res.StatusCode != 422 || !strings.Contains(toString(body["errors"]), "host") {
		t.Errorf("empty host: %d %v", res.StatusCode, body)
	}
}

func TestMailCheckIsForAdminsOnly(t *testing.T) {
	a, _, _ := newWritableMailApp(t, &recorder{on: true})
	user := a.createUser("plain@test.com", "secret1", roleEditor)
	u := a.issue(user, false)
	res := a.do("POST", "/api/admin/mail/check", mailBody(nil), u.Token, u.CSRF)
	if res.StatusCode != 403 {
		t.Errorf("plain user: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestResetLinkEmailedWithoutRevealingToken(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)
	target := a.createUser("target@test.com", "oldpass12", roleEditor)

	res := a.do(
		"POST",
		"/api/users/"+itoa(target.ID)+"/password-token",
		map[string]any{"send": true},
		sess.Token,
		sess.CSRF,
	)
	body := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("send %d %v", res.StatusCode, body)
	}
	data := body["data"].(map[string]any)
	if _, leaked := data["token"]; leaked {
		t.Error("the token is in the response although the link was emailed")
	}
	if data["emailSent"] != true {
		t.Errorf("emailSent %v", data["emailSent"])
	}
	got := rec.messages()
	if len(got) != 1 || got[0].To != "target@test.com" {
		t.Fatalf("sent %+v", got)
	}
	const prefix = "http://localhost:8080/set-password/"
	i := strings.Index(got[0].Text, prefix)
	if i < 0 {
		t.Fatalf("no reset link in:\n%s", got[0].Text)
	}
	token := strings.Fields(got[0].Text[i+len(prefix):])[0]
	res = a.do("GET", "/api/auth/password/"+token, nil, "", "")
	prev := decodeJSON(t, res)
	if res.StatusCode != 200 || prev["email"] != "target@test.com" {
		t.Errorf("emailed token does not work: %d %v", res.StatusCode, prev)
	}
}

// sendResetLink asks for the reset link of a user to be emailed.
func sendResetLink(t *testing.T, a *testApp, sess *auth.Issued, userID int64) (int, map[string]any) {
	t.Helper()
	res := a.do("POST", "/api/users/"+itoa(userID)+"/password-token", map[string]any{"send": true}, sess.Token, sess.CSRF)
	return res.StatusCode, decodeJSON(t, res)
}

func TestResetLinkNotSentIfUnusable(t *testing.T) {
	cases := []struct {
		name string
		// setup prepares the app and returns the user to send to.
		setup func(t *testing.T, rec *recorder) (*testApp, *auth.Issued, int64)
		// say is a word the refusal must contain, so the admin learns why.
		say string
	}{
		{"own account", func(t *testing.T, rec *recorder) (*testApp, *auth.Issued, int64) {
			a, sess, admin := newMailApp(t, rec)
			return a, sess, admin.ID
		}, ""},
		{"no public URL, so the link would point nowhere", func(t *testing.T, rec *recorder) (*testApp, *auth.Issued, int64) {
			a, sess, _ := newMailAppAt(t, "", rec)
			return a, sess, a.createUser("target@test.com", "oldpass12", roleEditor).ID
		}, "public URL"},
		{"password sign-in is off in favour of SSO", func(t *testing.T, rec *recorder) (*testApp, *auth.Issued, int64) {
			a, sess, _ := newMailApp(t, rec)
			insertSSO(t, a.db, true)
			if err := (settings.Store{DB: a.db}).Set(context.Background(), "password_login_enabled", false); err != nil {
				t.Fatal(err)
			}
			return a, sess, a.createUser("target@test.com", "oldpass12", roleEditor).ID
		}, ""},
		{"SMTP is down", func(t *testing.T, rec *recorder) (*testApp, *auth.Issued, int64) {
			rec.sendErr = errors.New("connection refused")
			a, sess, _ := newMailApp(t, rec)
			return a, sess, a.createUser("target@test.com", "oldpass12", roleEditor).ID
		}, "connection refused"},
		{
			"mail is off: asking to send is an error, not a silent fallback",
			func(t *testing.T, rec *recorder) (*testApp, *auth.Issued, int64) {
				rec.on = false
				a, sess, _ := newMailApp(t, rec)
				return a, sess, a.createUser("target@test.com", "oldpass12", roleEditor).ID
			},
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := &recorder{on: true}
			a, sess, id := c.setup(t, rec)
			code, body := sendResetLink(t, a, sess, id)
			if code != 422 || !strings.Contains(toString(body), c.say) {
				t.Errorf("%d %v, want 422 mentioning %q", code, body, c.say)
			}
			if n := len(rec.messages()); n != 0 {
				t.Errorf("%d emails recorded", n)
			}
		})
	}
}

// Without "send" the admin still gets the link to hand over, mail or not.
func TestResetLinkReturnsTokenUnsent(t *testing.T) {
	for _, on := range []bool{true, false} {
		rec := &recorder{on: on}
		a, sess, _ := newMailApp(t, rec)
		target := a.createUser("target@test.com", "oldpass12", roleEditor)
		res := a.do("POST", "/api/users/"+itoa(target.ID)+"/password-token", nil, sess.Token, sess.CSRF)
		data := decodeJSON(t, res)["data"].(map[string]any)
		if res.StatusCode != 200 || data["token"] == nil || data["token"] == "" {
			t.Errorf("mail=%v: %d %v", on, res.StatusCode, data)
		}
		if n := len(rec.messages()); n != 0 {
			t.Errorf("mail=%v: %d emails sent without being asked", on, n)
		}
	}
}

func TestNewUserGetsSetPasswordLink(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)

	res := a.do(
		"POST",
		"/api/users",
		map[string]any{"name": "New", "email": "new@test.com", "role": "user", "send": true},
		sess.Token,
		sess.CSRF,
	)
	body := decodeJSON(t, res)
	data, _ := body["data"].(map[string]any)
	if res.StatusCode != 201 || data["emailSent"] != true {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	if _, leaked := data["token"]; leaked {
		t.Error("the token is in the response although the link was emailed")
	}
	got := rec.messages()
	if len(got) != 1 || got[0].To != "new@test.com" || !strings.Contains(got[0].Text, "/set-password/") {
		t.Fatalf("sent %+v", got)
	}
}

func TestNewUserEmailNeedsMail(t *testing.T) {
	create := func(a *testApp, sess *auth.Issued, email string, extra map[string]any) (int, map[string]any) {
		in := map[string]any{"name": "New", "email": email, "role": "user", "send": true}
		for k, v := range extra {
			in[k] = v
		}
		res := a.do("POST", "/api/users", in, sess.Token, sess.CSRF)
		return res.StatusCode, decodeJSON(t, res)
	}

	t.Run("a password and a link are exclusive", func(t *testing.T) {
		rec := &recorder{on: true}
		a, sess, _ := newMailApp(t, rec)
		if code, body := create(a, sess, "pw@test.com", map[string]any{"password": "longenough1"}); code != 422 {
			t.Errorf("send with a password: %d %v", code, body)
		}
		if n := len(rec.messages()); n != 0 {
			t.Errorf("%d emails sent", n)
		}
	})

	// The user exists, nobody got the link, and the response says so.
	t.Run("SMTP down", func(t *testing.T) {
		a, sess, _ := newMailApp(t, &recorder{on: true, sendErr: errors.New("connection refused")})
		code, body := create(a, sess, "down@test.com", nil)
		data, _ := body["data"].(map[string]any)
		if code != 201 || data["emailError"] == nil || data["token"] != nil || data["emailSent"] == true {
			t.Errorf("smtp down: %d %v", code, body)
		}
	})

	t.Run("mail off", func(t *testing.T) {
		a, sess, _ := newMailApp(t, &recorder{})
		if code, body := create(a, sess, "off@test.com", nil); code != 422 {
			t.Errorf("mail off: %d %v", code, body)
		}
	})
}

// With verification required a link handed over by the admin proves nothing
// about the mailbox, so a user without a password has to be emailed.
func TestNewUserLinkRefusedWhenVerifying(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, admin := newMailApp(t, rec)
	_ = a.svc.Users.SetEmailVerified(context.Background(), admin.ID, true)
	_ = settings.Store{DB: a.db}.Set(context.Background(), "require_email_verification", true)

	res := a.do(
		"POST",
		"/api/users",
		map[string]any{"name": "New", "email": "new@test.com", "role": "user"},
		sess.Token,
		sess.CSRF,
	)
	if res.StatusCode != 422 {
		t.Errorf("link instead of email: %d", res.StatusCode)
	}
	res = a.do(
		"POST",
		"/api/users",
		map[string]any{"name": "New", "email": "new@test.com", "role": "user", "send": true},
		sess.Token,
		sess.CSRF,
	)
	if res.StatusCode != 201 {
		t.Errorf("emailed: %d", res.StatusCode)
	}
}
