package httpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/mailer"
	"savvy-go/internal/settings"
)

// recorder is a mailer that keeps what it is asked to send.
type recorder struct {
	mu       sync.Mutex
	on       bool
	sent     []mailer.Message
	sendErr  error
	checkErr error
}

func (r *recorder) Enabled() bool { return r.on }

func (r *recorder) Send(_ context.Context, m mailer.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.on {
		return mailer.ErrNotConfigured
	}
	if r.sendErr != nil {
		return r.sendErr
	}
	r.sent = append(r.sent, m)
	return nil
}

func (r *recorder) Check(context.Context) error {
	if !r.on {
		return mailer.ErrNotConfigured
	}
	return r.checkErr
}

func (r *recorder) messages() []mailer.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]mailer.Message(nil), r.sent...)
}

// newMailApp is newTestApp with mail configured (a recorder stands in for the
// SMTP server) and a signed-in server admin.
func newMailApp(t *testing.T, rec *recorder) (*testApp, *auth.Issued, *auth.User) {
	t.Helper()
	cfg, dir := testConfig(t)
	cfg.Mail = mailer.Config{
		Enabled: rec.on, Host: "smtp.example.com", Port: 587, Protocol: mailer.STARTTLS,
		User: "savvy", Password: "s3cret-smtp-password", From: "Go Savvy <savvy@example.com>",
		Timeout: 5 * time.Second,
	}
	// A real config file, so the admin panel can save the mail section into it.
	loaded, err := config.Load(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := loaded.Live.SetAppURL(cfg.AppURL); err != nil {
		t.Fatal(err)
	}
	cfg.Live = loaded.Live
	st := openTestStore(t, cfg)
	s := New(cfg, st, testKeys(t, cfg), WithRates(currencydata.NewOffline()), WithMailer(rec))
	space, err := s.Spaces.Create(context.Background(), "Test", nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	a := &testApp{t: t, db: st.SQL(), store: st, space: space, srv: ts, s: s, client: ts.Client()}
	admin := a.createUser("admin@test.com", "secret1", auth.RoleAdmin)
	return a, a.issue(admin, false), admin
}

func TestMailStatusIsAdminOnlyAndHidesThePassword(t *testing.T) {
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

func TestMailStatusReportsAConfigProblem(t *testing.T) {
	rec := &recorder{}
	a, sess, _ := newMailApp(t, rec)
	a.s.Mail.SetProblem("mailer.host is empty")
	data := decodeJSON(t, a.do("GET", "/api/admin/mail", nil, sess.Token, sess.CSRF))["data"].(map[string]any)
	if data["enabled"] != false || data["problem"] != "mailer.host is empty" {
		t.Errorf("status %v", data)
	}
}

// mailForm is a complete, valid body for PUT /api/admin/mail.
func mailForm(over map[string]any) map[string]any {
	form := map[string]any{
		"enabled": true, "host": "smtp.new.example", "port": 465, "protocol": "smtps",
		"user": "bot", "password": "new-password", "from": "Savvy <bot@new.example>",
		"skipVerify": true, "timeout": 7,
	}
	for k, v := range over {
		form[k] = v
	}
	return form
}

func TestMailSettingsAreAdminOnly(t *testing.T) {
	a, _, _ := newMailApp(t, &recorder{on: true})
	user := a.createUser("plain@test.com", "secret1", roleEditor)
	u := a.issue(user, false)
	res := a.do("PUT", "/api/admin/mail", mailForm(nil), u.Token, u.CSRF)
	if res.StatusCode != 403 {
		t.Errorf("plain user: %d", res.StatusCode)
	}
	res.Body.Close()
	res = a.do("PUT", "/api/admin/mail", mailForm(nil), "", "")
	if res.StatusCode != 401 {
		t.Errorf("signed out: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestMailSettingsSaveApplyAndPersist(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{})
	next := &recorder{on: true}
	var built mailer.Config
	a.s.Mail.NewMailer = func(c mailer.Config) mailer.Mailer { built = c; return next }

	res := a.do("PUT", "/api/admin/mail", mailForm(nil), sess.Token, sess.CSRF)
	raw := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("save %d %v", res.StatusCode, raw)
	}
	data := raw["data"].(map[string]any)
	if data["enabled"] != true || data["host"] != "smtp.new.example" || data["port"] != float64(465) ||
		data["protocol"] != "smtps" || data["user"] != "bot" || data["from"] != "Savvy <bot@new.example>" ||
		data["skipVerify"] != true || data["timeout"] != float64(7) || data["passwordSet"] != true {
		t.Errorf("response %v", data)
	}
	if strings.Contains(toString(raw), "new-password") {
		t.Error("the SMTP password is in the response")
	}

	// It is live at once: the new mailer is built from the saved values and used.
	if built.Host != "smtp.new.example" || built.Password != "new-password" || built.Timeout != 7*time.Second {
		t.Errorf("mailer built from %+v", built)
	}
	res = a.do("POST", "/api/admin/mail/test", map[string]any{}, sess.Token, sess.CSRF)
	res.Body.Close()
	if len(next.messages()) != 1 {
		t.Errorf("test email went through the old mailer")
	}

	// And it is on disk, so a restart keeps it.
	again, err := config.Load(filepath.Join(a.s.Cfg.DataDir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	want := mailer.Config{
		Enabled: true, Host: "smtp.new.example", Port: 465, Protocol: mailer.SMTPS, User: "bot",
		Password: "new-password", From: "Savvy <bot@new.example>", SkipVerify: true, Timeout: 7 * time.Second,
	}
	if again.Mail != want {
		t.Errorf("file holds %+v", again.Mail)
	}
}

func TestMailSettingsKeepThePasswordWhenOmitted(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	var built mailer.Config
	a.s.Mail.NewMailer = func(c mailer.Config) mailer.Mailer { built = c; return &recorder{on: c.Enabled} }

	form := mailForm(nil)
	delete(form, "password")
	res := a.do("PUT", "/api/admin/mail", form, sess.Token, sess.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("save %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	if built.Password != "s3cret-smtp-password" {
		t.Errorf("password after an omitted field: %q", built.Password)
	}

	// An empty string clears it; with a user set that is not valid.
	res = a.do("PUT", "/api/admin/mail", mailForm(map[string]any{"password": ""}), sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Errorf("empty password with a user: %d", res.StatusCode)
	}
	res.Body.Close()
	res = a.do("PUT", "/api/admin/mail", mailForm(map[string]any{"password": "", "user": ""}), sess.Token, sess.CSRF)
	if res.StatusCode != 200 || built.Password != "" || built.User != "" {
		t.Errorf("relay without login: %d %+v", res.StatusCode, built)
	}
	res.Body.Close()
}

func TestMailSettingsRejectAMistakeAndChangeNothing(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	path := filepath.Join(a.s.Cfg.DataDir, "config.toml")
	before, _ := os.ReadFile(path)

	for name, over := range map[string]map[string]any{
		"host":     {"host": " "},
		"port":     {"port": 70000},
		"protocol": {"protocol": "ssl"},
		"from":     {"from": "nobody"},
	} {
		res := a.do("PUT", "/api/admin/mail", mailForm(over), sess.Token, sess.CSRF)
		body := decodeJSON(t, res)
		errs, _ := body["errors"].(map[string]any)
		if res.StatusCode != 422 || errs[name] == nil {
			t.Errorf("%s: %d %v", name, res.StatusCode, body)
		}
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("the config file changed although every save was refused")
	}
	data := decodeJSON(t, a.do("GET", "/api/admin/mail", nil, sess.Token, sess.CSRF))["data"].(map[string]any)
	if data["host"] != "smtp.example.com" || data["enabled"] != true {
		t.Errorf("state changed: %v", data)
	}
}

func TestMailSettingsCanSwitchMailOffAndKeepTheValues(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	// A disabled section need not be complete.
	res := a.do("PUT", "/api/admin/mail", mailForm(map[string]any{"enabled": false, "host": ""}), sess.Token, sess.CSRF)
	data := decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 200 || data["enabled"] != false || data["problem"] != nil {
		t.Fatalf("switch off: %d %v", res.StatusCode, data)
	}
	res = a.do("POST", "/api/admin/mail/test", map[string]any{}, sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Errorf("test with mail off: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestMailSettingsOnAReadOnlyConfig(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{on: true})
	// Same trick as the config tests: a directory where the temp file goes.
	if err := os.Mkdir(filepath.Join(a.s.Cfg.DataDir, "ro"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(a.s.Cfg.DataDir, "ro", "config.toml.tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(filepath.Join(a.s.Cfg.DataDir, "ro", "config.toml"))
	if err != nil || loaded.Live.Writable() {
		t.Fatalf("setup: %v writable=%v", err, loaded.Live.Writable())
	}
	a.s.Cfg.Live = loaded.Live
	a.s.Mail.UseConfig(a.s.Cfg)

	data := decodeJSON(t, a.do("GET", "/api/admin/mail", nil, sess.Token, sess.CSRF))["data"].(map[string]any)
	if data["writable"] != false {
		t.Errorf("writable %v", data["writable"])
	}
	res := a.do("PUT", "/api/admin/mail", mailForm(nil), sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Errorf("save on a read-only config: %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestMailTestSendsToTheAdminOrTheGivenAddress(t *testing.T) {
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

func TestResetLinkIsEmailedWithoutRevealingTheToken(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)
	target := a.createUser("target@test.com", "oldpass12", roleEditor)

	res := a.do("POST", "/api/users/"+itoa(target.ID)+"/password-token", map[string]any{"send": true}, sess.Token, sess.CSRF)
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

func TestResetLinkIsNotSentWhenItCannotWork(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, admin := newMailApp(t, rec)
	target := a.createUser("target@test.com", "oldpass12", roleEditor)
	send := func(id int64) (int, map[string]any) {
		res := a.do("POST", "/api/users/"+itoa(id)+"/password-token", map[string]any{"send": true}, sess.Token, sess.CSRF)
		return res.StatusCode, decodeJSON(t, res)
	}

	if code, _ := send(admin.ID); code != 422 {
		t.Errorf("own account: %d", code)
	}

	// No public URL: the link would point nowhere.
	_ = a.s.Cfg.Live.SetAppURL("")
	if code, body := send(target.ID); code != 422 || !strings.Contains(toString(body), "public URL") {
		t.Errorf("no app_url: %d %v", code, body)
	}
	_ = a.s.Cfg.Live.SetAppURL("http://localhost:8080")

	// Password sign-in switched off in favour of SSO.
	insertSSO(t, a.db, true)
	_ = settings.Store{DB: a.db}.Set(context.Background(), "password_login_enabled", false)
	if code, _ := send(target.ID); code != 422 {
		t.Errorf("password login off: %d", code)
	}
	_ = settings.Store{DB: a.db}.Set(context.Background(), "password_login_enabled", true)

	// SMTP down: say so, and keep the old token usable for nobody else.
	rec.sendErr = errors.New("connection refused")
	if code, body := send(target.ID); code != 422 || !strings.Contains(toString(body), "connection refused") {
		t.Errorf("smtp down: %d %v", code, body)
	}
	if n := len(rec.messages()); n != 0 {
		t.Errorf("%d emails recorded", n)
	}

	// Mail off: asking to send is an error, not a silent fallback.
	off := &recorder{}
	a, sess, _ = newMailApp(t, off)
	target = a.createUser("target@test.com", "oldpass12", roleEditor)
	if code, _ := send(target.ID); code != 422 {
		t.Errorf("mail off: %d", code)
	}
}

// Without "send" the admin still gets the link to hand over, mail or not.
func TestResetLinkStillReturnsTheTokenWhenNotSending(t *testing.T) {
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

func TestNewUserGetsTheSetPasswordLinkByEmail(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)

	res := a.do("POST", "/api/users", map[string]any{"name": "New", "email": "new@test.com", "role": "user", "send": true}, sess.Token, sess.CSRF)
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

func TestNewUserEmailNeedsMailAndNoPassword(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, _ := newMailApp(t, rec)
	create := func(email string, extra map[string]any) int {
		in := map[string]any{"name": "New", "email": email, "role": "user"}
		for k, v := range extra {
			in[k] = v
		}
		return a.do("POST", "/api/users", in, sess.Token, sess.CSRF).StatusCode
	}

	if code := create("pw@test.com", map[string]any{"send": true, "password": "longenough1"}); code != 422 {
		t.Errorf("send with a password: %d", code)
	}
	// SMTP down: the user exists, nobody got the link, and the response says so.
	rec.sendErr = errors.New("connection refused")
	res := a.do("POST", "/api/users", map[string]any{"name": "New", "email": "down@test.com", "role": "user", "send": true}, sess.Token, sess.CSRF)
	data, _ := decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 201 || data["emailError"] == nil || data["token"] != nil {
		t.Errorf("smtp down: %d %v", res.StatusCode, data)
	}

	off := &recorder{}
	a, sess, _ = newMailApp(t, off)
	if code := create("off@test.com", map[string]any{"send": true}); code != 422 {
		t.Errorf("mail off: %d", code)
	}
}

// With verification required a link handed over by the admin proves nothing
// about the mailbox, so a user without a password has to be emailed.
func TestNewUserLinkIsRefusedWhenVerificationIsRequired(t *testing.T) {
	rec := &recorder{on: true}
	a, sess, admin := newMailApp(t, rec)
	_ = a.s.Users.SetEmailVerified(context.Background(), admin.ID, true)
	_ = settings.Store{DB: a.db}.Set(context.Background(), "require_email_verification", true)

	res := a.do("POST", "/api/users", map[string]any{"name": "New", "email": "new@test.com", "role": "user"}, sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Errorf("link instead of email: %d", res.StatusCode)
	}
	res = a.do("POST", "/api/users", map[string]any{"name": "New", "email": "new@test.com", "role": "user", "send": true}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		t.Errorf("emailed: %d", res.StatusCode)
	}
}

func toString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		return strings.TrimSpace(strings.Join(flatten(v), " "))
	}
}

func flatten(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case map[string]any:
		var out []string
		for k, x := range t {
			out = append(out, k)
			out = append(out, flatten(x)...)
		}
		return out
	case []any:
		var out []string
		for _, x := range t {
			out = append(out, flatten(x)...)
		}
		return out
	default:
		return nil
	}
}

func TestInvitationReturnsAnIdThatRevokesIt(t *testing.T) {
	a, sess, _ := newMailApp(t, &recorder{})
	path := "/api/spaces/" + itoa(a.space.ID) + "/invitations"

	res := a.do("POST", path, map[string]any{"role": "viewer"}, sess.Token, sess.CSRF)
	data := decodeJSON(t, res)["data"].(map[string]any)
	id, _ := data["id"].(float64)
	if res.StatusCode != 201 || id == 0 {
		t.Fatalf("invite %d %v", res.StatusCode, data)
	}

	res = a.do("DELETE", path+"/"+itoa(int64(id)), nil, sess.Token, sess.CSRF)
	if res.StatusCode != 204 {
		t.Fatalf("revoke %d", res.StatusCode)
	}
	list := decodeJSON(t, a.do("GET", path, nil, sess.Token, sess.CSRF))["data"].([]any)
	if len(list) != 0 {
		t.Errorf("invitation still listed: %v", list)
	}
}
