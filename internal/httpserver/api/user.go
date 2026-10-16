package api

import (
	"context"
	"net/http"

	"savvy-go/internal/auth"
)

const (
	ctxUser    ctxKey = 100
	ctxSession ctxKey = 101
	ctxToken   ctxKey = 102
)

// WithSession returns ctx carrying the cookie session of a request.
func WithSession(ctx context.Context, s *auth.Session) context.Context {
	return context.WithValue(ctx, ctxSession, s)
}

// Session is the cookie session of a request, nil when it came with an API token.
func Session(r *http.Request) *auth.Session {
	s, _ := r.Context().Value(ctxSession).(*auth.Session)
	return s
}

// WithToken returns ctx carrying the API token of a request.
func WithToken(ctx context.Context, t *auth.APIToken) context.Context {
	return context.WithValue(ctx, ctxToken, t)
}

// Token is the API token of a request, nil when it came with a session.
func Token(r *http.Request) *auth.APIToken {
	t, _ := r.Context().Value(ctxToken).(*auth.APIToken)
	return t
}

// WithUser returns ctx carrying the signed-in user of a request.
func WithUser(ctx context.Context, u *auth.User) context.Context {
	return context.WithValue(ctx, ctxUser, u)
}

// User is the signed-in user of a request that passed requireSession.
func User(r *http.Request) *auth.User {
	u, _ := r.Context().Value(ctxUser).(*auth.User)
	return u
}
