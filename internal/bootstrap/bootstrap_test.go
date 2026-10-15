package bootstrap

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewGeneratesGroupedPasswordAndWritesFile(t *testing.T) {
	dir := t.TempDir()
	g := New(dir, Binary)

	if !g.Active() {
		t.Fatal("guard should be active")
	}
	if !regexp.MustCompile(`^[A-Z2-7]{5}(-[A-Z2-7]{5}){5}$`).MatchString(g.Password()) {
		t.Fatalf("password %q is not 30 base32 characters in groups of five", g.Password())
	}
	path := filepath.Join(dir, FileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != g.Password() {
		t.Fatalf("file holds %q, want %q", raw, g.Password())
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("file mode %v", info.Mode().Perm())
		}
	}
	if g.Hint().File != path {
		t.Fatalf("hint file %q, want %q", g.Hint().File, path)
	}
}

func TestAnnounceLogsOneRecordPerLine(t *testing.T) {
	var logs strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	g := New(t.TempDir(), Binary)
	g.Announce("http://localhost:80")

	out := logs.String()
	if strings.Contains(out, `\n`) {
		t.Fatalf("announcement has escaped newlines:\n%s", out)
	}
	if !strings.Contains(out, "Bootstrap password: "+g.Password()) {
		t.Fatalf("password missing from announcement:\n%s", out)
	}
	if n := strings.Count(out, "\n"); n != 5 {
		t.Fatalf("got %d log lines, want 5:\n%s", n, out)
	}
}

func TestPasswordsDiffer(t *testing.T) {
	if New(t.TempDir(), Binary).Password() == New(t.TempDir(), Binary).Password() {
		t.Fatal("two guards share a password")
	}
}

func TestRestartReusesTheFile(t *testing.T) {
	dir := t.TempDir()
	first := New(dir, Binary).Password()
	if again := New(dir, Binary).Password(); again != first {
		t.Fatalf("restart gave %q, want %q", again, first)
	}
}

