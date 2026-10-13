// Package bootstrap protects the first run. While the server has no users,
// registration needs a one-time password that is printed in the log and kept in
// a file in the data directory, so only someone with access to the machine can
// claim the admin account.
package bootstrap

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileName is the password file in the data directory.
const FileName = "bootstrap-password"

// Env says how the server was started, which decides how the operator reads
// the password.
type Env string

const (
	Docker  Env = "docker"
	Systemd Env = "systemd"
	Binary  Env = "binary"
)

var (
	// ErrClosed means no bootstrap password is in force: the first user exists.
	ErrClosed = errors.New("bootstrap: registration is closed")
	// ErrWrongPassword means the password is missing or does not match.
	ErrWrongPassword = errors.New("bootstrap: wrong password")
)

// Hint tells the setup page where the operator finds the password.
type Hint struct {
	Environment string
	File        string
}

// DetectEnv reads the environment of the running process.
func DetectEnv() Env {
	return detectEnv(func(p string) bool { _, err := os.Stat(p); return err == nil }, os.Getenv)
}

func detectEnv(exists func(string) bool, getenv func(string) string) Env {
	switch {
	case exists("/.dockerenv"):
		return Docker
	case getenv("INVOCATION_ID") != "":
		return Systemd
	}
	return Binary
}

// Guard holds the bootstrap password. A nil Guard is closed.
type Guard struct {
	mu       sync.Mutex
	password string // empty once invalidated
	path     string
	env      Env
}

// New creates the password for a server with no users. A restart reuses the
// value in the file. If the file cannot be written the password lives in
// memory and in the log only.
func New(dataDir string, env Env) *Guard {
	g := &Guard{path: filepath.Join(dataDir, FileName), env: env}
	if raw, err := os.ReadFile(g.path); err == nil {
		if p := strings.TrimSpace(string(raw)); wellFormed(p) {
			g.password = p
			return g
		}
	}
	g.password = generate()
	if err := os.WriteFile(g.path, []byte(g.password+"\n"), 0o600); err != nil {
		slog.Warn("could not write the bootstrap password file; read the password from the log", "file", g.path, "err", err)
	}
	return g
}

// Active reports whether registration still needs the password.
func (g *Guard) Active() bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.password != ""
}

// Password is the current password, empty once invalidated.
func (g *Guard) Password() string {
	if g == nil {
		return ""
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.password
}

// Hint says where to find the password.
func (g *Guard) Hint() Hint {
	if g == nil {
		return Hint{}
	}
	return Hint{Environment: string(g.env), File: g.path}
}

// Register checks the password and runs create as one step. When create
// succeeds the password is invalidated and its file removed; when it fails the
// password stays valid so the operator can retry. At most one create succeeds.
func (g *Guard) Register(candidate string, create func() error) error {
	if g == nil {
		return ErrClosed
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.password == "" {
		return ErrClosed
	}
	if !secureCompare(g.password, candidate) {
		return ErrWrongPassword
	}
	if err := create(); err != nil {
		return err
	}
	g.invalidate()
	return nil
}

// Invalidate closes the guard and removes the file. Call it when the first
// user comes into being by another route, such as a restore.
func (g *Guard) Invalidate() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.invalidate()
}

func (g *Guard) invalidate() {
	g.password = ""
	if err := os.Remove(g.path); err != nil && !os.IsNotExist(err) {
		slog.Warn("could not remove the bootstrap password file", "file", g.path, "err", err)
	}
}

// Announce prints the password once, in one block that is easy to spot.
func (g *Guard) Announce(url string) {
	if !g.Active() {
		return
	}
	slog.Info("FIRST RUN: there are no users yet.\n"+
		"    Open "+url+" and create the admin account.\n"+
		"    Bootstrap password: "+g.Password()+"\n"+
		"    It is also in the file "+g.path+"\n"+
		"    It stops working as soon as the first user exists.",
		"file", g.path)
}

// generate returns 150 random bits as six dash-separated groups of five base32
// characters.
func generate() string {
	buf := make([]byte, 19)
	if _, err := rand.Read(buf); err != nil {
		panic("bootstrap: no randomness: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)[:30]
	groups := make([]string, 0, 6)
	for i := 0; i < len(enc); i += 5 {
		groups = append(groups, enc[i:i+5])
	}
	return strings.Join(groups, "-")
}

func wellFormed(p string) bool {
	if len(p) != 35 {
		return false
	}
	for i, r := range p {
		switch {
		case i%6 == 5:
			if r != '-' {
				return false
			}
		case !(r >= 'A' && r <= 'Z' || r >= '2' && r <= '7'):
			return false
		}
	}
	return true
}

// normalize makes typing forgiving: case, spaces and dashes do not matter.
func normalize(s string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "", "\t", "").Replace(s))
}

// secureCompare compares two passwords in constant time.
func secureCompare(a, b string) bool {
	ha := sha256.Sum256([]byte(normalize(a)))
	hb := sha256.Sum256([]byte(normalize(b)))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
