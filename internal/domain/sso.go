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

	"github.com/chiririll/savvy-plus/internal/auth"
	"github.com/chiririll/savvy-plus/internal/settings"
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
	return s.list(ctx, `ORDER BY sort_order, id`)
}

func (s SSO) Enabled(ctx context.Context) ([]IdentityProvider, error) {
	return s.list(ctx, `WHERE enabled = 1 ORDER BY sort_order, id`)
}

func (s SSO) ByID(ctx context.Context, id int64) (*IdentityProvider, error) {
	list, err := s.list(ctx, `WHERE id = ?`, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s SSO) BySlug(ctx context.Context, slug string) (*IdentityProvider, error) {
	list, err := s.list(ctx, `WHERE slug = ?`, slug)
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
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO identity_providers (name, slug, protocol, preset, enabled, sort_order, config, secrets,
			claim_mappings, role_mapping, default_role, allow_jit, sync_role_on_login, link_by_email, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		in.Name, in.Slug, preset.Protocol, preset.Key, boolInt(enabled), sort, string(cfgJSON), string(secJSON),
		string(claimJSON), string(roleJSON), role, boolInt(allowJIT), boolInt(sync), boolInt(link), now, now)
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
	_, err = s.DB.ExecContext(ctx, `
		UPDATE identity_providers SET name=?, slug=?, protocol=?, preset=?, enabled=?, sort_order=?, config=?, secrets=?,
			claim_mappings=?, role_mapping=?, default_role=?, allow_jit=?, sync_role_on_login=?, link_by_email=?, updated_at=?
		WHERE id=?`,
		cur.Name, cur.Slug, cur.Protocol, cur.Preset, boolInt(cur.Enabled), cur.SortOrder, string(cfgJSON), string(secJSON),
		string(claimJSON), string(roleJSON), cur.DefaultRole, boolInt(cur.AllowJIT), boolInt(cur.SyncRoleOnLogin), boolInt(cur.LinkByEmail), now, id)
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s SSO) Delete(ctx context.Context, id int64) error {
	var orphans int
	_ = s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM users u
		WHERE u.is_sso_only = 1
		  AND EXISTS (SELECT 1 FROM user_identities i WHERE i.user_id = u.id AND i.identity_provider_id = ?)
		  AND NOT EXISTS (SELECT 1 FROM user_identities i WHERE i.user_id = u.id AND i.identity_provider_id != ?)`,
		id, id).Scan(&orphans)
	if orphans > 0 {
		return fmt.Errorf("Cannot delete: SSO-only users would be left without a way to sign in.")
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM identity_providers WHERE id = ?`, id)
	return err
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
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO sso_login_tickets (ticket, user_id, requires_2fa, expires_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?)`, ticket, userID, boolInt(requires2FA), now.Add(2*time.Minute).Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	return ticket, err
}

func (s SSO) ConsumeTicket(ctx context.Context, ticket string) (userID int64, requires2FA bool, ok bool) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `
		UPDATE sso_login_tickets SET consumed_at=?, updated_at=?
		WHERE ticket=? AND consumed_at IS NULL AND expires_at > ?`, now, now, ticket, now)
	if err != nil {
		return 0, false, false
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return 0, false, false
	}
	var req int
	if err := s.DB.QueryRowContext(ctx, `SELECT user_id, requires_2fa FROM sso_login_tickets WHERE ticket=?`, ticket).Scan(&userID, &req); err != nil {
		return 0, false, false
	}
	return userID, req != 0, true
}

func (s SSO) SaveState(ctx context.Context, st IdPState) (string, error) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)
	now := time.Now().UTC()
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO sso_login_states (state, identity_provider_id, nonce, code_verifier, saml_request_id, redirect_after, expires_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?)`, state, st.ProviderID, nullIfEmpty(st.Nonce), nullIfEmpty(st.CodeVerifier),
		nullIfEmpty(st.SAMLRequestID), nullIfEmpty(st.RedirectAfter), now.Add(10*time.Minute).Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	return state, err
}

func (s SSO) AttachSAMLRequestID(ctx context.Context, state, requestID string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE sso_login_states SET saml_request_id=? WHERE state=?`, requestID, state)
	return err
}

func (s SSO) ConsumeState(ctx context.Context, state string) (*IdPState, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var row IdPState
	var nonce, verifier, saml, after sql.NullString
	var exp string
	err := s.DB.QueryRowContext(ctx, `
		SELECT identity_provider_id, nonce, code_verifier, saml_request_id, redirect_after, expires_at
		FROM sso_login_states WHERE state=?`, state).Scan(&row.ProviderID, &nonce, &verifier, &saml, &after, &exp)
	if err != nil {
		return nil, nil
	}
	_, _ = s.DB.ExecContext(ctx, `DELETE FROM sso_login_states WHERE state=?`, state)
	if exp <= now {
		return nil, nil
	}
	row.Nonce, row.CodeVerifier, row.SAMLRequestID, row.RedirectAfter = nonce.String, verifier.String, saml.String, after.String
	return &row, nil
}

func (s SSO) list(ctx context.Context, where string, args ...any) ([]IdentityProvider, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, name, slug, protocol, preset, enabled, sort_order, config, secrets,
			claim_mappings, role_mapping, default_role, allow_jit, sync_role_on_login, link_by_email, created_at, updated_at
		FROM identity_providers `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IdentityProvider
	for rows.Next() {
		p, err := scanIdP(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanIdP(row interface{ Scan(...any) error }) (IdentityProvider, error) {
	var p IdentityProvider
	var cfg, sec, claims, roles, created, updated sql.NullString
	var enabled, jit, sync, link int
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Protocol, &p.Preset, &enabled, &p.SortOrder, &cfg, &sec,
		&claims, &roles, &p.DefaultRole, &jit, &sync, &link, &created, &updated)
	p.Enabled = enabled != 0
	p.AllowJIT = jit != 0
	p.SyncRoleOnLogin = sync != 0
	p.LinkByEmail = link != 0
	if cfg.Valid {
		_ = json.Unmarshal([]byte(cfg.String), &p.Config)
	}
	if sec.Valid {
		_ = json.Unmarshal([]byte(sec.String), &p.Secrets)
	}
	if claims.Valid {
		_ = json.Unmarshal([]byte(claims.String), &p.ClaimMappings)
	}
	if roles.Valid {
		_ = json.Unmarshal([]byte(roles.String), &p.RoleMapping)
	}
	if tm, ok := parseNullTime(created); ok {
		p.CreatedAt = &tm
	}
	if tm, ok := parseNullTime(updated); ok {
		p.UpdatedAt = &tm
	}
	return p, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
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
