// Command oauth-probe is a throwaway OAuth client for checking a running Go Savvy
// server by hand: it registers itself, sends you through the consent page,
// catches the redirect on a loopback port, trades the code for tokens, calls
// /mcp, then refreshes. Every step prints PASS or FAIL.
//
//	go run ./cmd/oauth-probe -server http://localhost:8080
//
// Registration is closed unless an admin opened it (Administration → Security)
// within the last 5 minutes.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "base URL of the Go Savvy server (its app_url)")
	port := flag.Int("port", 0, "loopback port for the redirect (0 picks a free one)")
	noBrowser := flag.Bool("no-browser", false, "only print the authorize URL")
	wait := flag.Duration("wait", 5*time.Minute, "how long to wait for you to approve")
	flag.Parse()

	p := &probe{server: strings.TrimRight(*server, "/"), out: os.Stdout, http: &http.Client{Timeout: 15 * time.Second}}
	if !*noBrowser {
		p.open = openBrowser
	}
	ctx, cancel := context.WithTimeout(context.Background(), *wait+time.Minute)
	defer cancel()
	if err := p.run(ctx, *port, *wait); err != nil {
		fmt.Fprintln(os.Stderr, "\nFAILED:", err)
		os.Exit(1)
	}
	fmt.Fprintln(p.out, "\nAll steps passed.")
}

type probe struct {
	server string
	out    io.Writer
	http   *http.Client
	open   func(url string) // nil: do not open a browser
}

type metadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
}

type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func (p *probe) pass(format string, a ...any) { fmt.Fprintf(p.out, "PASS  "+format+"\n", a...) }

func (p *probe) run(ctx context.Context, port int, wait time.Duration) error {
	var meta metadata
	if err := p.getJSON(ctx, p.server+"/.well-known/oauth-authorization-server", &meta); err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	if meta.Issuer != p.server {
		return fmt.Errorf(
			"discovery: issuer is %q but you asked for %q; set app_url to the address you use",
			meta.Issuer,
			p.server,
		)
	}
	p.pass("discovery, issuer %s", meta.Issuer)
	if meta.RegistrationEndpoint == "" {
		return errors.New("registration is closed: open it in Administration → Security, then run again")
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	redirect := "http://" + ln.Addr().String() + "/callback"
	callbacks := make(chan url.Values, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintln(w, "You can close this tab and go back to the terminal.")
		select {
		case callbacks <- r.URL.Query():
		default:
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	var reg struct {
		ClientID string `json:"client_id"`
	}
	if err := p.postJSON(ctx, meta.RegistrationEndpoint, map[string]any{
		"client_name": "oauth-probe", "redirect_uris": []string{redirect},
	}, &reg); err != nil || reg.ClientID == "" {
		return fmt.Errorf("registration: %v", orMissing(err, "no client_id"))
	}
	p.pass("registered client %s for %s", reg.ClientID, redirect)

	verifier, state := randomString(32), randomString(16)
	sum := sha256.Sum256([]byte(verifier))
	authURL := meta.AuthorizationEndpoint + "?" + url.Values{
		"response_type": {"code"}, "client_id": {reg.ClientID}, "redirect_uri": {redirect},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"},
		"state": {state}, "resource": {p.server + "/mcp"},
	}.Encode()
	fmt.Fprintf(p.out, "\nOpen this address, sign in and press Allow access:\n\n  %s\n\n", authURL)
	if p.open != nil {
		p.open(authURL)
	}

	var q url.Values
	select {
	case q = <-callbacks:
	case <-time.After(wait):
		return errors.New("no redirect arrived: did you press Allow access?")
	case <-ctx.Done():
		return ctx.Err()
	}
	if e := q.Get("error"); e != "" {
		return fmt.Errorf("authorization refused: %s %s", e, q.Get("error_description"))
	}
	if q.Get("state") != state {
		return errors.New("redirect came back with a different state")
	}
	if q.Get("code") == "" {
		return errors.New("redirect carried no code")
	}
	p.pass("redirect received with code and matching state")

	var t tokens
	if err := p.postForm(ctx, meta.TokenEndpoint, url.Values{
		"grant_type": {"authorization_code"}, "client_id": {reg.ClientID}, "code": {q.Get("code")},
		"code_verifier": {verifier}, "redirect_uri": {redirect},
	}, &t); err != nil || t.AccessToken == "" || t.RefreshToken == "" {
		return fmt.Errorf("token exchange: %v", orMissing(err, "missing tokens"))
	}
	p.pass("code exchanged, access token expires in %ds, scope %q", t.ExpiresIn, t.Scope)

	var again tokens
	if err := p.postForm(ctx, meta.TokenEndpoint, url.Values{
		"grant_type": {"authorization_code"}, "client_id": {reg.ClientID}, "code": {q.Get("code")},
		"code_verifier": {verifier}, "redirect_uri": {redirect},
	}, &again); err == nil {
		return errors.New("the same code was accepted twice")
	}
	p.pass("a spent code is refused")

	n, err := p.listTools(ctx, t.AccessToken)
	if err != nil {
		return fmt.Errorf("mcp: %w", err)
	}
	p.pass("/mcp tools/list returned %d tools", n)

	var next tokens
	if err := p.postForm(ctx, meta.TokenEndpoint, url.Values{
		"grant_type": {"refresh_token"}, "client_id": {reg.ClientID}, "refresh_token": {t.RefreshToken},
	}, &next); err != nil || next.AccessToken == "" {
		return fmt.Errorf("refresh: %v", orMissing(err, "missing tokens"))
	}
	p.pass("refresh gave a new access token")
	if _, err := p.listTools(ctx, next.AccessToken); err != nil {
		return fmt.Errorf("mcp with refreshed token: %w", err)
	}
	p.pass("the refreshed token works on /mcp")
	return nil
}

// listTools calls tools/list on the stateless MCP endpoint and counts the tools.
func (p *probe) listTools(ctx context.Context, access string) (int, error) {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.server+"/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := p.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	// The reply is plain JSON or one server-sent event.
	if i := bytes.Index(raw, []byte("data:")); i >= 0 {
		raw = raw[i+len("data:"):]
	}
	var out struct {
		Result struct {
			Tools []json.RawMessage `json:"tools"`
		} `json:"result"`
		Error *struct{ Message string } `json:"error"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &out); err != nil {
		return 0, fmt.Errorf("unreadable reply: %w", err)
	}
	if out.Error != nil {
		return 0, errors.New(out.Error.Message)
	}
	return len(out.Result.Tools), nil
}

func (p *probe) getJSON(ctx context.Context, u string, into any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	return p.do(req, into)
}

func (p *probe) postJSON(ctx context.Context, u string, body, into any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return p.do(req, into)
}

func (p *probe) postForm(ctx context.Context, u string, form url.Values, into any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return p.do(req, into)
}

func (p *probe) do(req *http.Request, into any) error {
	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.Unmarshal(raw, into)
}

func orMissing(err error, what string) error {
	if err != nil {
		return err
	}
	return errors.New(what)
}

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func openBrowser(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	_ = cmd.Start()
}
