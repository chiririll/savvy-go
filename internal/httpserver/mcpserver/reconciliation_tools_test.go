package mcpserver

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func listedTools(t *testing.T, perms ...string) map[string]*mcp.Tool {
	t.Helper()
	ctx := context.Background()
	srv := Handler{}.mcpServer(perms)
	ct, st := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	sess, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	list, err := sess.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*mcp.Tool{}
	for _, tl := range list.Tools {
		out[tl.Name] = tl
	}
	return out
}

func TestReconciliationToolsFollowThePermissions(t *testing.T) {
	cases := []struct {
		perms []string
		want  map[string]bool
	}{
		{[]string{"accounts:read"}, map[string]bool{"list_reconciliations": true}},
		{[]string{"accounts:read", "accounts:write"}, map[string]bool{"list_reconciliations": true, "reconcile_account": true}},
		{
			[]string{"accounts:read", "accounts:write", "accounts:delete"},
			map[string]bool{"list_reconciliations": true, "reconcile_account": true, "delete_reconciliation": true},
		},
		{[]string{"transactions:read", "transactions:write"}, map[string]bool{}},
	}
	for _, c := range cases {
		tools := listedTools(t, c.perms...)
		for _, name := range []string{"list_reconciliations", "reconcile_account", "delete_reconciliation"} {
			if _, ok := tools[name]; ok != c.want[name] {
				t.Errorf("perms %v: %s listed = %v, want %v", c.perms, name, ok, c.want[name])
			}
		}
	}
}

func TestReconciliationToolSchemasAndHints(t *testing.T) {
	tools := listedTools(t, "accounts:read", "accounts:write", "accounts:delete")

	rec := tools["reconcile_account"]
	if rec.Annotations.ReadOnlyHint || rec.Annotations.DestructiveHint == nil || *rec.Annotations.DestructiveHint {
		t.Errorf("reconcile_account hints = %+v", rec.Annotations)
	}
	if del := tools["delete_reconciliation"]; del.Annotations.DestructiveHint == nil || !*del.Annotations.DestructiveHint {
		t.Errorf("delete_reconciliation must be destructive: %+v", del.Annotations)
	}
	if !tools["list_reconciliations"].Annotations.ReadOnlyHint {
		t.Error("list_reconciliations must be read-only")
	}

	raw, _ := json.Marshal(rec.InputSchema)
	var schema struct {
		Required   []string `json:"required"`
		Properties struct {
			Accounts struct {
				Items struct {
					Required   []string                  `json:"required"`
					Properties map[string]map[string]any `json:"properties"`
				} `json:"items"`
			} `json:"accounts"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"space_id", "date", "accounts"} {
		if !slices.Contains(schema.Required, f) {
			t.Errorf("%s is not required: %v", f, schema.Required)
		}
	}
	item := schema.Properties.Accounts.Items
	if !slices.Contains(item.Required, "account_id") || !slices.Contains(item.Required, "actual_balance") ||
		slices.Contains(item.Required, "note") {
		t.Errorf("item required = %v", item.Required)
	}
	if item.Properties["actual_balance"]["type"] != "string" {
		t.Errorf("actual_balance must be a string: %v", item.Properties["actual_balance"])
	}
}

func TestReconcileBodyKeepsBalancesExact(t *testing.T) {
	in := `{"date":"2026-03-01","accounts":[{"account_id":3,"actual_balance":"0.10","note":"statement"},` +
		`{"account_id":4,"actual_balance":"123456789012345.67"}]}`
	var args map[string]any
	if err := json.Unmarshal([]byte(in), &args); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(reconcileBody(args))
	want := `{"date":"2026-03-01","items":[{"accountId":3,"actualBalance":"0.10","note":"statement"},` +
		`{"accountId":4,"actualBalance":"123456789012345.67"}]}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
