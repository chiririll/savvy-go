// Package apitokens serves the API tokens of the signed-in user.
package apitokens

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"strconv"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
	"savvy-go/internal/httpserver/httpx/confirm"
	"savvy-go/internal/httpserver/httpx/dto"

	"github.com/go-chi/chi/v5"
)

func (h Handler) apiTokensIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Tokens.List(r.Context(), httpx.User(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, t := range list {
		data = append(data, dto.NewAPIToken(t))
	}
	httpx.WriteData(w, http.StatusOK, data)
}

func (h Handler) apiTokensStore(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	var body struct {
		Name        string   `json:"name"`
		Preset      string   `json:"preset"`
		AllowDelete bool     `json:"allow_delete"`
		Permissions []string `json:"permissions"`
		AllSpaces   bool     `json:"all_spaces"`
		SpaceIDs    []int64  `json:"space_ids"`
		ExpiresAt   *string  `json:"expires_at"`
		confirm.Fields
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	errs := map[string][]string{}
	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 100 {
		errs["name"] = []string{"The name is required and must be at most 100 characters."}
	}
	// A token never gets more than it was asked for: no permissions means it
	// can list its spaces and nothing else, and it must name its spaces. It
	// writes only where its owner may too: the request is limited by the
	// token's permissions and the owner's role in the space.
	grant, grantErrs := h.grantFromBody(r.Context(), u, tokenGrantBody{
		Preset: body.Preset, AllowDelete: body.AllowDelete, Permissions: body.Permissions, AllSpaces: body.AllSpaces, SpaceIDs: body.SpaceIDs,
	})
	maps.Copy(errs, grantErrs)
	var expires *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil || !t.After(time.Now()) {
			errs["expires_at"] = []string{"The expiration must be a future date (RFC 3339)."}
		} else {
			t = t.UTC()
			expires = &t
		}
	}
	if len(errs) > 0 {
		httpx.WriteValidation(w, errs)
		return
	}
	// Asked last: a form that fails validation costs no attempt.
	if !h.Confirm.Require(w, r, body.Fields) {
		return
	}
	raw, tok, err := h.Tokens.Issue(r.Context(), u, name, grant, expires)
	if err == auth.ErrTokenLimit {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "You can create at most 50 API tokens.")
		return
	}
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteDataMsg(w, http.StatusCreated, dto.NewCreatedAPIToken(*tok, raw), "API token created.")
}

func (h Handler) apiTokensDestroy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Tokens.Revoke(r.Context(), httpx.User(r).ID, id); err != nil {
		if err == auth.ErrNotFound {
			httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
			return
		}
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteMessage(w, http.StatusOK, "API token revoked.")
}

// apiTokenCapabilities says what each kind of token can be given: the parts of
// a space, and what each preset grants over them. The screens that configure a
// token offer exactly this and keep no list of their own. Session only, like
// the rest of token management.
func (h Handler) apiTokenCapabilities(w http.ResponseWriter, r *http.Request) {
	kinds := map[string]dto.TokenKindCapabilities{}
	for _, kind := range []string{permissions.KindPersonal, permissions.KindMCP} {
		resources := permissions.ResourcesFor(kind)
		presets := map[string][]string{}
		for _, p := range permissions.Presets {
			perms, err := permissions.FromPreset(p, false, resources)
			if err != nil {
				httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
				return
			}
			presets[p] = perms
		}
		kinds[kind] = dto.TokenKindCapabilities{Resources: resources, Presets: presets}
	}
	httpx.WriteData(
		w,
		http.StatusOK,
		dto.TokenCapabilities{Levels: permissions.Levels, Presets: permissions.Presets, Kinds: kinds},
	)
}

// tokenGrantBody is the part of a token request that says what it may do: a
// preset or a list of permissions, and the spaces it reaches.
type tokenGrantBody struct {
	Preset      string
	AllowDelete bool
	Permissions []string
	AllSpaces   bool
	SpaceIDs    []int64
}

// grantFromBody checks a token request and turns it into a grant, or says what
// is wrong, by field. A token never gets more than it was asked for: no
// permissions means it can list its spaces and nothing else, and it must name
// its spaces, all of which the owner belongs to. It writes only where its owner
// may too: the request is limited by the token's permissions and the owner's
// role in the space.
func (h Handler) grantFromBody(ctx context.Context, u *auth.User, b tokenGrantBody) (auth.Grant, map[string][]string) {
	errs := map[string][]string{}
	var perms []string
	var err error
	switch {
	case b.Preset != "" && len(b.Permissions) > 0:
		errs["permissions"] = []string{"Give a preset or a list of permissions, not both."}
	case b.Preset != "":
		if perms, err = permissions.FromPreset(b.Preset, b.AllowDelete, permissions.APIResources); err != nil {
			errs["preset"] = []string{"The preset must be viewer, editor or admin."}
		}
	default:
		if perms, err = permissions.Normalize(b.Permissions); err != nil {
			errs["permissions"] = []string{err.Error() + "."}
		}
	}
	var spaceIDs []int64
	switch {
	case b.AllSpaces && len(b.SpaceIDs) > 0:
		errs["space_ids"] = []string{"Choose all spaces or a list of spaces, not both."}
	case !b.AllSpaces && len(b.SpaceIDs) == 0:
		errs["space_ids"] = []string{"Choose the spaces this token may reach."}
	case !b.AllSpaces:
		for _, id := range b.SpaceIDs {
			if role, err := h.Spaces.Role(ctx, id, u.ID); err != nil || role == "" {
				errs["space_ids"] = []string{"You can only choose spaces you belong to."}
				break
			}
		}
		spaceIDs = b.SpaceIDs
	}
	return auth.Grant{Permissions: perms, AllSpaces: b.AllSpaces, SpaceIDs: spaceIDs}, errs
}

// apiTokensUpdate changes what a token may do and which spaces it reaches,
// without issuing a new one, so a connected app keeps working with less or
// more. Like creating a token, it needs a real session. A connected AI app can
// be given only the parts of a space its tools use.
func (h Handler) apiTokensUpdate(w http.ResponseWriter, r *http.Request) {
	u := httpx.User(r)
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Preset      string   `json:"preset"`
		AllowDelete bool     `json:"allow_delete"`
		Permissions []string `json:"permissions"`
		AllSpaces   bool     `json:"all_spaces"`
		SpaceIDs    []int64  `json:"space_ids"`
		confirm.Fields
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"permissions": {"The given data was invalid."}})
		return
	}
	grant, errs := h.grantFromBody(r.Context(), u, tokenGrantBody{
		Preset: body.Preset, AllowDelete: body.AllowDelete, Permissions: body.Permissions, AllSpaces: body.AllSpaces, SpaceIDs: body.SpaceIDs,
	})
	if len(errs) > 0 {
		httpx.WriteValidation(w, errs)
		return
	}
	if !h.Confirm.Require(w, r, body.Fields) {
		return
	}
	tok, err := h.Tokens.UpdateGrant(r.Context(), u.ID, id, grant)
	switch {
	case errors.Is(err, auth.ErrNotFound):
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
	case errors.Is(err, permissions.ErrBadGrant):
		// A connected app can be given only the parts of a space its tools use; the error says which.
		msg := "The given data was invalid."
		if err != permissions.ErrBadGrant {
			msg = err.Error() + "."
		}
		httpx.WriteValidation(w, map[string][]string{"permissions": {msg}})
	case err != nil:
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
	default:
		httpx.WriteDataMsg(w, http.StatusOK, dto.NewAPIToken(*tok), "API token updated.")
	}
}
