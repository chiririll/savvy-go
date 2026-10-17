package smtp

import (
	"log/slog"
	"strings"
	"time"

	"savvy-go/internal/config"
)

// MailerConfig is the [mailer] table of the config file: the SMTP server.
type MailerConfig struct {
	Enabled    bool   `toml:"enabled"`
	Host       string `toml:"host"`
	Port       int    `toml:"port"`
	Protocol   string `toml:"protocol" comment:"Supported: \"starttls\", \"smtps\" or \"smtp\""`
	User       string `toml:"user"`
	Password   string `toml:"password"`
	From       string `toml:"from"`
	SkipVerify bool   `toml:"skip_verify"`
	Timeout    int    `toml:"timeout" comment:"Seconds."`
}

// Defaults are the values of the keys the file lacks.
func (MailerConfig) Defaults() MailerConfig {
	return MailerConfig{Port: 587, Protocol: STARTTLS, Timeout: int(DefaultTimeout / time.Second)}
}

// Setup is the [mailer] section as the mailer uses it.
type Setup struct {
	Config Config
	// Problem is why mail is switched off although the section enables it, empty when it
	// is not.
	Problem string
}

// Section is where the config file keeps MailerConfig.
var Section = config.Section[MailerConfig, Setup]{Name: "mailer", Decode: decode}

// decode trims spaces and lower-cases the protocol. A timeout that is not positive becomes
// DefaultTimeout. A mistake does not stop the server: it switches mail off, and Problem
// says why.
func decode(m MailerConfig) (Setup, error) {
	timeout := DefaultTimeout
	if m.Timeout > 0 {
		timeout = time.Duration(m.Timeout) * time.Second
	}
	conf := Config{
		Enabled:    m.Enabled,
		Host:       strings.TrimSpace(m.Host),
		Port:       m.Port,
		Protocol:   strings.ToLower(strings.TrimSpace(m.Protocol)),
		User:       m.User,
		Password:   m.Password,
		From:       strings.TrimSpace(m.From),
		SkipVerify: m.SkipVerify,
		Timeout:    timeout,
	}
	if err := conf.Validate(); err != nil {
		conf.Enabled = false
		slog.Warn("mail is disabled: invalid [mailer] section in the config file", "problem", err.Error())
		return Setup{Config: conf, Problem: err.Error()}, nil
	}
	return Setup{Config: conf}, nil
}
