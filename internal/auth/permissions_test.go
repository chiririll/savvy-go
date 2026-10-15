package auth

import (
	"slices"
	"strings"
	"testing"
)

// The level each preset grants on each part of a space ("" = nothing). One row per
// resource, so a new resource has to be placed in every preset on purpose.
var presetMatrix = []struct {
	resource string
	viewer   string
	editor   string
	deleter  string // editor with deleting switched on
	admin    string
}{
	{"accounts", PermRead, PermWrite, PermWrite, PermDelete},
	{"automation", PermRead, PermWrite, PermWrite, PermDelete},
	{"budgets", PermRead, PermWrite, PermWrite, PermDelete},
	{"categories", PermRead, PermRead, PermRead, PermDelete},
	{"currencies", PermRead, PermRead, PermRead, PermDelete},
	{"debts", PermRead, PermWrite, PermWrite, PermDelete},
	{"recurring", PermRead, PermWrite, PermWrite, PermDelete},
	{"reports", PermRead, PermRead, PermRead, PermDelete},
	{"space", PermRead, PermRead, PermRead, PermDelete},
	{"tags", PermRead, PermWrite, PermWrite, PermDelete},
	{"transactions", PermRead, PermWrite, PermDelete, PermDelete},
	{"transfers", PermRead, PermRead, PermRead, PermDelete},
}

func TestPresetMatrix(t *testing.T) {
	cases := []struct {
		name        string
		preset      string
		allowDelete bool
		want        func(i int) string
	}{
		{"viewer", PresetViewer, false, func(i int) string { return presetMatrix[i].viewer }},
		{"editor", PresetEditor, false, func(i int) string { return presetMatrix[i].editor }},
		{"editor with delete", PresetEditor, true, func(i int) string { return presetMatrix[i].deleter }},
		{"admin", PresetAdmin, false, func(i int) string { return presetMatrix[i].admin }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := PresetPermissions(c.preset, c.allowDelete, APIResources)
			if err != nil {
				t.Fatal(err)
			}
			for i, row := range presetMatrix {
				want := Perm(row.resource, c.want(i))
				if !slices.Contains(got, want) {
					t.Errorf("%s: want %s in %v", row.resource, want, got)
				}
			}
			if len(got) != len(presetMatrix) {
				t.Errorf("want one grant per resource, got %v", got)
			}
		})
	}
}

// Every resource a token can hold is in the matrix, and the editor's list names no other.
func TestPresetMatrixCoversResources(t *testing.T) {
	var rows []string
	for _, row := range presetMatrix {
		rows = append(rows, row.resource)
	}
	slices.Sort(rows)
	all := slices.Sorted(slices.Values(APIResources))
	if !slices.Equal(rows, all) {
		t.Errorf("matrix rows %v differ from APIResources %v", rows, all)
	}
	for _, r := range EditorResources {
		if !slices.Contains(APIResources, r) {
			t.Errorf("EditorResources has unknown %q", r)
		}
	}
}

// A connected app is given a preset over only the parts its tools use.
func TestPresetMatrixForConnectedApp(t *testing.T) {
	got, err := PresetPermissions(PresetEditor, false, MCPResources)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range got {
		resource, _, _ := strings.Cut(p, ":")
		if !slices.Contains(MCPResources, resource) {
			t.Errorf("%q is outside the connected-app resources", p)
		}
	}
	if slices.Contains(got, "recurring:write") {
		t.Errorf("recurring is not a connected-app resource, got %v", got)
	}
}
