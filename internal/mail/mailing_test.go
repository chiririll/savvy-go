package mail

import (
	"errors"
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

func newSavedService(t *testing.T) (*Service, *config.Config) {
	t.Helper()
	file, err := config.Open(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(file, appconfig.Config{}, nil)
	if err := file.Finish(); err != nil {
		t.Fatal(err)
	}
	return svc, file
}

func validUpdate() Settings {
	return Settings{
		Enabled: true, Host: " smtp.example.com ", Port: 465, Protocol: "SMTPS",
		From: "Go Savvy <savvy@example.com>", Timeout: 7,
	}
}

func TestSaveChangesTheMailerAndKeepsThePassword(t *testing.T) {
	svc, file := newSavedService(t)
	pw := "s3cret"
	in := validUpdate()
	in.Password = &pw
	if err := svc.Save(in); err != nil {
		t.Fatal(err)
	}
	conf, problem := svc.Config()
	if !conf.Enabled || conf.Host != "smtp.example.com" || conf.Port != 465 || conf.Protocol != "smtps" ||
		conf.Password != "s3cret" || conf.Timeout.Seconds() != 7 || problem != "" {
		t.Fatalf("saved %+v %q", conf, problem)
	}

	// No password in the update keeps the stored one; an empty one clears it.
	in = validUpdate()
	in.User = ""
	if err := svc.Save(in); err != nil {
		t.Fatal(err)
	}
	if conf, _ := svc.Config(); conf.Password != "s3cret" {
		t.Errorf("password lost: %q", conf.Password)
	}
	empty := ""
	in.Password = &empty
	if err := svc.Save(in); err != nil {
		t.Fatal(err)
	}
	if conf, _ := svc.Config(); conf.Password != "" {
		t.Errorf("password not cleared: %q", conf.Password)
	}

	// The change reached the file, not only memory.
	again, err := config.Open(file.Path())
	if err != nil {
		t.Fatal(err)
	}
	svc2 := New(again, appconfig.Config{}, nil)
	if err := again.Finish(); err != nil {
		t.Fatal(err)
	}
	if conf, _ := svc2.Config(); conf.Host != "smtp.example.com" || conf.Port != 465 {
		t.Errorf("not on disk: %+v", conf)
	}
}

func TestSaveRefusesInvalidMailerAndChangesNothing(t *testing.T) {
	svc, _ := newSavedService(t)
	if err := svc.Save(validUpdate()); err != nil {
		t.Fatal(err)
	}
	in := validUpdate()
	in.Host = ""
	err := svc.Save(in)
	var bad smtp.InvalidError
	if !errors.As(err, &bad) || bad.Field != "host" {
		t.Fatalf("want an InvalidError for host, got %v", err)
	}
	if conf, problem := svc.Config(); conf.Host != "smtp.example.com" || problem != "" {
		t.Errorf("a rejected update changed the mailer: %+v %q", conf, problem)
	}

	// A disabled mailer may be half filled in.
	in.Enabled = false
	if err := svc.Save(in); err != nil {
		t.Errorf("disabled update rejected: %v", err)
	}
}

func TestSaveFailsOnAReadOnlyFile(t *testing.T) {
	svc := New(config.Empty(), appconfig.Config{}, nil)
	if err := svc.Save(validUpdate()); !errors.Is(err, config.ErrReadOnly) {
		t.Fatalf("want ErrReadOnly, got %v", err)
	}
}

func TestServiceUsesSenderItWasGiven(t *testing.T) {
	sender := smtp.Disabled{}
	svc := New(config.Empty(), appconfig.Config{}, sender)
	if svc.Mailer() != sender {
		t.Error("the given sender was replaced")
	}
}
