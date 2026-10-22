package dto_test

import (
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/dto"
)

func TestSessionPayloadWithoutSession(t *testing.T) {
	got := marshal(t, dto.SessionPayload{})
	want := `{"user":null,"expires_at":null,"refresh_at":null,"email_verification_required":false}`
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestUserWithLink(t *testing.T) {
	at := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	u := dto.NewUser(auth.User{ID: 1, Name: "A", Email: "a@x", Role: "editor", CreatedAt: &at})

	// A user with a password carries no link keys at all.
	plain := decode(t, dto.UserWithLink{User: u})
	for _, k := range []string{"expiresAt", "token", "emailSent", "emailError"} {
		if _, has := plain[k]; has {
			t.Errorf("plain user has %q: %v", k, plain)
		}
	}
	if plain["createdAt"] != "2030-01-02T03:04:05Z" || plain["isInactive"] == nil {
		t.Errorf("user = %v", plain)
	}

	linked := decode(t, dto.UserWithLink{User: u, SetupLink: &dto.SetupLink{ExpiresAt: at, Token: "tok"}})
	if linked["token"] != "tok" || linked["expiresAt"] != "2030-01-02T03:04:05Z" || linked["email"] != "a@x" {
		t.Errorf("linked user = %v", linked)
	}
	if _, has := linked["emailSent"]; has {
		t.Errorf("emailSent present when nothing was mailed: %v", linked)
	}
}

func TestTwoFactorStatusNullWhenOff(t *testing.T) {
	got := marshal(t, dto.TwoFactorStatus{})
	if want := `{"enabled":false,"pending_confirmation":false,"recovery_codes_remaining":null}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
