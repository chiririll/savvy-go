package permissions

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
	{"accounts", Read, Write, Write, Delete},
	{"automation", Read, Write, Write, Delete},
	{"budgets", Read, Write, Write, Delete},
	{"categories", Read, Read, Read, Delete},
	{"currencies", Read, Read, Read, Delete},
	{"debts", Read, Write, Write, Delete},
	{"recurring", Read, Write, Write, Delete},
	{"reports", Read, Read, Read, Delete},
	{"space", Read, Read, Read, Delete},
	{"tags", Read, Write, Write, Delete},
	{"transactions", Read, Write, Delete, Delete},
	{"transfers", Read, Read, Read, Delete},
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
			got, err := FromPreset(c.preset, c.allowDelete, APIResources)
			if err != nil {
				t.Fatal(err)
			}
			for i, row := range presetMatrix {
				want := New(row.resource, c.want(i))
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
	got, err := FromPreset(PresetEditor, false, MCPResources)
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

func TestLevelForMethod(t *testing.T) {
	for method, want := range map[string]string{"GET": Read, "HEAD": Read, "OPTIONS": Read, "POST": Write, "PUT": Write,
		"PATCH": Write, "DELETE": Delete} {
		if got := LevelFor(method); got != want {
			t.Errorf("LevelFor(%s) = %s, want %s", method, got, want)
		}
	}
}

func TestNormalizePermissions(t *testing.T) {
	got, err := Normalize([]string{"transactions:read", "transactions:delete", "accounts:read", "accounts:read"})
	if err != nil || !slices.Equal(got, []string{"accounts:read", "transactions:delete"}) {
		t.Errorf("got %v, %v", got, err)
	}
	if got, err := Normalize(nil); err != nil || len(got) != 0 {
		t.Errorf("empty: %v, %v", got, err)
	}
	if _, err := FromPreset("superuser", false, APIResources); err == nil {
		t.Error("an unknown preset")
	}
}
