package domain

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/settings"
)

type SSOError struct {
	Code    string
	Message string
	Status  int
}

func (e *SSOError) Error() string { return e.Message }

func ssoErr(code, msg string, status int) *SSOError {
	return &SSOError{Code: code, Message: msg, Status: status}
}

func AsSSOError(err error) *SSOError {
	if e, ok := err.(*SSOError); ok {
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

type SSO struct {
	DB       *sql.DB
	Users    auth.Users
	Settings settings.Store
	AppURL   string
	HTTP     *http.Client
}

func (s SSO) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s SSO) All(ctx context.Context) ([]IdentityProvider, error) {
	return s.list(ctx, sqlc.ListIdentityProvidersParams{})
}

func (s SSO) Enabled(ctx context.Context) ([]IdentityProvider, error) {
	return s.list(ctx, sqlc.ListIdentityProvidersParams{EnabledOnly: db.Flag(true)})
}

func (s SSO) ByID(ctx context.Context, id int64) (*IdentityProvider, error) {
	list, err := s.list(ctx, sqlc.ListIdentityProvidersParams{ID: db.NI(id)})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s SSO) BySlug(ctx context.Context, slug string) (*IdentityProvider, error) {
	list, err := s.list(ctx, sqlc.ListIdentityProvidersParams{Slug: db.NullStringVal(slug)})
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

func (s SSO) Create(ctx context.Context, in IdPWrite) (*IdentityProvider, error) {
	preset, ok := PresetByKey(in.Preset)
	if !ok {
		return nil, ssoErr("unknown_preset", "Unknown SSO preset: "+in.Preset, 404)
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
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).InsertIdentityProvider(ctx, sqlc.InsertIdentityProviderParams{
		Name: in.Name, Slug: in.Slug, Protocol: preset.Protocol, Preset: preset.Key,
		Enabled: db.BoolInt(enabled), SortOrder: int64(sort),
		Config: db.NS(string(cfgJSON)), Secrets: db.NS(string(secJSON)),
		ClaimMappings: db.NS(string(claimJSON)), RoleMapping: db.NS(string(roleJSON)),
		DefaultRole: role, AllowJit: db.BoolInt(allowJIT), SyncRoleOnLogin: db.BoolInt(sync),
		LinkByEmail: db.BoolInt(link), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s SSO) Update(ctx context.Context, id int64, in IdPWrite) (*IdentityProvider, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	preset, ok := PresetByKey(cur.Preset)
	if !ok {
		return nil, ssoErr("unknown_preset", "Unknown SSO preset: "+cur.Preset, 404)
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
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateIdentityProvider(ctx, sqlc.UpdateIdentityProviderParams{
		Name: cur.Name, Slug: cur.Slug, Protocol: cur.Protocol, Preset: cur.Preset,
		Enabled: db.BoolInt(cur.Enabled), SortOrder: int64(cur.SortOrder),
		Config: db.NS(string(cfgJSON)), Secrets: db.NS(string(secJSON)),
		ClaimMappings: db.NS(string(claimJSON)), RoleMapping: db.NS(string(roleJSON)),
		DefaultRole: cur.DefaultRole, AllowJit: db.BoolInt(cur.AllowJIT),
		SyncRoleOnLogin: db.BoolInt(cur.SyncRoleOnLogin), LinkByEmail: db.BoolInt(cur.LinkByEmail),
		UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s SSO) Delete(ctx context.Context, id int64) error {
	orphans, _ := db.Q(s.DB).CountSSOOnlyOrphans(ctx, id)
	if orphans > 0 {
		return fmt.Errorf("Cannot delete: SSO-only users would be left without a way to sign in.")
	}
	return db.Q(s.DB).DeleteIdentityProvider(ctx, id)
}

func assembleFields(preset SSOPreset, input map[string]any, existing *IdentityProvider) (cfg, sec map[string]any, err error) {
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

func (s SSO) IssueTicket(ctx context.Context, userID int64, requires2FA bool) (string, error) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	ticket := hex.EncodeToString(b)
	now := time.Now().UTC()
	err := db.Q(s.DB).InsertSSOTicket(ctx, sqlc.InsertSSOTicketParams{
		Ticket: ticket, UserID: userID, Requires2fa: db.BoolInt(requires2FA),
		ExpiresAt: now.Add(2 * time.Minute).Format(time.RFC3339),
		CreatedAt: db.NS(now.Format(time.RFC3339)), UpdatedAt: db.NS(now.Format(time.RFC3339)),
	})
	return ticket, err
}

func (s SSO) ConsumeTicket(ctx context.Context, ticket string) (userID int64, requires2FA bool, ok bool) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Q(s.DB).ConsumeSSOTicket(ctx, sqlc.ConsumeSSOTicketParams{
		ConsumedAt: db.NS(now), UpdatedAt: db.NS(now), Ticket: ticket, ExpiresAt: now,
	})
	if err != nil {
		return 0, false, false
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return 0, false, false
	}
	row, err := db.Q(s.DB).GetSSOTicket(ctx, ticket)
	if err != nil {
		return 0, false, false
	}
	return row.UserID, row.Requires2fa != 0, true
}

func (s SSO) SaveState(ctx context.Context, st IdPState) (string, error) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)
	now := time.Now().UTC()
	err := db.Q(s.DB).InsertSSOState(ctx, sqlc.InsertSSOStateParams{
		State: state, IdentityProviderID: st.ProviderID,
		Nonce: db.NullStringVal(st.Nonce), CodeVerifier: db.NullStringVal(st.CodeVerifier),
		SamlRequestID: db.NullStringVal(st.SAMLRequestID), RedirectAfter: db.NullStringVal(st.RedirectAfter),
		ExpiresAt: now.Add(10 * time.Minute).Format(time.RFC3339),
		CreatedAt: db.NS(now.Format(time.RFC3339)), UpdatedAt: db.NS(now.Format(time.RFC3339)),
	})
	return state, err
}

func (s SSO) AttachSAMLRequestID(ctx context.Context, state, requestID string) error {
	return db.Q(s.DB).AttachSAMLRequestID(ctx, sqlc.AttachSAMLRequestIDParams{SamlRequestID: db.NS(requestID), State: state})
}

func (s SSO) ConsumeState(ctx context.Context, state string) (*IdPState, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	row, err := db.Q(s.DB).GetSSOState(ctx, state)
	if err != nil {
		return nil, nil
	}
	_ = db.Q(s.DB).DeleteSSOState(ctx, state)
	if row.ExpiresAt <= now {
		return nil, nil
	}
	return &IdPState{
		ProviderID: row.IdentityProviderID, Nonce: row.Nonce.String, CodeVerifier: row.CodeVerifier.String,
		SAMLRequestID: row.SamlRequestID.String, RedirectAfter: row.RedirectAfter.String,
	}, nil
}

func (s SSO) list(ctx context.Context, arg sqlc.ListIdentityProvidersParams) ([]IdentityProvider, error) {
	rows, err := db.Q(s.DB).ListIdentityProviders(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]IdentityProvider, 0, len(rows))
	for _, r := range rows {
		out = append(out, idpFrom(r))
	}
	return out, nil
}

func idpFrom(row sqlc.IdentityProvider) IdentityProvider {
	p := IdentityProvider{
		ID: row.ID, Name: row.Name, Slug: row.Slug, Protocol: row.Protocol, Preset: row.Preset,
		Enabled: row.Enabled != 0, SortOrder: int(row.SortOrder), DefaultRole: row.DefaultRole,
		AllowJIT: row.AllowJit != 0, SyncRoleOnLogin: row.SyncRoleOnLogin != 0, LinkByEmail: row.LinkByEmail != 0,
	}
	if row.Config.Valid {
		_ = json.Unmarshal([]byte(row.Config.String), &p.Config)
	}
	if row.Secrets.Valid {
		_ = json.Unmarshal([]byte(row.Secrets.String), &p.Secrets)
	}
	if row.ClaimMappings.Valid {
		_ = json.Unmarshal([]byte(row.ClaimMappings.String), &p.ClaimMappings)
	}
	if row.RoleMapping.Valid {
		_ = json.Unmarshal([]byte(row.RoleMapping.String), &p.RoleMapping)
	}
	if tm, ok := parseNullTime(row.CreatedAt); ok {
		p.CreatedAt = &tm
	}
	if tm, ok := parseNullTime(row.UpdatedAt); ok {
		p.UpdatedAt = &tm
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
