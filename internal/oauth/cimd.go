package oauth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/oauth"
)

const (
	cimdMinTTL     = 5 * time.Minute
	cimdMaxTTL     = 24 * time.Hour
	cimdDefaultTTL = time.Hour
)

// Fetcher downloads a client metadata document. The production
// implementation is internal/safehttp; this package must not import net/http.
type Fetcher interface {
	Fetch(ctx context.Context, rawURL string) (body []byte, cacheControl string, err error)
}

var hostRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)

// CIMDHosts lists the hosts allowed to serve client metadata documents.
// Empty means client ID metadata documents are off.
func (s *Service) CIMDHosts(ctx context.Context) ([]string, error) {
	return s.repo().CIMDHosts(ctx)
}

// AddCIMDHost allows documents served from host (exact host name: no scheme,
// port, path or wildcard).
func (s *Service) AddCIMDHost(ctx context.Context, host string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if _, err := netip.ParseAddr(host); err == nil || !hostRE.MatchString(host) || len(host) > 253 {
		return fail("invalid_request", "Enter a bare host name such as claude.ai.")
	}
	return s.repo().AddCIMDHost(ctx, host, s.Now())
}

// RemoveCIMDHost stops trusting host; clients already cached from it stop
// working at once.
func (s *Service) RemoveCIMDHost(ctx context.Context, host string) (bool, error) {
	return s.repo().RemoveCIMDHost(ctx, strings.ToLower(strings.TrimSpace(host)))
}

// cimdURL parses a client_id that is a metadata document URL; nil when the
// id is an ordinary (non-URL) client id or an unusable URL.
func isCIMDID(id string) bool {
	return strings.HasPrefix(id, "https://") || strings.HasPrefix(id, "http://")
}

func parseCIMDURL(id string) *url.URL {
	u, err := url.Parse(id)
	if err != nil || len(id) > maxURILen || u.Scheme != "https" || u.Hostname() == "" ||
		u.User != nil || u.Fragment != "" || u.Path == "" || u.Path == "/" {
		return nil
	}
	if _, err := netip.ParseAddr(u.Hostname()); err == nil {
		return nil
	}
	for _, seg := range strings.Split(u.Path, "/") {
		if seg == "." || seg == ".." {
			return nil
		}
	}
	return u
}

// resolve finds a client by id. CIMD clients must come from an allowed host
// (checked every time, so removing a host cuts off its apps); with fetch their
// cached document is refreshed when stale.
func (s *Service) resolve(ctx context.Context, id string, fetch bool) (*Client, error) {
	if !isCIMDID(id) {
		c, err := s.repo().Client(ctx, id)
		if err != nil {
			return nil, fail("server_error", err.Error())
		}
		return c, nil
	}
	u := parseCIMDURL(id)
	if u == nil {
		return nil, nil
	}
	ok, err := s.repo().CIMDHostAllowed(ctx, strings.ToLower(u.Hostname()))
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if !ok {
		return nil, nil
	}
	c, err := s.repo().Client(ctx, id)
	if err != nil {
		return nil, fail("server_error", err.Error())
	}
	if c != nil && c.Kind != repo.KindCIMD {
		return nil, nil
	}
	if c != nil && (!fetch || s.Now().Before(c.ExpiresAt)) {
		return c, nil
	}
	if !fetch {
		return nil, nil
	}
	return s.fetchCIMD(ctx, id)
}

type cimdDocument struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
	AuthMethod   string   `json:"token_endpoint_auth_method"`
}

