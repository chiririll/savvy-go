package mail

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/mail/smtp"
)

var expiry = time.Date(2026, 3, 4, 15, 30, 0, 0, time.UTC)

func TestMessagesCarryTheirLinkRecipient(t *testing.T) {
	const link = "https://savvy.example/some/link"
	cases := []struct {
		name string
		msg  smtp.Message
		to   string
		with []string
	}{
		{
			"password",
			PasswordReset("en", "ann@test.com", "Ann", link, expiry),
			"ann@test.com",
			[]string{"Hello Ann", "reset your Go Savvy password", "until 4 March"},
		},
		{
			"verification",
			EmailVerification("en", "ann@test.com", "Ann", link, expiry),
			"ann@test.com",
			[]string{"Hello Ann", "confirm that this is your email address", "until 4 March"},
		},
		{
			"invitation",
			Invitation("en", "bob@test.com", "Ann", "Home", "editor", link),
			"bob@test.com",
			[]string{`Ann invited you to the space "Home"`, "as editor"},
		},
	}
	for _, c := range cases {
		if c.msg.To != c.to || c.msg.Subject == "" {
			t.Errorf("%s: %+v", c.name, c.msg)
		}
		for part, body := range map[string]string{"text": c.msg.Text, "html": c.msg.HTML} {
			if !strings.Contains(body, link) {
				t.Errorf("%s %s: link missing from %q", c.name, part, body)
			}
			for _, want := range c.with {
				if !strings.Contains(strings.ReplaceAll(body, "&#34;", `"`), want) {
					t.Errorf("%s %s: %q missing from %q", c.name, part, want, body)
				}
			}
		}
	}
}

func TestTestEmailUsesTheLayoutWithoutALink(t *testing.T) {
	m := Test("en", "ann@test.com")
	if m.To != "ann@test.com" || m.Subject == "" || m.HTML == "" || len(m.Inline) != 1 {
		t.Fatalf("test email %+v", m)
	}
	if !strings.Contains(m.HTML, "prefers-color-scheme: dark") || !strings.Contains(m.Text, "Your mail settings work") {
		t.Errorf("not the themed layout: %q / %q", m.HTML, m.Text)
	}
	for _, body := range []string{m.Text, m.HTML} {
		if strings.Contains(body, "href=") || strings.Contains(body, "http") {
			t.Errorf("a test email has nothing to link to: %q", body)
		}
	}
	if ru := Test("ru", "ann@test.com"); ru.Subject == m.Subject {
		t.Errorf("not translated: %q", ru.Subject)
	}
}

func TestPlainTextStripsComponentsToTheirText(t *testing.T) {
	const link = "https://savvy.example/x"
	m := PasswordReset("en", "ann@test.com", "Ann", link, expiry)
	if strings.Contains(m.Text, "<") || strings.Count(m.Text, link) != 1 {
		t.Errorf("button must be its bare link, once: %q", m.Text)
	}
	if strings.Contains(m.Text, catalogs["en"].LinkFallback) || strings.Contains(m.Text, "\n\n\n") {
		t.Errorf("an empty component leaves a gap or the fallback stays: %q", m.Text)
	}
}

func TestExpiryIsADateWithoutTimeOrYear(t *testing.T) {
	m := PasswordReset("en", "ann@test.com", "Ann", "https://savvy.example/x", expiry)
	for _, body := range []string{m.Text, m.HTML} {
		if strings.Contains(body, "15:30") || strings.Contains(body, "2026") || strings.Contains(body, "UTC") {
			t.Errorf("expiry is not just a date: %q", body)
		}
	}
}

func TestMessagesAreTranslated(t *testing.T) {
	const link = "https://savvy.example/x"
	ru := PasswordReset("ru", "ann@test.com", "Анна", link, expiry)
	en := PasswordReset("en", "ann@test.com", "Анна", link, expiry)
	if ru.Subject == en.Subject || !strings.Contains(ru.Text, "до 4 марта") || !strings.Contains(ru.Text, "Анна") {
		t.Errorf("russian message: %+v", ru)
	}
	if !strings.Contains(ru.HTML, `lang="ru"`) {
		t.Errorf("html does not name its language: %q", ru.HTML)
	}
	inv := Invitation("ru", "bob@test.com", "Анна", "Дом", "editor", link)
	if !strings.Contains(inv.Text, "редактор") {
		t.Errorf("role was not translated: %q", inv.Text)
	}
}

