package auth

import (
	"fmt"
	"slices"
	"strings"
)

// What an API token may do, one grant per resource: "transactions:read" or
// "transactions:write". Write includes read. A token has no grant it was not
// given, so a new resource is closed to every existing token.
const (
	PermRead  = "read"
	PermWrite = "write"
)

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
// duplicates, with a read grant dropped where the same resource has write.
// An unknown resource or level is an error: a typo must not become a token
// that silently grants less (or, by a loose match, more) than asked.
func NormalizePermissions(in []string) ([]string, error) {
	level := map[string]string{}
	for _, p := range in {
		resource, lv, ok := strings.Cut(p, ":")
		if !ok || !slices.Contains(APIResources, resource) || (lv != PermRead && lv != PermWrite) {
			return nil, fmt.Errorf("unknown permission %q", p)
		}
		if level[resource] != PermWrite {
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
