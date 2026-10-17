package mail

import (
	"path/filepath"
	"testing"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/config"
	"savvy-go/internal/mail/smtp"
)

func TestServiceReadsTheMailerSection(t *testing.T) {
	file, err := config.Open(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(file, appconfig.Config{}, nil)
	if err := file.Finish(); err != nil {
		t.Fatal(err)
	}

	if conf, problem := svc.Config(); conf.Enabled || problem != "" || svc.Mailer().Enabled() {
		t.Fatalf("a new install must have mail off without a problem: %+v %q", conf, problem)
	}

	err = smtp.Section.Update(file, func(s *smtp.MailerConfig) {
		s.Enabled, s.Host, s.From = true, " smtp.example.com ", "savvy@example.com"
	})
	if err != nil {
		t.Fatal(err)
	}
	conf, problem := svc.Config()
	if !conf.Enabled || conf.Host != "smtp.example.com" || problem != "" || !svc.Mailer().Enabled() {
		t.Fatalf("change not seen: %+v %q", conf, problem)
	}

	// A mistake switches mail off and says why.
	if err = smtp.Section.Update(file, func(s *smtp.MailerConfig) { s.Host = "" }); err != nil {
		t.Fatal(err)
	}
	if conf, problem := svc.Config(); conf.Enabled || problem == "" || svc.Mailer().Enabled() {
		t.Errorf("invalid section kept mail on: %+v %q", conf, problem)
	}
}

func TestServiceUsesTheSenderItWasGiven(t *testing.T) {
	sender := smtp.Disabled{}
	svc := New(config.Empty(), appconfig.Config{}, sender)
	if svc.Mailer() != sender {
		t.Error("the given sender was replaced")
	}
}
