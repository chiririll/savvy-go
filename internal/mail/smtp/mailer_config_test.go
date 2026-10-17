package smtp

import (
	"testing"
	"time"
)

func TestDecodeTrimsAndFillsTheTimeout(t *testing.T) {
	got, err := decode(MailerConfig{Enabled: true, Host: " h ", Port: 465, Protocol: " SMTPS ", From: " a@b.example "})
	if err != nil || got.Problem != "" {
		t.Fatalf("%+v %v", got, err)
	}
	if c := got.Config; !c.Enabled || c.Host != "h" || c.Protocol != SMTPS || c.From != "a@b.example" ||
		c.Timeout != DefaultTimeout {
		t.Errorf("config %+v", c)
	}
	got, _ = decode(MailerConfig{Timeout: 3})
	if got.Config.Timeout != 3*time.Second {
		t.Errorf("timeout %v", got.Config.Timeout)
	}
}

func TestDecodeSwitchesMailOffOnAMistake(t *testing.T) {
	got, err := decode(MailerConfig{Enabled: true, Port: 587, Protocol: STARTTLS, From: "a@b.example"})
	if err != nil {
		t.Fatalf("a mistake must not stop the server: %v", err)
	}
	if got.Config.Enabled || got.Problem != "mailer.host is empty" {
		t.Errorf("%+v", got)
	}
}
