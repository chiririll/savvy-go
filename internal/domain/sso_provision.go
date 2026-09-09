package domain

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/chiririll/savvy-plus/internal/auth"
)

func (s SSO) Provision(ctx context.Context, p IdentityProvider, id NormalizedIdentity) (*auth.User, bool, error) {
	if id.Subject == "" {
		return nil, false, ssoErr("missing_subject", "The identity is missing a subject.", 422)
	}
	if u, ok, err := s.loginExisting(ctx, p, id); err != nil || ok {
		return u, u != nil && u.HasTwoFactor(), err
	}
	n, err := s.Users.Count(ctx)
	if err != nil {
		return nil, false, err
	}
	if n == 0 {
		return nil, false, ssoErr("no_admin", "Create a local administrator before signing in with SSO.", 403)
	}
	user, err := s.linkByEmail(ctx, p, id)
	if err != nil {
		return nil, false, err
	}
	if user == nil {
		user, err = s.justInTime(ctx, p, id)
		if err != nil {
			return nil, false, err
		}
	}
	if err := s.createLink(ctx, p, id, user.ID); err != nil {
		return nil, false, err
	}
	return user, user.HasTwoFactor(), nil
}

func (s SSO) loginExisting(ctx context.Context, p IdentityProvider, id NormalizedIdentity) (*auth.User, bool, error) {
	var userID int64
	var linkID int64
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, user_id FROM user_identities WHERE identity_provider_id=? AND subject=?`,
		p.ID, id.Subject).Scan(&linkID, &userID)
	if err != nil {
		return nil, false, nil
	}
	user, err := s.Users.ByID(ctx, userID)
	if err != nil || user == nil {
		return nil, false, ssoErr("user_missing", "The linked user no longer exists.", 403)
	}
	if p.SyncRoleOnLogin {
		_ = s.syncRole(ctx, user, p, id)
		user, _ = s.Users.ByID(ctx, user.ID)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	raw, _ := json.Marshal(id.Raw)
	_, _ = s.DB.ExecContext(ctx, `UPDATE user_identities SET last_login_at=?, claims=?, updated_at=? WHERE id=?`,
		now, string(raw), now, linkID)
	return user, true, nil
}

func (s SSO) linkByEmail(ctx context.Context, p IdentityProvider, id NormalizedIdentity) (*auth.User, error) {
	if !p.LinkByEmail || id.Email == "" || !id.EmailVerified {
		return nil, nil
	}
	user, err := s.Users.ByEmail(ctx, id.Email)
	if err != nil || user == nil {
		return nil, err
	}
	if p.SyncRoleOnLogin {
		_ = s.syncRole(ctx, user, p, id)
		user, _ = s.Users.ByID(ctx, user.ID)
	}
	return user, nil
}

func (s SSO) justInTime(ctx context.Context, p IdentityProvider, id NormalizedIdentity) (*auth.User, error) {
	if !p.AllowJIT || !s.Settings.Bool(ctx, "sso_allow_signup", true) {
		return nil, ssoErr("signup_disabled", "Sign-up with this provider is disabled.", 403)
	}
	if id.Email == "" {
		return nil, ssoErr("no_email", "The identity provider did not return an email address.", 422)
	}
	if s.Settings.Bool(ctx, "sso_require_verified_email", false) && !id.EmailVerified {
		return nil, ssoErr("email_unverified", "The email address is not verified.", 422)
	}
	if existing, _ := s.Users.ByEmail(ctx, id.Email); existing != nil {
		return nil, ssoErr("email_in_use", "An account with this email already exists.", 409)
	}
	name := id.Name
	if name == "" {
		name = id.Email
	}
	role := s.mappedRole(p, id)
	if role == "" {
		role = p.DefaultRole
	}
	if role == "" {
		role = auth.RoleReadOnly
	}
	pass := auth.RandomString(40)
	user, err := s.Users.Create(ctx, name, id.Email, &pass, role)
	if err != nil {
		return nil, err
	}
	if err := s.Users.MarkSSOOnly(ctx, user.ID); err != nil {
		return nil, err
	}
	return s.Users.ByID(ctx, user.ID)
}

func (s SSO) createLink(ctx context.Context, p IdentityProvider, id NormalizedIdentity, userID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	raw, _ := json.Marshal(id.Raw)
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO user_identities (user_id, identity_provider_id, subject, last_login_at, claims, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?)`, userID, p.ID, id.Subject, now, string(raw), now, now)
	return err
}

func (s SSO) syncRole(ctx context.Context, user *auth.User, p IdentityProvider, id NormalizedIdentity) error {
	mapped := s.mappedRole(p, id)
	if mapped == "" || mapped == user.Role {
		return nil
	}
	if user.IsAdmin() && mapped != auth.RoleAdmin {
		n, _ := s.Users.AdminCount(ctx)
		if n <= 1 {
			return nil
		}
	}
	return s.Users.SetRole(ctx, user.ID, mapped)
}

func (s SSO) mappedRole(p IdentityProvider, id NormalizedIdentity) string {
	rules, _ := p.RoleMapping.([]any)
	if rules == nil {
		if typed, ok := p.RoleMapping.([]map[string]any); ok {
			for _, rule := range typed {
				if role := matchRoleRule(rule, id); role != "" {
					return role
				}
			}
		}
		return ""
	}
	for _, raw := range rules {
		rule, _ := raw.(map[string]any)
		if role := matchRoleRule(rule, id); role != "" {
			return role
		}
	}
	return ""
}

func matchRoleRule(rule map[string]any, id NormalizedIdentity) string {
	claim, _ := rule["claim"].(string)
	role, _ := rule["role"].(string)
	op, _ := rule["operator"].(string)
	if claim == "" || role == "" {
		return ""
	}
	if op == "" {
		op = "equals"
	}
	var actual any
	if claim == "groups" {
		actual = id.Groups
	} else if id.Raw != nil {
		actual = id.Raw[claim]
	}
	if ruleMatches(op, actual, rule["value"]) {
		switch role {
		case auth.RoleAdmin, auth.RoleReadWrite, auth.RoleReadOnly:
			return role
		}
	}
	return ""
}

func ruleMatches(op string, actual, value any) bool {
	switch op {
	case "equals":
		return stringify(actual) == stringify(value)
	case "contains":
		if arr, ok := asAnySlice(actual); ok {
			for _, v := range arr {
				if stringify(v) == stringify(value) {
					return true
				}
			}
			return false
		}
		return strings.Contains(stringify(actual), stringify(value))
	case "one_of":
		want, _ := asAnySlice(value)
		if want == nil {
			want = []any{value}
		}
		if arr, ok := asAnySlice(actual); ok {
			set := map[string]bool{}
			for _, v := range arr {
				set[stringify(v)] = true
			}
			for _, v := range want {
				if set[stringify(v)] {
					return true
				}
			}
			return false
		}
		for _, v := range want {
			if stringify(actual) == stringify(v) {
				return true
			}
		}
	}
	return false
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
	}
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), `"`)
}

func asAnySlice(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out, true
	}
	return nil, false
}
