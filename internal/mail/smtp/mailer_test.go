package smtp

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func msg() Message {
	return Message{To: "ann@example.com", Subject: "Hello", Text: "Reset your password at https://x.test/set-password/abc"}
}

func requireDelivered(t *testing.T, b *inbox) {
	t.Helper()
	got := b.messages()
	if len(got) != 1 {
		t.Fatalf("want 1 message, got %d", len(got))
	}
	if !strings.Contains(got[0].from, "savvy@example.com") || !strings.Contains(got[0].to, "ann@example.com") {
		t.Errorf("envelope: %+v", got[0])
	}
	for _, want := range []string{"Subject: Hello", "https://x.test/set-password/abc"} {
		if !strings.Contains(got[0].body, want) {
			t.Errorf("body lacks %q:\n%s", want, got[0].body)
		}
	}
}

func TestSendPlain(t *testing.T) {
	b := &inbox{}
	c := serve(t, b, false, false)
	c.Protocol = Plain
	if err := New(c).Send(context.Background(), msg()); err != nil {
		t.Fatal(err)
	}
	requireDelivered(t, b)
}

func TestSendPlainWithAuth(t *testing.T) {
	b := &inbox{user: "savvy", pass: "secret"}
	c := serve(t, b, false, false)
	c.Protocol, c.User, c.Password = Plain, "savvy", "secret"
	if err := New(c).Send(context.Background(), msg()); err != nil {
		t.Fatal(err)
	}
	requireDelivered(t, b)
}

func TestSendSTARTTLS(t *testing.T) {
	b := &inbox{user: "savvy", pass: "secret"}
	c := serve(t, b, true, false)
	c.User, c.Password, c.SkipVerify = "savvy", "secret", true
	if err := New(c).Send(context.Background(), msg()); err != nil {
		t.Fatal(err)
	}
	requireDelivered(t, b)
}

func TestSendSMTPS(t *testing.T) {
	b := &inbox{}
	c := serve(t, b, true, true)
	c.Protocol, c.SkipVerify = SMTPS, true
	if err := New(c).Send(context.Background(), msg()); err != nil {
		t.Fatal(err)
	}
	requireDelivered(t, b)
}

func TestCertificateIsVerifiedUnlessSkipped(t *testing.T) {
	for _, protocol := range []string{STARTTLS, SMTPS} {
		b := &inbox{}
		c := serve(t, b, true, protocol == SMTPS)
		c.Protocol = protocol
		if err := New(c).Send(context.Background(), msg()); err == nil {
			t.Errorf("%s: self-signed certificate accepted without skip_verify", protocol)
		}
		if n := len(b.messages()); n != 0 {
			t.Errorf("%s: %d messages delivered over an unverified connection", protocol, n)
		}
	}
}

// starttls must never fall back to an unencrypted session.
func TestSTARTTLSDoesNotDowngrade(t *testing.T) {
	b := &inbox{}
	c := serve(t, b, false, false) // the server offers no STARTTLS
	c.SkipVerify = true
	if err := New(c).Send(context.Background(), msg()); err == nil {
		t.Fatal("sent in clear text although starttls is required")
	}
	if n := len(b.messages()); n != 0 {
		t.Errorf("%d messages delivered unencrypted", n)
	}
}

func TestWrongPassword(t *testing.T) {
	b := &inbox{user: "savvy", pass: "secret"}
	c := serve(t, b, true, false)
	c.User, c.Password, c.SkipVerify = "savvy", "wrong", true
	if err := New(c).Send(context.Background(), msg()); err == nil {
		t.Fatal("bad credentials accepted")
	}
}

func TestCheck(t *testing.T) {
	b := &inbox{user: "savvy", pass: "secret"}
	c := serve(t, b, true, false)
	c.User, c.Password, c.SkipVerify = "savvy", "secret", true
	if err := New(c).Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(b.messages()); n != 0 {
		t.Errorf("Check sent %d messages", n)
	}
	c.Password = "wrong"
	if err := New(c).Check(context.Background()); err == nil {
		t.Error("Check accepted bad credentials")
	}
}

func TestUnreachableServerFailsFast(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close() // nothing listens there now
	c := Config{
		Enabled:  true,
		Host:     "127.0.0.1",
		Port:     port,
		Protocol: Plain,
		From:     "a@example.com",
		Timeout:  2 * time.Second,
	}
	start := time.Now()
	if err := New(c).Send(context.Background(), msg()); err == nil {
		t.Fatal("no error for an unreachable server")
	}
	if time.Since(start) > 10*time.Second {
		t.Errorf("took %v", time.Since(start))
	}
}

func TestDisabled(t *testing.T) {
	m := New(Config{})
	if m.Enabled() {
		t.Error("an empty config must not be enabled")
	}
	if err := m.Send(context.Background(), msg()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Send: %v", err)
	}
	if err := m.Check(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Check: %v", err)
	}
}

func TestValidate(t *testing.T) {
	ok := Config{
		Enabled:  true,
		Host:     "smtp.example.com",
		Port:     587,
		Protocol: STARTTLS,
		From:     "Go Savvy <savvy@example.com>",
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if err := (Config{Protocol: "nonsense"}).Validate(); err != nil {
		t.Errorf("a disabled config is always valid: %v", err)
	}
	cases := map[string]func(*Config){
		"host":     func(c *Config) { c.Host = " " },
		"port":     func(c *Config) { c.Port = 0 },
		"port ":    func(c *Config) { c.Port = 70000 },
		"protocol": func(c *Config) { c.Protocol = "ssl" },
		"from":     func(c *Config) { c.From = "not an address" },
		"password": func(c *Config) { c.User = "savvy" },
	}
	for name, mutate := range cases {
		c := ok
		mutate(&c)
		err := c.Validate()
		if err == nil {
			t.Errorf("%s: accepted", name)
		} else if !strings.Contains(err.Error(), "mailer.") {
			t.Errorf("%s: error does not name the key: %v", name, err)
		}
	}
}

func TestSendHTMLIsAnAlternativeAfterTheText(t *testing.T) {
	b := &inbox{}
	c := serve(t, b, false, false)
	c.Protocol = Plain
	m := msg()
	m.HTML = "<p>Reset your password at <a href=\"https://x.test/set-password/abc\">this link</a></p>"
	m.Inline = []Inline{{CID: "logo", Name: "logo.png", Type: "image/png", Data: []byte("not really a png")}}
	if err := New(c).Send(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	got := b.messages()
	if len(got) != 1 {
		t.Fatalf("want 1 message, got %d", len(got))
	}
	body := got[0].body
	if !strings.Contains(body, "multipart/alternative") {
		t.Fatalf("not multipart/alternative:\n%s", body)
	}
	text, html, logo := strings.Index(body, "text/plain"), strings.Index(body, "text/html"), strings.Index(body, "Content-Id: <logo>")
	if text < 0 || html < 0 || text > html {
		t.Errorf("text must come first (text %d, html %d)", text, html)
	}
	if logo < 0 || !strings.Contains(body, "image/png") {
		t.Errorf("logo is not attached inline:\n%s", body)
	}
}

func TestSendWithoutHTMLStaysPlainText(t *testing.T) {
	b := &inbox{}
	c := serve(t, b, false, false)
	c.Protocol = Plain
	if err := New(c).Send(context.Background(), msg()); err != nil {
		t.Fatal(err)
	}
	if body := b.messages()[0].body; strings.Contains(body, "multipart") {
		t.Errorf("a text message became multipart:\n%s", body)
	}
}
