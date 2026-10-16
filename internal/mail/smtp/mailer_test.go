package smtp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// inbox is a fake SMTP server that keeps what it receives.
type inbox struct {
	mu   sync.Mutex
	got  []received
	user string // when set, AUTH PLAIN with user/pass is required
	pass string
}

type received struct{ from, to, body string }

func (b *inbox) NewSession(*smtp.Conn) (smtp.Session, error) { return &session{b: b}, nil }

func (b *inbox) messages() []received {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]received(nil), b.got...)
}

type session struct {
	b        *inbox
	from, to string
	authed   bool
}

func (s *session) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *session) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, user, pass string) error {
		if user != s.b.user || pass != s.b.pass {
			return errors.New("bad credentials")
		}
		s.authed = true
		return nil
	}), nil
}
func (s *session) Mail(from string, _ *smtp.MailOptions) error {
	if s.b.user != "" && !s.authed {
		return errors.New("authentication required")
	}
	s.from = from
	return nil
}
func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error { s.to = to; return nil }
func (s *session) Data(r io.Reader) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.b.mu.Lock()
	s.b.got = append(s.b.got, received{s.from, s.to, string(body)})
	s.b.mu.Unlock()
	return nil
}
func (s *session) Reset()        {}
func (s *session) Logout() error { return nil }

func selfSigned(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// serve starts the fake server. withTLS offers STARTTLS; implicit wraps the
// listener in TLS (SMTPS). It returns the client config pointing at it.
func serve(t *testing.T, b *inbox, withTLS, implicit bool) Config {
	t.Helper()
	srv := smtp.NewServer(b)
	srv.Domain = "localhost"
	srv.AllowInsecureAuth = !withTLS
	if withTLS {
		srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{selfSigned(t)}, MinVersion: tls.VersionTLS12}
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if implicit {
		l = tls.NewListener(l, srv.TLSConfig)
	}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })
	return Config{
		Enabled:  true,
		Host:     "127.0.0.1",
		Port:     l.Addr().(*net.TCPAddr).Port,
		From:     "Go Savvy <savvy@example.com>",
		Timeout:  5 * time.Second,
		Protocol: STARTTLS,
	}
}

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
	c := Config{Enabled: true, Host: "127.0.0.1", Port: port, Protocol: Plain, From: "a@example.com", Timeout: 2 * time.Second}
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
	ok := Config{Enabled: true, Host: "smtp.example.com", Port: 587, Protocol: STARTTLS, From: "Go Savvy <savvy@example.com>"}
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