func (s *Service) fetchCIMD(ctx context.Context, id string) (*Client, error) {
	bad := func(why string) (*Client, error) {
		slog.Warn("oauth client metadata rejected", "client_id", id, "reason", why)
		return nil, fail("invalid_client", "The client metadata document could not be used.")
	}
	if s.Fetcher == nil {
		return bad("no fetcher")
	}
	body, cacheControl, err := s.Fetcher.Fetch(ctx, id)
	if err != nil {
		return bad(err.Error())
	}
	var d cimdDocument
	if err := json.Unmarshal(body, &d); err != nil {
		return bad("not json")
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	for _, k := range []string{"client_secret", "client_secret_expires_at", "jwks", "jwks_uri"} {
		if _, has := raw[k]; has {
			return bad("unsupported field " + k)
		}
	}
	if d.ClientID != id {
		return bad("client_id does not match the document URL")
	}
	if d.AuthMethod != "" && d.AuthMethod != "none" {
		return bad("token_endpoint_auth_method must be none")
	}
	if err := validateRedirects(d.RedirectURIs); err != nil {
		return bad(err.Error())
	}
	name := strings.TrimSpace(d.ClientName)
	if name == "" {
		name = parseCIMDURL(id).Hostname()
	}
	if r := []rune(name); len(r) > maxNameLen {
		name = string(r[:maxNameLen])
	}
	now := s.Now()
	if err := s.makeRoom(ctx, id, now); err != nil {
		return nil, err
	}
	if err := s.repo().UpsertCIMDClient(ctx, id, name, d.RedirectURIs, now.Add(cimdTTL(cacheControl)), now); err != nil {
		return nil, fail("server_error", err.Error())
	}
	return s.repo().Client(ctx, id)
}

// makeRoom enforces MaxClients for a new client, pruning idle ones first.
// An id that is already stored needs no room.
func (s *Service) makeRoom(ctx context.Context, id string, now time.Time) error {
	if c, err := s.repo().Client(ctx, id); err != nil {
		return fail("server_error", err.Error())
	} else if c != nil {
		return nil
	}
	n, err := s.repo().CountClients(ctx)
	if err != nil {
		return fail("server_error", err.Error())
	}
	if n < MaxClients {
		return nil
	}
	if err := s.repo().PruneClients(ctx, now.Add(-idleClientTTL)); err != nil {
		return fail("server_error", err.Error())
	}
	if n, err = s.repo().CountClients(ctx); err != nil {
		return fail("server_error", err.Error())
	}
	if n >= MaxClients {
		return fail("server_error", "Too many registered clients.")
	}
	return nil
}

// cimdTTL is how long a fetched document is trusted: its max-age, kept
// between 5 minutes and a day.
func cimdTTL(cacheControl string) time.Duration {
	ttl := cimdDefaultTTL
	for _, part := range strings.Split(strings.ToLower(cacheControl), ",") {
		part = strings.TrimSpace(part)
		if v, ok := strings.CutPrefix(part, "max-age="); ok {
			if n, err := strconv.Atoi(strings.Trim(v, `"`)); err == nil && n >= 0 {
				ttl = time.Duration(n) * time.Second
			}
		}
	}
	return min(max(ttl, cimdMinTTL), cimdMaxTTL)
}

// CreateClient registers a client for an admin. A confidential client gets a
// generated secret, returned once and kept only as a hash.
func (s *Service) CreateClient(
	ctx context.Context, name string, redirectURIs []string, confidential bool, admin *auth.User,
) (*Client, string, error) {
	if err := validateRedirects(redirectURIs); err != nil {
		return nil, "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}
	if r := []rune(name); len(r) > maxNameLen {
		name = string(r[:maxNameLen])
	}
	now := s.Now()
	if err := s.makeRoom(ctx, "", now); err != nil {
		return nil, "", err
	}
	id := auth.RandomString(32)
	secret, hash := "", ""
	if confidential {
		secret = auth.RandomString(48)
		hash = auth.HashToken(secret)
	}
	if err := s.repo().InsertManualClient(ctx, id, name, redirectURIs, hash, now); err != nil {
		return nil, "", fail("server_error", err.Error())
	}
	slog.Info("oauth client created", "client_id", id, "name", name, "confidential", confidential, "by", admin.Email)
	c, err := s.repo().Client(ctx, id)
	if err != nil {
		return nil, "", fail("server_error", err.Error())
	}
	return c, secret, nil
}

// AuthenticateClient checks the secret a client presented at the token
// endpoint. Public clients (no stored secret) pass without one. Unknown
// clients and wrong or missing secrets are invalid_client.
func (s *Service) AuthenticateClient(ctx context.Context, clientID, secret string) (*Client, error) {
	c, err := s.resolve(ctx, clientID, false)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fail("invalid_client", "Unknown client.")
	}
	if c.SecretHash == "" {
		return c, nil
	}
	if secret == "" || subtle.ConstantTimeCompare([]byte(auth.HashToken(secret)), []byte(c.SecretHash)) != 1 {
		return nil, fail("invalid_client", "Client authentication failed.")
	}
	return c, nil
}
