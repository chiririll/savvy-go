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

// permRank orders the levels; a grant covers every level up to its own.
var permRank = map[string]int{PermRead: 1, PermWrite: 2, PermDelete: 3}

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
	PresetEditor = "editor" // read everything, add and change transactions
	PresetAdmin  = "admin"  // read, write and delete everything
)

// PresetPermissions expands a preset. allowDelete lets an editor delete
// transactions too; it means nothing for the other presets.
func PresetPermissions(preset string, allowDelete bool) ([]string, error) {
	grant := func(level string, resources ...string) []string {
		out := make([]string, len(resources))
		for i, r := range resources {
			out[i] = Perm(r, level)
		}
		return out
	}
	switch preset {
	case PresetViewer:
		return grant(PermRead, APIResources...), nil
	case PresetEditor:
		level := PermWrite
		if allowDelete {
			level = PermDelete
		}
		return NormalizePermissions(append(grant(PermRead, APIResources...), Perm("transactions", level)))
	case PresetAdmin:
		return grant(PermDelete, APIResources...), nil
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
	"accounts", "budgets", "categories", "currencies", "debts", "reports", "tags", "transactions",
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
		if !ok || !slices.Contains(APIResources, resource) || permRank[lv] == 0 {
			return nil, fmt.Errorf("unknown permission %q", p)
		}
		if permRank[lv] > permRank[level[resource]] {
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
