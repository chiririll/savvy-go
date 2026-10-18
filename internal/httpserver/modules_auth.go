package httpserver

// The sign-in, two-factor, OAuth and SSO flows.
import (
	_ "savvy-go/internal/httpserver/auth/emailverify"
	_ "savvy-go/internal/httpserver/auth/oauth"
	_ "savvy-go/internal/httpserver/auth/session"
	_ "savvy-go/internal/httpserver/auth/sso"
	_ "savvy-go/internal/httpserver/auth/twofactor"
	_ "savvy-go/internal/httpserver/auth/webauthn"
)
