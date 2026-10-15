package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeServer is just enough of the authorization server for the probe's happy path.
type fakeServer struct {
	*httptest.Server
	registrationOpen bool
	challenge        string
	spent            bool
}

func newFake(t *testing.T) *fakeServer {
	f := &fakeServer{registrationOpen: true}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		m := map[string]string{
			"issuer": f.URL, "authorization_endpoint": f.URL + "/oauth/authorize", "token_endpoint": f.URL + "/oauth/token",
		}
		if f.registrationOpen {
			m["registration_endpoint"] = f.URL + "/oauth/register"
		}
		json.NewEncoder(w).Encode(m)
	})
	mux.HandleFunc("/oauth/register", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"client_id": "c1"})
	})
	mux.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		f.challenge = q.Get("code_challenge")
		http.Redirect(w, r, q.Get("redirect_uri")+"?code=abc&state="+url.QueryEscape(q.Get("state")), http.StatusFound)
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.PostForm.Get("grant_type") {
		case "authorization_code":
			sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
			if f.spent || r.PostForm.Get("code") != "abc" || base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
				http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
				return
			}
			f.spent = true
		case "refresh_token":
			if r.PostForm.Get("refresh_token") != "r1" {
				http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
				return
			}
		}
		json.NewEncoder(w).Encode(tokens{AccessToken: "a1", RefreshToken: "r1", ExpiresIn: 3600, Scope: "read"})
	})
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer a1" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"tools\":[{},{}]}}\n\n"))
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func newProbe(f *fakeServer, out *bytes.Buffer) *probe {
	p := &probe{server: f.URL, out: out, http: f.Client()}
	// The "browser": follow the authorize URL, which redirects to the probe's listener.
	p.open = func(u string) { go http.Get(u) }
	return p
}

func TestProbeRunsTheWholeFlow(t *testing.T) {
	f := newFake(t)
	var out bytes.Buffer
	if err := newProbe(f, &out).run(context.Background(), 0, 5*time.Second); err != nil {
		t.Fatalf("run: %v\n%s", err, out.String())
	}
	for _, want := range []string{"registered client c1", "a spent code is refused", "returned 2 tools", "refreshed token works"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestProbeSaysWhenRegistrationIsClosed(t *testing.T) {
	f := newFake(t)
	f.registrationOpen = false
	err := newProbe(f, &bytes.Buffer{}).run(context.Background(), 0, time.Second)
	if err == nil || !strings.Contains(err.Error(), "registration is closed") {
		t.Fatalf("got %v, want the registration hint", err)
	}
}

func TestProbeCatchesAnIssuerThatIsNotTheAddressYouUse(t *testing.T) {
	f := newFake(t)
	p := newProbe(f, &bytes.Buffer{})
	p.server = strings.Replace(f.URL, "127.0.0.1", "localhost", 1)
	err := p.run(context.Background(), 0, time.Second)
	if err == nil || !strings.Contains(err.Error(), "issuer") {
		t.Fatalf("got %v, want an issuer mismatch", err)
	}
}
