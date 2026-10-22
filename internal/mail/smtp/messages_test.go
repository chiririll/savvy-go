package smtp

import (
	"strings"
	"testing"
	"time"
)

var expiry = time.Date(2026, 3, 4, 15, 30, 0, 0, time.UTC)

func TestMessagesCarryTheirLinkRecipient(t *testing.T) {
	const link = "https://savvy.example/some/link"
	cases := []struct {
		name string
		msg  Message
		to   string
		with []string
	}{
		{
			"password",
			PasswordReset("ann@test.com", "Ann", link, expiry),
			"ann@test.com",
			[]string{"Hello Ann", "reset your Go Savvy password", "2026-03-04 15:30 UTC"},
		},
		{
			"verification",
			EmailVerification("ann@test.com", "Ann", link, expiry),
			"ann@test.com",
			[]string{"Hello Ann", "confirm that this is your email address", "2026-03-04 15:30 UTC"},
		},
		{
			"invitation",
			Invitation("bob@test.com", "Ann", "Home", "editor", link),
			"bob@test.com",
			[]string{`Ann invited you to the space "Home"`, "as editor"},
		},
	}
	for _, c := range cases {
		if c.msg.To != c.to || !strings.Contains(c.msg.Text, link) || c.msg.Subject == "" {
			t.Errorf("%s: %+v", c.name, c.msg)
		}
		for _, want := range c.with {
			if !strings.Contains(c.msg.Text, want) {
				t.Errorf("%s: %q missing from %q", c.name, want, c.msg.Text)
			}
		}
	}
}

func TestNamesCannotBreakOutHeaderLine(t *testing.T) {
	m := Invitation("bob@test.com", "Ann", "Home\r\nBcc: evil@example.org", "viewer", "https://savvy.example/x")
	if strings.ContainsAny(m.Subject, "\r\n") {
		t.Fatalf("subject %q spans lines", m.Subject)
	}
	if strings.Contains(m.Text, "\r\n") || !strings.Contains(m.Subject, "Home Bcc: evil@example.org") {
		t.Errorf("name was not flattened: %q / %q", m.Subject, m.Text)
	}
}
