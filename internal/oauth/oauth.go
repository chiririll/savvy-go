// Package oauth makes Go Savvy an OAuth 2.1 authorization server for MCP
// clients such as Claude web: dynamic client registration, authorization
// codes with PKCE, and rotating refresh tokens.
//
// Access tokens are ordinary API tokens (internal/auth) that hold what the user
// ticked on the consent screen for the resources the MCP tools use (read unless
// they chose more), in the spaces they picked. They
// show up in the user's token list, can be revoked there, and are limited by
// the owner's role in every space exactly like other API tokens.
package oauth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/oauth"
	"savvy-go/internal/store"
)

const (
	// MaxClients caps dynamic registrations; the endpoint is unauthenticated.
	MaxClients = 500

	// RegistrationWindow is how long an admin's "allow new apps" stays open.
	RegistrationWindow = 5 * time.Minute

	AccessTTL  = time.Hour
	RefreshTTL = 30 * 24 * time.Hour
	codeTTL    = time.Minute
	// A client that registered but never finished a login is dropped after this.
	idleClientTTL = 24 * time.Hour

	refreshPrefix = "svr_"
	maxNameLen    = 100
	maxRedirects  = 10
	maxURILen     = 2000
	maxIPLen      = 64
	maxUALen      = 300
	maxOriginLen  = 200
	defaultName   = "MCP client"
)

// Error is an OAuth error response (RFC 6749 §5.2, RFC 7591 §3.2.2).
type Error struct{ Code, Description string }

func (e *Error) Error() string { return e.Code + ": " + e.Description }

func fail(code, description string) *Error { return &Error{Code: code, Description: description} }

type (
	Client      = repo.Client
	RequestInfo = repo.RequestInfo
)

// TokenSet is what the token endpoint returns.
type TokenSet struct {
	AccessToken, RefreshToken string
	ExpiresIn                 int
}

// CodeRequest is a user's approval of one client, turned into a code.
type CodeRequest struct {
	ClientID, RedirectURI, CodeChallenge, CodeChallengeMethod string
	User                                                      *auth.User
	// Spaces the user let the app read: every space, or just these.
	AllSpaces bool
	SpaceIDs  []int64
	// Permissions the user chose, grants on auth.MCPResources. Empty means read
	// on all of them: write and delete are never given by default.
	Permissions []string
}

type codeEntry struct {
	clientID, redirectURI, challenge string
	user                             *auth.User
	spaces                           spaceGrant
	expires                          time.Time
}

// spaceGrant is which spaces a connection reaches and what it may do there. It is fixed when the user
// approves and carried through every refresh.
type spaceGrant struct {
	all   bool
	ids   []int64
	perms []string
}

// Service is shared by all requests: authorization codes are short-lived and
// kept in memory, so a restart only cancels logins that are in flight.
type Service struct {
	DB     store.DB
	Tokens auth.APITokens
	Users  auth.Users
	// Now is the clock; tests move it.
	Now func() time.Time

	mu       sync.Mutex
	codes    map[string]codeEntry
	regUntil time.Time // registration is closed unless an admin opened it; kept in memory, so a restart closes it
}

func New(db store.DB) *Service {
	return &Service{DB: db, Tokens: auth.APITokens{DB: db}, Users: auth.Users{DB: db}, Now: func() time.Time { return time.Now().UTC() }, codes: map[string]codeEntry{}}
}

func (s *Service) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// OpenRegistration lets new apps register for d and returns when it ends.
func (s *Service) OpenRegistration(d time.Duration) time.Time {
	until := s.Now().Add(d)
	s.mu.Lock()
	s.regUntil = until
	s.mu.Unlock()
	return until
}

func (s *Service) CloseRegistration() {
	s.mu.Lock()
	s.regUntil = time.Time{}
	s.mu.Unlock()
}

// RegistrationOpenUntil reports whether registration is open and until when.
func (s *Service) RegistrationOpenUntil() (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.regUntil, s.Now().Before(s.regUntil)
}

