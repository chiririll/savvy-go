package httpserver

import (
	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/store"
)

// accountServices are the server-level services behind sign-in and accounts.
type accountServices struct {
	users         auth.Users
	sessions      auth.Sessions
	apiTokens     auth.APITokens
	tokens        auth.PasswordTokens
	challenges    auth.Challenges
	verifications auth.EmailVerifications
	twoFactor     auth.TwoFactor
	webauthn      auth.WebAuthn
}

func newAccountServices(cfg config.Config, srv store.DB) accountServices {
	return accountServices{
		users:         auth.Users{DB: srv},
		sessions:      auth.Sessions{DB: srv, SessionTTL: cfg.SessionTTL, RememberTTL: cfg.RememberTTL},
		apiTokens:     auth.APITokens{DB: srv},
		tokens:        auth.PasswordTokens{DB: srv},
		challenges:    auth.Challenges{DB: srv, Cfg: cfg},
		verifications: auth.EmailVerifications{DB: srv},
		twoFactor:     auth.TwoFactor{DB: srv, Users: auth.Users{DB: srv}},
		webauthn:      auth.WebAuthn{DB: srv, Cfg: cfg},
	}
}