func TestUnknownLanguageFallsBackToEnglish(t *testing.T) {
	const link = "https://savvy.example/x"
	want := PasswordReset("en", "ann@test.com", "Ann", link, expiry)
	for _, lang := range []string{"", "xx", "de-DE", "../etc"} {
		if got := PasswordReset(lang, "ann@test.com", "Ann", link, expiry); !reflect.DeepEqual(got, want) {
			t.Errorf("lang %q: %+v", lang, got)
		}
	}
	// A region or case does not matter.
	ru := PasswordReset("ru", "ann@test.com", "Ann", link, expiry)
	for _, lang := range []string{"ru-RU", "RU", " ru "} {
		if got := PasswordReset(lang, "ann@test.com", "Ann", link, expiry); !reflect.DeepEqual(got, ru) {
			t.Errorf("lang %q is not russian", lang)
		}
	}
}

func TestHTMLEscapesNamesAndLinksFallbackToo(t *testing.T) {
	const link = "https://savvy.example/x?a=1&b=2"
	m := Invitation("en", "bob@test.com", `<b>Ann</b>`, `"><script>alert(1)</script>`, "viewer", link)
	if strings.Contains(m.HTML, "<script>") || strings.Contains(m.HTML, "<b>Ann") {
		t.Errorf("names were not escaped: %q", m.HTML)
	}
	if strings.Count(m.HTML, "a=1&amp;b=2") < 2 {
		t.Errorf("the button and the fallback must both carry the link: %q", m.HTML)
	}
	if !strings.Contains(m.Text, `<b>Ann</b>`) {
		t.Errorf("plain text must not be html-escaped: %q", m.Text)
	}
}

func TestHTMLIsSelfContained(t *testing.T) {
	m := PasswordReset("en", "ann@test.com", "Ann", "https://savvy.example/x", expiry)
	if strings.Contains(m.HTML, "http://") || strings.Contains(m.HTML, "src=\"https:") {
		t.Errorf("remote resources leak a request: %q", m.HTML)
	}
	if !strings.Contains(m.HTML, "prefers-color-scheme: dark") || !strings.Contains(m.HTML, `src="cid:logo"`) {
		t.Errorf("dark mode or logo is missing: %q", m.HTML)
	}
	if len(m.Inline) != 1 || m.Inline[0].CID != "logo" || len(m.Inline[0].Data) == 0 {
		t.Errorf("logo is not attached: %+v", m.Inline)
	}
}

func TestEveryLanguageHasEveryString(t *testing.T) {
	en := catalogs["en"]
	for lang, c := range catalogs {
		for _, name := range []string{"password_reset", "email_verification", "invitation", "test"} {
			if got, want := c.Messages[name], en.Messages[name]; got.Subject == "" || got.Body == "" || (got.Button == "") != (name == "test") {
				t.Errorf("%s/%s: incomplete %+v", lang, name, got)
			} else if (got.Expiry == "") != (want.Expiry == "") {
				t.Errorf("%s/%s: expiry differs from en", lang, name)
			}
		}
		if len(c.Months) != 12 || c.Ignore == "" || c.LinkFallback == "" || c.SentBy == "" {
			t.Errorf("%s: incomplete catalog", lang)
		}
		for _, role := range []string{"admin", "editor", "viewer"} {
			if c.Roles[role] == "" {
				t.Errorf("%s: role %s", lang, role)
			}
		}
	}
}

func TestNamesCannotBreakOutHeaderLine(t *testing.T) {
	m := Invitation("en", "bob@test.com", "Ann", "Home\r\nBcc: evil@example.org", "viewer", "https://savvy.example/x")
	if strings.ContainsAny(m.Subject, "\r\n") {
		t.Fatalf("subject %q spans lines", m.Subject)
	}
	if strings.Contains(m.Text, "\r\n") || !strings.Contains(m.Subject, "Home Bcc: evil@example.org") {
		t.Errorf("name was not flattened: %q / %q", m.Subject, m.Text)
	}
}
