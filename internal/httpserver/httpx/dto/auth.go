package dto

// TwoFactorRequired answers a first factor that still needs the second. Its
// keys are snake_case like the rest of the sign-in flow.
type TwoFactorRequired struct {
	Required bool   `json:"requires_2fa"`
	Token    string `json:"two_factor_token"`
}

func NewTwoFactorRequired(token string) TwoFactorRequired {
	return TwoFactorRequired{Required: true, Token: token}
}

// WebAuthnChallenge is a WebAuthn ceremony the browser has to answer; Token
// comes back with the response.
type WebAuthnChallenge struct {
	Token   string `json:"token"`
	Options any    `json:"options"`
}
