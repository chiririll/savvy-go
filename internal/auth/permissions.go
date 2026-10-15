package auth

import (
	"fmt"
	"slices"
	"strings"
)

// What an API token may do, one grant per resource: "transactions:read",
// "transactions:write" or "transactions:delete". Each level includes the ones
// before it: write (create and change) includes read, delete includes write.
// Deleting is its own level so a token can add data without being able to
// remove it. A token has no grant it was not given, so a new resource is
// closed to every existing token.
const (
	PermRead   = "read"
	PermWrite  = "write"
	PermDelete = "delete"
)

// Levels lists the permission levels, lowest first.
var Levels = []string{PermRead, PermWrite, PermDelete}

// permRank orders the levels (0 for an unknown one); a grant covers every level
// up to its own.
func permRank(level string) int { return slices.Index(Levels, level) + 1 }

// LevelFor is the level a request method needs.
func LevelFor(method string) string {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return PermRead
	case "DELETE":
		return PermDelete
	}
	return PermWrite
}

// Presets are ready-made sets of grants, named after space roles. They are
// what a token gets when it is created with a preset, and never more than the
// owner's role in a space allows anyway.
const (
	PresetViewer = "viewer" // read everything
	PresetEditor = "editor" // read everything, add and change the data in EditorResources
	PresetAdmin  = "admin"  // read, write and delete everything
)

// Presets lists the presets, in the order a screen offers them.
var Presets = []string{PresetViewer, PresetEditor, PresetAdmin}

// EditorResources are the parts of a space the editor preset may add to and
// change. Categories, currencies, reports, transfers and the space itself stay
// read-only: they shape how everything else is counted or sent.
var EditorResources = []string{"accounts", "automation", "budgets", "debts", "recurring", "tags", "transactions"}

// PresetPermissions expands a preset over some parts of a space (those a kind
// of token can hold, see ResourcesFor). allowDelete lets an editor delete
// transactions too; it means nothing for the other presets.
func PresetPermissions(preset string, allowDelete bool, resources []string) ([]string, error) {
	grant := func(level string, resources ...string) []string {
		out := make([]string, len(resources))
		for i, r := range resources {
			out[i] = Perm(r, level)
		}
		return out
	}
	switch preset {
	case PresetViewer:
		return grant(PermRead, resources...), nil
	case PresetEditor:
		out := grant(PermRead, resources...)
		for _, r := range EditorResources {
			if !slices.Contains(resources, r) {
				continue
			}
			level := PermWrite
			if allowDelete && r == "transactions" {
				level = PermDelete
			}
			out = append(out, Perm(r, level))
		}
		return NormalizePermissions(out)
	case PresetAdmin:
		return grant(PermDelete, resources...), nil
	}
	return nil, fmt.Errorf("unknown preset %q", preset)
}

// APIResources are the parts of a space a token can be given. httpserver maps
// every space route to one of them and refuses a route it cannot map.
var APIResources = []string{
	"accounts", "automation", "budgets", "categories", "currencies", "debts",
	"recurring", "reports", "space", "tags", "transactions", "transfers",
}

// MCPResources are the ones the MCP tools read: what an assistant is granted
// when someone connects it.
var MCPResources = []string{
	"accounts", "automation", "budgets", "categories", "currencies", "debts", "reports", "tags", "transactions",
}

// What a token is for. A personal token is made in Settings and may be given
// any part of a space; an mcp token is the access token of a connected AI app
// and only the parts its tools use. The kind is set when the token is issued
// and never changes, so nothing has to guess it from the name.
const (
	TokenPersonal = "personal"
	TokenMCP      = "mcp"
)

// ResourcesFor are the parts of a space a token of this kind can be given.
func ResourcesFor(kind string) []string {
	if kind == TokenMCP {
		return MCPResources
	}
	return APIResources
}

// grantError is an ErrBadGrant that says what is wrong.
type grantError string

func (e grantError) Error() string      { return string(e) }
func (grantError) Is(target error) bool { return target == ErrBadGrant }

// CheckKindPermissions refuses a grant that reaches beyond what the kind of
// token may hold. permissions are already normalized. The error is an
// ErrBadGrant whose text can be shown to the user.
func CheckKindPermissions(kind string, permissions []string) error {
	if kind == TokenMCP && len(permissions) == 0 {
		return grantError("A connected app needs at least one permission")
	}
	allowed := ResourcesFor(kind)
	for _, p := range permissions {
		resource, _, _ := strings.Cut(p, ":")
		if !slices.Contains(allowed, resource) {
			return grantError(fmt.Sprintf("A connected app can only be given access to %s, not %q", strings.Join(allowed, ", "), p))
		}
	}
	return nil
}

// Perm is the grant string for a resource and level.
func Perm(resource, level string) string { return resource + ":" + level }

// ReadPerms grants read on each resource.
func ReadPerms(resources ...string) []string {
	out := make([]string, len(resources))
	for i, r := range resources {
		out[i] = Perm(r, PermRead)
	}
	return out
}

// NormalizePermissions checks a list of grants and returns it sorted, without
// duplicates, keeping only the highest level of each resource.
// An unknown resource or level is an error: a typo must not become a token
// that silently grants less (or, by a loose match, more) than asked.
func NormalizePermissions(in []string) ([]string, error) {
	level := map[string]string{}
	for _, p := range in {
		resource, lv, ok := strings.Cut(p, ":")
		if !ok || !slices.Contains(APIResources, resource) || permRank(lv) == 0 {
			return nil, fmt.Errorf("unknown permission %q", p)
		}
		if permRank(lv) > permRank(level[resource]) {
			level[resource] = lv
		}
	}
	out := make([]string, 0, len(level))
	for resource, lv := range level {
		out = append(out, Perm(resource, lv))
	}
	slices.Sort(out)
	return out, nil
}
