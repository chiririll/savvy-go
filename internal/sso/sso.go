package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/sso"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string { return e.Message }

func ssoErr(code, msg string, status int) *Error {
	return &Error{Code: code, Message: msg, Status: status}
}

func AsError(err error) *Error {
	if e, ok := err.(*Error); ok {
		return e
	}
	return nil
}

type IdentityProvider struct {
	ID              int64
	Name            string
	Slug            string
	Protocol        string
	Preset          string
	Enabled         bool
	SortOrder       int
	Config          map[string]any
	Secrets         map[string]any
	ClaimMappings   map[string]any
	RoleMapping     any
	DefaultRole     string
	AllowJIT        bool
	SyncRoleOnLogin bool
	LinkByEmail     bool
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func (p IdentityProvider) JSON(appURL string) map[string]any {
	base := strings.TrimRight(appURL, "/")
	var created, updated any
	if p.CreatedAt != nil {
		created = p.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if p.UpdatedAt != nil {
		updated = p.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	cfg := p.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	return map[string]any{
		"id": p.ID, "name": p.Name, "slug": p.Slug, "protocol": p.Protocol, "preset": p.Preset,
		"enabled": p.Enabled, "sortOrder": p.SortOrder, "config": cfg,
		"claimMappings": p.ClaimMappings, "roleMapping": p.RoleMapping,
		"defaultRole": p.DefaultRole, "allowJit": p.AllowJIT,
		"syncRoleOnLogin": p.SyncRoleOnLogin, "linkByEmail": p.LinkByEmail,
		"hasClientSecret": secretFilled(p.Secrets, "client_secret"),
		"metadataUrl":     base + "/api/auth/sso/" + p.Slug + "/metadata",
		"callbackUrl":     base + "/api/auth/sso/" + p.Slug + "/callback",
		"acsUrl":          base + "/api/auth/sso/" + p.Slug + "/acs",
		"createdAt":       created, "updatedAt": updated,
	}
}

func (p IdentityProvider) PublicJSON() map[string]any {
	return map[string]any{"slug": p.Slug, "name": p.Name, "preset": p.Preset, "protocol": p.Protocol}
}

func secretFilled(secrets map[string]any, key string) bool {
	if secrets == nil {
		return false
	}
	v, ok := secrets[key]
	if !ok || v == nil {
		return false
	}
	s, _ := v.(string)
	return strings.TrimSpace(s) != ""
}

type IdPState struct {
	ProviderID    int64
	Nonce         string
	CodeVerifier  string
	SAMLRequestID string
	RedirectAfter string
}

type NormalizedIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Groups        []string
	Raw           map[string]any
}

type Service struct {
	DB       store.DB
	Users    auth.Users
	Settings settings.Store
	// AppURL returns the public URL of the instance, empty when unset.
	Cfg  appconfig.Config
	HTTP *http.Client
}

func (s Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s Service) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// AnyEnabled reports whether some identity provider is enabled.
func (s Service) AnyEnabled(ctx context.Context) bool {
	n, _ := s.repo().CountEnabled(ctx)
	return n > 0
}

func (s Service) All(ctx context.Context) ([]IdentityProvider, error) {
	return s.list(ctx, repo.Filter{})
}

func (s Service) Enabled(ctx context.Context) ([]IdentityProvider, error) {
	return s.list(ctx, repo.Filter{EnabledOnly: true})
}

func (s Service) ByID(ctx context.Context, id int64) (*IdentityProvider, error) {
	list, err := s.list(ctx, repo.Filter{ID: id})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Service) BySlug(ctx context.Context, slug string) (*IdentityProvider, error) {
	list, err := s.list(ctx, repo.Filter{Slug: slug})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

type IdPWrite struct {
	Name            string
	Slug            string
	Preset          string
	Enabled         *bool
	SortOrder       *int
	Fields          map[string]any
	ClaimMappings   map[string]any
	RoleMapping     any
	DefaultRole     string
	AllowJIT        *bool
	SyncRoleOnLogin *bool
	LinkByEmail     *bool
}

var slugRE = regexp.MustCompile(`^[a-z0-9-]+$`)

func (s Service) Create(ctx context.Context, in IdPWrite) (*IdentityProvider, error) {
	preset, ok := PresetByKey(in.Preset)
	if !ok {
		return nil, ssoErr("unknown_preset", "Unknown Service preset: "+in.Preset, 404)
	}
	if !slugRE.MatchString(in.Slug) {
		return nil, ssoErr("invalid_slug", "The slug format is invalid.", 422)
	}
	if existing, _ := s.BySlug(ctx, in.Slug); existing != nil {
		return nil, ssoErr("slug_taken", "The slug has already been taken.", 422)
	}
	cfg, sec, err := assembleFields(preset, in.Fields, nil)
	if err != nil {
		return nil, err
	}
	enabled := false
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	sort := 0
	if in.SortOrder != nil {
		sort = *in.SortOrder
	}
	allowJIT, link := true, true
	if in.AllowJIT != nil {
		allowJIT = *in.AllowJIT
	}
	if in.LinkByEmail != nil {
		link = *in.LinkByEmail
	}
	sync := false
	if in.SyncRoleOnLogin != nil {
		sync = *in.SyncRoleOnLogin
	}
	role := in.DefaultRole
	if role == "" {
		role = "read-only"
	}
	claims := in.ClaimMappings
	if claims == nil {
		claims = preset.DefaultClaims
	}
	roles := in.RoleMapping
	if roles == nil {
		roles = []any{}
	}
	cfgJSON, _ := json.Marshal(cfg)
	secJSON, _ := json.Marshal(sec)
	claimJSON, _ := json.Marshal(claims)
	roleJSON, _ := json.Marshal(roles)
	id, err := s.repo().Insert(ctx, repo.Save{
		Name: in.Name, Slug: in.Slug, Protocol: preset.Protocol, Preset: preset.Key,
		Enabled: enabled, SortOrder: sort, Config: string(cfgJSON), Secrets: string(secJSON),
		ClaimMappings: string(claimJSON), RoleMapping: string(roleJSON),
		DefaultRole: role, AllowJIT: allowJIT, SyncRoleOnLogin: sync, LinkByEmail: link,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Service) Update(ctx context.Context, id int64, in IdPWrite) (*IdentityProvider, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	preset, ok := PresetByKey(cur.Preset)
	if !ok {
		return nil, ssoErr("unknown_preset", "Unknown Service preset: "+cur.Preset, 404)
	}
	if in.Name != "" {
		cur.Name = in.Name
	}
	if in.Slug != "" {
		if !slugRE.MatchString(in.Slug) {
			return nil, ssoErr("invalid_slug", "The slug format is invalid.", 422)
		}
		if other, _ := s.BySlug(ctx, in.Slug); other != nil && other.ID != id {
			return nil, ssoErr("slug_taken", "The slug has already been taken.", 422)
		}
		cur.Slug = in.Slug
	}
	if in.Enabled != nil {
		cur.Enabled = *in.Enabled
	}
	if in.SortOrder != nil {
		cur.SortOrder = *in.SortOrder
	}
	if in.DefaultRole != "" {
		cur.DefaultRole = in.DefaultRole
	}
	if in.AllowJIT != nil {
		cur.AllowJIT = *in.AllowJIT
	}
	if in.SyncRoleOnLogin != nil {
		cur.SyncRoleOnLogin = *in.SyncRoleOnLogin
	}
	if in.LinkByEmail != nil {
		cur.LinkByEmail = *in.LinkByEmail
	}
	if in.ClaimMappings != nil {
		cur.ClaimMappings = in.ClaimMappings
	}
	if in.RoleMapping != nil {
		cur.RoleMapping = in.RoleMapping
	}
	if in.Fields != nil {
		cfg, sec, err := assembleFields(preset, in.Fields, cur)
		if err != nil {
			return nil, err
		}
		cur.Config, cur.Secrets = cfg, sec
	}
	cfgJSON, _ := json.Marshal(cur.Config)
	secJSON, _ := json.Marshal(cur.Secrets)
	claimJSON, _ := json.Marshal(cur.ClaimMappings)
	roleJSON, _ := json.Marshal(cur.RoleMapping)
	err = s.repo().Update(ctx, id, repo.Save{
		Name: cur.Name, Slug: cur.Slug, Protocol: cur.Protocol, Preset: cur.Preset,
		Enabled: cur.Enabled, SortOrder: cur.SortOrder, Config: string(cfgJSON), Secrets: string(secJSON),
		ClaimMappings: string(claimJSON), RoleMapping: string(roleJSON),
		DefaultRole: cur.DefaultRole, AllowJIT: cur.AllowJIT, SyncRoleOnLogin: cur.SyncRoleOnLogin, LinkByEmail: cur.LinkByEmail,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Service) Delete(ctx context.Context, id int64) error {
	orphans, _ := s.repo().SSOOnlyOrphans(ctx, id)
	if orphans > 0 {
		return fmt.Errorf("Cannot delete: Service-only users would be left without a way to sign in.")
	}
	return s.repo().Delete(ctx, id)
}

func assembleFields(
	preset Preset,
	input map[string]any,
	existing *IdentityProvider,
) (cfg, sec map[string]any, err error) {
	cfg, sec = map[string]any{}, map[string]any{}
	if input == nil {
		input = map[string]any{}
	}
	for _, f := range preset.Fields {
		raw := input[f.Key]
		if f.Key == "scopes" {
			if s, ok := raw.(string); ok {
				parts := strings.Fields(s)
				var out []any
				for _, p := range parts {
					out = append(out, p)
				}
				raw = out
			}
		}
		if f.Secret {
			if filled(raw) {
				sec[f.Key] = raw
			} else if existing != nil && secretFilled(existing.Secrets, f.Key) {
				sec[f.Key] = existing.Secrets[f.Key]
			}
			continue
		}
		cfg[f.Key] = raw
	}
	for _, f := range preset.Fields {
		if !f.Required {
			continue
		}
		ok := false
		if f.Secret {
			ok = secretFilled(sec, f.Key)
		} else {
			ok = filled(cfg[f.Key])
		}
		if !ok {
			return nil, nil, ssoErr("missing_field", "Missing required field: "+f.Label+".", 422)
		}
	}
	return cfg, sec, nil
}

func filled(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t) != ""
	case []any:
		return len(t) > 0
	default:
		return true
	}
}

func (s Service) IssueTicket(ctx context.Context, userID int64, requires2FA bool) (string, error) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	ticket := hex.EncodeToString(b)
	now := time.Now().UTC()
	err := s.repo().InsertTicket(ctx, ticket, userID, requires2FA, now.Add(2*time.Minute), now)
	return ticket, err
}

func (s Service) ConsumeTicket(ctx context.Context, ticket string) (userID int64, requires2FA bool, ok bool) {
	userID, requires2FA, ok, err := s.repo().TakeTicket(ctx, ticket, time.Now().UTC())
	if err != nil || !ok {
		return 0, false, false
	}
	return userID, requires2FA, true
}

func (s Service) SaveState(ctx context.Context, st IdPState) (string, error) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)
	now := time.Now().UTC()
	err := s.repo().InsertState(ctx, state, repo.State(st), now.Add(10*time.Minute), now)
	return state, err
}

func (s Service) AttachSAMLRequestID(ctx context.Context, state, requestID string) error {
	return s.repo().AttachSAMLRequestID(ctx, state, requestID)
}

func (s Service) ConsumeState(ctx context.Context, state string) (*IdPState, error) {
	st, _ := s.repo().TakeState(ctx, state, time.Now().UTC())
	if st == nil {
		return nil, nil
	}
	out := IdPState(*st)
	return &out, nil
}

func (s Service) list(ctx context.Context, f repo.Filter) ([]IdentityProvider, error) {
	rows, err := s.repo().List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]IdentityProvider, len(rows))
	for i, r := range rows {
		out[i] = idpFrom(r)
	}
	return out, nil
}

func idpFrom(row repo.Provider) IdentityProvider {
	p := IdentityProvider{
		ID: row.ID, Name: row.Name, Slug: row.Slug, Protocol: row.Protocol, Preset: row.Preset,
		Enabled: row.Enabled, SortOrder: row.SortOrder, DefaultRole: row.DefaultRole,
		AllowJIT: row.AllowJIT, SyncRoleOnLogin: row.SyncRoleOnLogin, LinkByEmail: row.LinkByEmail,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &p.Config)
	}
	if row.Secrets != "" {
		_ = json.Unmarshal([]byte(row.Secrets), &p.Secrets)
	}
	if row.ClaimMappings != "" {
		_ = json.Unmarshal([]byte(row.ClaimMappings), &p.ClaimMappings)
	}
	if row.RoleMapping != "" {
		_ = json.Unmarshal([]byte(row.RoleMapping), &p.RoleMapping)
	}
	return p
}

func cfgString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return strings.TrimSpace(v)
}

func secretString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
