package smtp

import (
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

func (s *session) Reset() {}

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
