package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func mcpSpacePath(id int64, rest string) string {
	return "/api/spaces/" + strconv.FormatInt(id, 10) + rest
}

func mcpFail(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}

// mcpGet runs a GET against the app's own router as the caller. The bearer
// token comes from the caller's /mcp request, so the usual middleware decides
// what they may see.
func (h Handler) mcpGet(ctx context.Context, req *mcp.CallToolRequest, path string, q url.Values) *mcp.CallToolResult {
	return h.mcpCall(ctx, req, http.MethodGet, path, q, nil)
}

// mcpGetFields is mcpGet that keeps only the named fields of the answer's data
// (all of it when none are named). The trimming comes before the size check, so
// the trimmed answer is measured, not the full one. allowed limits the first
// segment of a field, or is nil for any.
func (h Handler) mcpGetFields(
	ctx context.Context,
	req *mcp.CallToolRequest,
	path string,
	q url.Values,
	fields []string,
	allowed map[string]bool,
) *mcp.CallToolResult {
	tree, err := parseFields(fields, allowed)
	if err != nil {
		return mcpFail(err.Error())
	}
	if tree == nil {
		return h.mcpGet(ctx, req, path, q)
	}
	return h.mcpDo(ctx, req, http.MethodGet, path, q, nil, shapeData(tree))
}

// mcpCall is mcpGet for any method. A body is sent as JSON.
func (h Handler) mcpCall(
	ctx context.Context,
	req *mcp.CallToolRequest,
	method, path string,
	q url.Values,
	body []byte,
) *mcp.CallToolResult {
	return h.mcpDo(ctx, req, method, path, q, body, nil)
}

func (h Handler) mcpDo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	method, path string,
	q url.Values,
	body []byte,
	shape func([]byte) ([]byte, error),
) *mcp.CallToolResult {
	var authz string
	if extra := req.GetExtra(); extra != nil {
		authz = extra.Header.Get("Authorization")
	}
	if authz == "" {
		return mcpFail("Unauthenticated.")
	}
	target := path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	// chi would reuse the routing state of the /mcp request (method POST) for a
	// context that carries it, so the call starts with a clean one.
	ctx = context.WithValue(ctx, chi.RouteCtxKey, (*chi.Context)(nil))
	hr, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return mcpFail(err.Error())
	}
	hr.Header.Set("Authorization", authz)
	hr.Header.Set("Accept", "application/json")
	if body != nil {
		hr.Header.Set("Content-Type", "application/json")
	}
	rec := &memoryResponse{header: http.Header{}, status: http.StatusOK}
	h.Inner.ServeHTTP(rec, hr)

	out := bytes.TrimSpace(rec.body.Bytes())
	if rec.status >= 400 {
		return mcpFail(apiErrorMessage(rec.status, out))
	}
	if len(out) == 0 { // 204: a delete has nothing to return
		out = []byte(`{"ok":true}`)
	}
	if shape != nil && len(out) > 0 {
		shaped, err := shape(out)
		if err != nil {
			return mcpFail("Could not shape the result: " + err.Error())
		}
		out = shaped
	}
	if len(out) > mcpMaxResult {
		return mcpFail(
			"The result is too large. Narrow it with filters (dates, account, category) or ask for a smaller page.",
		)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(out)}}}
}

// apiErrorMessage is the message of a REST error body, or its status.
func apiErrorMessage(status int, body []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return e.Message
	}
	return http.StatusText(status)
}

// memoryResponse is a ResponseWriter that keeps the answer in memory.
type memoryResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (m *memoryResponse) Header() http.Header         { return m.header }
func (m *memoryResponse) WriteHeader(status int)      { m.status = status }
func (m *memoryResponse) Write(b []byte) (int, error) { return m.body.Write(b) }