// Register stores a new public client (dynamic client registration). It only
// works while an admin has opened registration, and the first success closes it,
// so a stranger cannot register an app that looks like a trusted one. info (who
// called) is kept and logged, and the admin sees it in the list of apps.
func (s *Service) Register(ctx context.Context, name string, redirectURIs []string, info RequestInfo) (*Client, error) {
	if _, open := s.RegistrationOpenUntil(); !open {
		return nil, fail("access_denied", "Registration is not available.")
	}
	if err := validateRedirects(redirectURIs); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}
	if r := []rune(name); len(r) > maxNameLen {
		name = string(r[:maxNameLen])
	}
	info = RequestInfo{IP: clip(info.IP, maxIPLen), UserAgent: clip(info.UserAgent, maxUALen), Origin: clip(info.Origin, maxOriginLen)}
	now := s.Now()
	n, err := s.repo().CountClients(ctx)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if n >= MaxClients {
		if err := s.repo().PruneClients(ctx, now.Add(-idleClientTTL)); err != nil {
			return nil, fail("server_error", err.Error())
		}
		if n, err = s.repo().CountClients(ctx); err != nil {
			return nil, fail("server_error", err.Error())
		}
		if n >= MaxClients {
			return nil, fail("server_error", "Too many registered clients.")
		}
	}
	id := auth.RandomString(32)
	if err := s.repo().InsertClient(ctx, id, name, redirectURIs, info, now); err != nil {
		return nil, fail("server_error", err.Error())
	}
	slog.Info("oauth app registered", "client_id", id, "name", name,
		"ip", info.IP, "user_agent", info.UserAgent, "origin", info.Origin, "redirect_uris", redirectURIs)
	// One window, one app: the admin opens it again for the next one.
	s.CloseRegistration()
	return &Client{ID: id, Name: name, RedirectURIs: redirectURIs, Request: info, CreatedAt: now}, nil
}

func clip(v string, n int) string {
	if r := []rune(v); len(r) > n {
		return string(r[:n])
	}
	return v
}

// Clients lists every registered app, newest first, for the admin.
func (s *Service) Clients(ctx context.Context) ([]Client, error) {
	return s.repo().Clients(ctx)
}

// Remove disconnects an app: the client, its refresh tokens and the
// access tokens issued to it are deleted at once.
func (s *Service) Remove(ctx context.Context, clientID string, admin *auth.User) (bool, error) {
	var ok bool
	err := store.Tx(ctx, s.DB, func(tx store.DB) (err error) {
		ok, err = repo.Repo{DB: tx}.DeleteClient(ctx, clientID)
		return err
	})
	if ok {
		slog.Info("oauth app removed", "client_id", clientID, "by", admin.Email)
	}
	return ok, err
}
func validateRedirects(uris []string) error {
	if len(uris) == 0 || len(uris) > maxRedirects {
		return fail("invalid_redirect_uri", "Give between 1 and 10 redirect URIs.")
	}
	for _, raw := range uris {
		u, err := url.Parse(raw)
		if err != nil || len(raw) > maxURILen || u.Host == "" || u.Fragment != "" || u.User != nil {
			return fail("invalid_redirect_uri", "Redirect URIs must be absolute and have no fragment.")
		}
		switch u.Scheme {
		case "https":
		case "http":
			if !loopback(u.Hostname()) {
				return fail("invalid_redirect_uri", "Plain http redirect URIs are only allowed for localhost.")
			}
		default:
			return fail("invalid_redirect_uri", "Redirect URIs must use https.")
		}
	}
	return nil
}

func loopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// Client finds a registered client, nil when unknown.
func (s *Service) Client(ctx context.Context, id string) (*Client, error) {
	return s.repo().Client(ctx, id)
}

