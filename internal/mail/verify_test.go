package mail

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/mail/smtp"
)

// recorder is a mailer that keeps what it is asked to send.
type recorder struct {
	sent []smtp.Message
	fail error
}

func (r *recorder) Enabled() bool               { return true }
func (r *recorder) Check(context.Context) error { return nil }
func (r *recorder) Send(_ context.Context, m smtp.Message) error {
	if r.fail != nil {
		return r.fail
	}
	r.sent = append(r.sent, m)
	return nil
}

type verifyEnv struct {
	ctx   context.Context
	v     Verifier
	rec   *recorder
	users auth.Users
	user  *auth.User
}

func newVerifyEnv(t *testing.T, appURL string, mailOn bool) *verifyEnv {
	t.Helper()
	d := dbtest.Server(t)
	users := auth.Users{DB: d}
	pw := "pw-pw-pw"
	u, err := users.Create(context.Background(), "Ann", "ann@test.com", &pw, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	cfg := appconfig.Config{}.WithAppURL(appURL)
	rec := &recorder{}
	var sender smtp.Mailer = smtp.Disabled{}
	if mailOn {
		sender = rec
	}
	return &verifyEnv{
		ctx: context.Background(), rec: rec, users: users, user: u,
		v: Verifier{
			Cfg:           cfg,
			Mail:          New(config.Empty(), cfg, sender),
			Verifications: auth.EmailVerifications{DB: d},
			Users:         users,
		},
	}
}

func TestCanRequireNeedsMailURLAdmin(t *testing.T) {
	t.Run("mail is off", func(t *testing.T) {
		e := newVerifyEnv(t, "https://app.test", false)
		var me MailRequiredError
		if err := e.v.CanRequire(e.user); !errors.As(err, &me) || !strings.Contains(err.Error(), "Mail is not configured") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("no public URL", func(t *testing.T) {
		e := newVerifyEnv(t, "", true)
		var me MailRequiredError
		if err := e.v.CanRequire(e.user); !errors.As(err, &me) || !strings.Contains(err.Error(), NoAppURLMessage) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("the admin has not confirmed their own address", func(t *testing.T) {
		e := newVerifyEnv(t, "https://app.test", true)
		if err := e.v.CanRequire(e.user); !errors.Is(err, ErrOwnAddressUnverified) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("everything is in place", func(t *testing.T) {
		e := newVerifyEnv(t, "https://app.test", true)
		e.user.EmailVerified = true
		if err := e.v.CanRequire(e.user); err != nil {
			t.Errorf("err = %v", err)
		}
	})
}

func TestSendEmailsLinkThatConfirmsAddress(t *testing.T) {
	e := newVerifyEnv(t, "https://app.test", true)
	if why := e.v.Send(e.ctx, e.user); why != "" {
		t.Fatalf("Send: %s", why)
	}
	if len(e.rec.sent) != 1 || e.rec.sent[0].To != "ann@test.com" {
		t.Fatalf("sent %+v", e.rec.sent)
	}
	_, after, ok := strings.Cut(e.rec.sent[0].Text, "https://app.test/verify-email/")
	if !ok {
		t.Fatalf("no link in %q", e.rec.sent[0].Text)
	}
	token := strings.Fields(after)[0]

	if ok, err := e.v.Confirm(e.ctx, "wrong"); ok || err != nil {
		t.Errorf("a wrong token: %v, %v", ok, err)
	}
	if ok, err := e.v.Confirm(e.ctx, token); !ok || err != nil {
		t.Fatalf("Confirm: %v, %v", ok, err)
	}
	if u, _ := e.users.ByID(e.ctx, e.user.ID); !u.EmailVerified {
		t.Error("the address was not marked as confirmed")
	}
	if ok, _ := e.v.Confirm(e.ctx, token); ok {
		t.Error("a link worked twice")
	}
}

func TestSendSaysWhyItCannot(t *testing.T) {
	off := newVerifyEnv(t, "https://app.test", false)
	if why := off.v.Send(off.ctx, off.user); !strings.Contains(why, "Mail is not configured") {
		t.Errorf("mail off: %q", why)
	}
	noURL := newVerifyEnv(t, "", true)
	if why := noURL.v.Send(noURL.ctx, noURL.user); why != NoAppURLMessage || len(noURL.rec.sent) != 0 {
		t.Errorf("no URL: %q", why)
	}
	broken := newVerifyEnv(t, "https://app.test", true)
	broken.rec.fail = errors.New("connection refused")
	if why := broken.v.Send(broken.ctx, broken.user); !strings.Contains(why, "connection refused") {
		t.Errorf("a failing server: %q", why)
	}
}

func TestOffReasonNamesMistakeInSection(t *testing.T) {
	f, err := config.Open(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(f, appconfig.Config{}, nil)
	if err := f.Finish(); err != nil {
		t.Fatal(err)
	}
	if reason, hint := svc.OffWhy(); reason != "Mail is not configured." || !strings.Contains(hint, "[mailer]") {
		t.Errorf("not configured: %q %q", reason, hint)
	}
	if enabled, problem := svc.Status(); enabled || problem != "" {
		t.Errorf("status %v %q", enabled, problem)
	}

	err = smtp.Section.Update(f, func(s *smtp.MailerConfig) { s.Enabled, s.Host = true, "" })
	if err != nil {
		t.Fatal(err)
	}
	reason, hint := svc.OffWhy()
	if !strings.Contains(reason, "Mail is switched off") || !strings.Contains(reason, "host") || hint != "" {
		t.Errorf("invalid section: %q %q", reason, hint)
	}
	if full := svc.OffReason(); full != reason {
		t.Errorf("OffReason without a hint is the reason: %q", full)
	}
	if _, problem := svc.Status(); problem == "" {
		t.Error("the problem is reported")
	}
}

func TestSendReturnsMailerError(t *testing.T) {
	rec := &recorder{fail: errors.New("boom")}
	svc := New(config.Empty(), appconfig.Config{}, rec)
	if err := svc.Send(context.Background(), smtp.Message{To: "a@b.c"}); err == nil || err.Error() != "boom" {
		t.Errorf("err = %v", err)
	}
	if svc.AppURL() != "" {
		t.Errorf("AppURL %q", svc.AppURL())
	}
}