func TestGarbageFileIsReplaced(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("hunter2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g := New(dir, Binary)
	if g.Password() == "hunter2" {
		t.Fatal("accepted a password that is not ours")
	}
	raw, _ := os.ReadFile(filepath.Join(dir, FileName))
	if strings.TrimSpace(string(raw)) != g.Password() {
		t.Fatalf("file not rewritten: %q", raw)
	}
}

func TestUnwritableFileOnlyLogs(t *testing.T) {
	g := New(filepath.Join(t.TempDir(), "missing", "dir"), Docker)
	if !g.Active() || g.Password() == "" {
		t.Fatal("guard must work from memory when the file cannot be written")
	}
}

func TestSecureCompare(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"ABCDE-FGHIJ", "ABCDE-FGHIJ", true},
		{"ABCDE-FGHIJ", "ABCDE-FGHIK", false},
		{"ABCDE-FGHIJ", "ABCDE", false},
		{"ABCDE-FGHIJ", "", false},
		{"", "", true},
	} {
		if got := secureCompare(c.a, c.b); got != c.want {
			t.Errorf("secureCompare(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

func TestRegisterAcceptsTypingVariants(t *testing.T) {
	g := New(t.TempDir(), Binary)
	typed := " " + strings.ToLower(strings.ReplaceAll(g.Password(), "-", " ")) + " "
	if err := g.Register(typed, func() error { return nil }); err != nil {
		t.Fatalf("lowercase, spaced password refused: %v", err)
	}
}

func TestRegisterRefusesWrongOrMissing(t *testing.T) {
	g := New(t.TempDir(), Binary)
	called := false
	create := func() error { called = true; return nil }
	for _, candidate := range []string{"", "nope", g.Password() + "A"} {
		if err := g.Register(candidate, create); !errors.Is(err, ErrWrongPassword) {
			t.Errorf("Register(%q) = %v, want ErrWrongPassword", candidate, err)
		}
	}
	if called || !g.Active() {
		t.Fatal("a wrong password must neither create nor invalidate")
	}
}

func TestRegisterInvalidatesAndRemovesFile(t *testing.T) {
	dir := t.TempDir()
	g := New(dir, Binary)
	pass := g.Password()
	if err := g.Register(pass, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if g.Active() {
		t.Fatal("still active after the first user")
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}
	if err := g.Register(pass, func() error { t.Fatal("created twice"); return nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("second Register = %v, want ErrClosed", err)
	}
}

func TestFailedCreateKeepsThePassword(t *testing.T) {
	g := New(t.TempDir(), Binary)
	boom := errors.New("email taken")
	if err := g.Register(g.Password(), func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("got %v, want the create error", err)
	}
	if !g.Active() {
		t.Fatal("a failed create must leave registration open for a retry")
	}
	if err := g.Register(g.Password(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestParallelRegistrationsCreateExactlyOne(t *testing.T) {
	g := New(t.TempDir(), Binary)
	var created, closed atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := g.Register(g.Password(), func() error { created.Add(1); return nil })
			if errors.Is(err, ErrClosed) {
				closed.Add(1)
			}
		}()
	}
	wg.Wait()
	if created.Load() != 1 || closed.Load() != 19 {
		t.Fatalf("created %d, closed %d", created.Load(), closed.Load())
	}
}

func TestInvalidateIsIdempotent(t *testing.T) {
	g := New(t.TempDir(), Binary)
	g.Invalidate()
	g.Invalidate()
	if g.Active() {
		t.Fatal("active after Invalidate")
	}
}

func TestNilGuardIsClosed(t *testing.T) {
	var g *Guard
	if g.Active() {
		t.Fatal("nil guard is active")
	}
	if err := g.Register("x", func() error { return nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("nil Register = %v", err)
	}
	g.Invalidate()
}

func TestHintPerEnvironment(t *testing.T) {
	for _, env := range []Env{Docker, Systemd, Binary} {
		g := New(t.TempDir(), env)
		if g.Hint().Environment != string(env) {
			t.Errorf("hint env %q, want %q", g.Hint().Environment, env)
		}
	}
}

func TestDetectEnv(t *testing.T) {
	none := func(string) bool { return false }
	dockerenv := func(p string) bool { return p == "/.dockerenv" }
	getenv := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	for _, c := range []struct {
		name   string
		exists func(string) bool
		env    map[string]string
		want   Env
	}{
		{"plain binary", none, nil, Binary},
		{"docker", dockerenv, nil, Docker},
		{"systemd", none, map[string]string{"INVOCATION_ID": "abc"}, Systemd},
		{"docker wins", dockerenv, map[string]string{"INVOCATION_ID": "abc"}, Docker},
	} {
		if got := detectEnv(c.exists, getenv(c.env)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestVerifyChecksWithoutSpendingThePassword(t *testing.T) {
	g := New(t.TempDir(), Binary)
	for _, candidate := range []string{"", "nope", g.Password() + "A"} {
		if err := g.Verify(candidate); !errors.Is(err, ErrWrongPassword) {
			t.Errorf("Verify(%q) = %v, want ErrWrongPassword", candidate, err)
		}
	}
	typed := " " + strings.ToLower(g.Password()) + " "
	if err := g.Verify(typed); err != nil {
		t.Fatalf("right password refused: %v", err)
	}
	if !g.Active() {
		t.Fatal("Verify must not invalidate")
	}
}

func TestVerifyIsClosedWithoutAPassword(t *testing.T) {
	var nilGuard *Guard
	if err := nilGuard.Verify("x"); !errors.Is(err, ErrClosed) {
		t.Errorf("nil guard: %v, want ErrClosed", err)
	}
	g := New(t.TempDir(), Binary)
	pass := g.Password()
	g.Invalidate()
	if err := g.Verify(pass); !errors.Is(err, ErrClosed) {
		t.Errorf("invalidated guard: %v, want ErrClosed", err)
	}
}