// CheckAuthorize validates the parts of an authorization request that decide
// whether an error may be sent back to the client at all: the client and its
// exact redirect URI. Anything wrong here must be shown to the user instead.
func (s *Service) CheckAuthorize(ctx context.Context, clientID, redirectURI string) (*Client, error) {
	c, err := s.repo().Client(ctx, clientID)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if c == nil {
		return nil, fail("invalid_client", "Unknown client.")
	}
	if !contains(c.RedirectURIs, redirectURI) {
		return nil, fail("invalid_request", "The redirect URI is not registered for this client.")
	}
	return c, nil
}

// IssueCode turns the user's approval into a single-use authorization code.
func (s *Service) IssueCode(ctx context.Context, req CodeRequest) (string, error) {
	if _, err := s.CheckAuthorize(ctx, req.ClientID, req.RedirectURI); err != nil {
		return "", err
	}
	if req.CodeChallengeMethod != "S256" || !validChallenge(req.CodeChallenge) {
		return "", fail("invalid_request", "PKCE with code_challenge_method=S256 is required.")
	}
	if req.User == nil {
		return "", fail("invalid_request", "No user.")
	}
	if !req.AllSpaces && len(req.SpaceIDs) == 0 {
		return "", fail("invalid_request", "Choose at least one space.")
	}
	perms, err := mcpPermissions(req.Permissions)
	if err != nil {
		return "", fail("invalid_request", err.Error())
	}
	now := s.Now()
	code := auth.RandomString(48)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range s.codes {
		if !now.Before(e.expires) {
			delete(s.codes, k)
		}
	}
	s.codes[auth.HashToken(code)] = codeEntry{
		clientID: req.ClientID, redirectURI: req.RedirectURI, challenge: req.CodeChallenge, user: req.User,
		spaces: spaceGrant{all: req.AllSpaces, ids: req.SpaceIDs, perms: perms}, expires: now.Add(codeTTL),
	}
	return code, nil
}

// Exchange trades an authorization code and its PKCE verifier for tokens. The
// code is spent whether or not the exchange succeeds.
func (s *Service) Exchange(ctx context.Context, clientID, code, verifier, redirectURI string) (*TokenSet, error) {
	key := auth.HashToken(code)
	s.mu.Lock()
	e, ok := s.codes[key]
	delete(s.codes, key)
	s.mu.Unlock()
	bad := fail("invalid_grant", "The authorization code is invalid or expired.")
	if !ok || !s.Now().Before(e.expires) || e.clientID != clientID || e.redirectURI != redirectURI {
		return nil, bad
	}
	if !validVerifier(verifier) || subtle.ConstantTimeCompare([]byte(s256(verifier)), []byte(e.challenge)) != 1 {
		return nil, bad
	}
	c, err := s.repo().Client(ctx, clientID)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if c == nil {
		return nil, bad
	}
	access, tokenID, err := s.issueAccess(ctx, e.user, c, e.spaces)
	if err != nil {
		return nil, err
	}
	refresh := refreshPrefix + auth.RandomString(48)
	now := s.Now()
	err = s.repo().InsertRefresh(ctx, repo.NewRefresh{
		Hash: auth.HashToken(refresh), ClientID: clientID, UserID: e.user.ID, APITokenID: tokenID,
		ExpiresAt: now.Add(RefreshTTL), Now: now,
	})
	if err != nil {
		_ = s.Tokens.Revoke(ctx, e.user.ID, tokenID)
		return nil, fail("server_error", err.Error())
	}
	_ = s.repo().DeleteExpiredRefresh(ctx, now)
	return &TokenSet{AccessToken: access, RefreshToken: refresh, ExpiresIn: int(AccessTTL / time.Second)}, nil
}

// Refresh trades a refresh token for a new pair and retires the old one.
func (s *Service) Refresh(ctx context.Context, clientID, refreshToken string) (*TokenSet, error) {
	bad := fail("invalid_grant", "The refresh token is invalid or expired.")
	oldHash := auth.HashToken(refreshToken)
	row, err := s.repo().RefreshByHash(ctx, oldHash)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if row == nil || row.ClientID != clientID {
		return nil, bad
	}
	now := s.Now()
	if !now.Before(row.ExpiresAt) {
		_ = s.repo().DeleteRefresh(ctx, row.ID)
		_ = s.Tokens.Revoke(ctx, row.UserID, row.APITokenID)
		return nil, bad
	}
	user, err := s.Users.ByID(ctx, row.UserID)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	c, err := s.repo().Client(ctx, clientID)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if user == nil || c == nil {
		return nil, bad
	}
	prev, err := s.accessToken(ctx, row.UserID, row.APITokenID)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if prev == nil {
		return nil, bad
	}
	access, tokenID, err := s.issueAccess(ctx, user, c, spaceGrant{all: prev.AllSpaces, ids: prev.SpaceIDs, perms: prev.Permissions})
	if errors.Is(err, auth.ErrBadGrant) {
		// Every space the connection could reach is gone.
		return nil, bad
	}
	if err != nil {
		return nil, err
	}
	next := refreshPrefix + auth.RandomString(48)
	ok, err := s.repo().RotateRefresh(ctx, row.ID, oldHash, auth.HashToken(next), tokenID, now.Add(RefreshTTL))
	if err != nil || !ok {
		_ = s.Tokens.Revoke(ctx, user.ID, tokenID)
		if err != nil {
			return nil, fail("server_error", err.Error())
		}
		return nil, bad
	}
	_ = s.Tokens.Revoke(ctx, user.ID, row.APITokenID)
	return &TokenSet{AccessToken: access, RefreshToken: next, ExpiresIn: int(AccessTTL / time.Second)}, nil
}

// accessToken is the user's API token with that id, nil when it is gone.
func (s *Service) accessToken(ctx context.Context, userID, id int64) (*auth.APIToken, error) {
	list, err := s.Tokens.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i], nil
		}
	}
	return nil, nil
}

// mcpPermissions checks what the user let an app do: grants on the resources
// the MCP tools use, at any level the user ticked. None given means read on
// all of them; write and delete are always a deliberate choice.
func mcpPermissions(in []string) ([]string, error) {
	if len(in) == 0 {
		return auth.ReadPerms(auth.MCPResources...), nil
	}
	perms, err := auth.NormalizePermissions(in)
	if err != nil {
		return nil, err
	}
	for _, p := range perms {
		resource, _, _ := strings.Cut(p, ":")
		if !slices.Contains(auth.MCPResources, resource) {
			return nil, fmt.Errorf("an app can only be given access to accounts, budgets, categories, currencies, debts, reports, tags and transactions, not %q", p)
		}
	}
	return perms, nil
}

// issueAccess creates the API token an MCP client calls with: the read access
// the user chose, in the spaces they chose, and nothing else.
func (s *Service) issueAccess(ctx context.Context, user *auth.User, c *Client, spaces spaceGrant) (raw string, id int64, err error) {
	exp := s.Now().Add(AccessTTL)
	grant := auth.Grant{Permissions: spaces.perms, AllSpaces: spaces.all, SpaceIDs: spaces.ids}
	raw, tok, err := s.Tokens.Issue(ctx, user, "MCP: "+c.Name, grant, &exp)
	if errors.Is(err, auth.ErrTokenLimit) {
		return "", 0, fail("server_error", "Too many API tokens: revoke some in Settings, then connect again.")
	}
	if errors.Is(err, auth.ErrBadGrant) {
		return "", 0, err
	}
	if err != nil {
		return "", 0, fail("server_error", err.Error())
	}
	return raw, tok.ID, nil
}

func s256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// validVerifier is RFC 7636 §4.1: 43 to 128 unreserved characters.
func validVerifier(v string) bool {
	if len(v) < 43 || len(v) > 128 {
		return false
	}
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.', r == '_', r == '~':
		default:
			return false
		}
	}
	return true
}

// validChallenge: a base64url SHA-256 is 43 characters.
func validChallenge(c string) bool {
	if len(c) != 43 {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(c)
	return err == nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
