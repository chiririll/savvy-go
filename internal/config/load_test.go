package config

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type server struct {
	Listen string `toml:"listen" comment:"Address to listen on."`
	AppURL string `toml:"app_url"`
}

func (server) Defaults() server { return server{Listen: "localhost:8080"} }

// mailer only holds the sections of its transports.
type mailer struct{}

func (mailer) Defaults() mailer { return mailer{} }

type smtpCfg struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func (smtpCfg) Defaults() smtpCfg { return smtpCfg{Port: 587} }

// sendgrid has no defaults: it is written only when the file has it.
type sendgrid struct {
	Key string `toml:"key"`
}

// conn is what a package makes of the smtp table.
type conn struct{ Addr string }

var (
	serverSection   = Section[server, server]{Name: "server"}
	mailerSection   = Section[mailer, mailer]{Name: "mailer"}
	smtpSection     = Optional[smtpCfg, conn]{Name: "mailer.smtp", Decode: decodeSMTP}
	sendgridSection = Optional[sendgrid, sendgrid]{Name: "mailer.sendgrid"}
)

func decodeSMTP(c smtpCfg) (conn, error) {
	if c.Port < 1 {
		return conn{}, errors.New("port must be positive")
	}
	return conn{Addr: strings.TrimSpace(c.Host) + ":" + itoa(c.Port)}, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &logs
}

// cur is the file of the test in progress.
var cur *Config

func initCur(path string) error {
	c, err := Open(path)
	if err == nil {
		cur = c
	}
	return err
}

func initAt(t *testing.T, path string) {
	t.Helper()
	if err := initCur(path); err != nil {
		t.Fatal(err)
	}
}

func finish(t *testing.T) {
	t.Helper()
	if err := cur.Finish(); err != nil {
		t.Fatal(err)
	}
}

// handles are the sections of the test program.
type handles struct {
	server   *Ref[server]
	mailer   *Ref[mailer]
	smtp     *OptRef[conn]
	sendgrid *OptRef[sendgrid]
}

// start reads path and asks for every section, as the program does at start.
func start(t *testing.T, path string) handles {
	t.Helper()
	initAt(t, path)
	h := handles{
		server:   Of(cur, serverSection),
		mailer:   Of(cur, mailerSection),
		smtp:     OfOptional(cur, smtpSection),
		sendgrid: OfOptional(cur, sendgridSection),
	}
	finish(t)
	return h
}

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestSectionsReadTheirTablesOverTheDefaults(t *testing.T) {
	h := start(t, write(t, "[server]\napp_url = \"https://a.example\"\n[mailer.smtp]\nhost = \" h \"\n"))
	if srv := h.server.Get(); srv.AppURL != "https://a.example" || srv.Listen != "localhost:8080" {
		t.Errorf("server %+v: a key the file lacks must take its default", srv)
	}
	if c, ok := h.smtp.Get(); !ok || c.Addr != "h:587" {
		t.Errorf("smtp %+v %v: must be decoded from the table over its defaults", c, ok)
	}
}

func TestDefaultsAreWrittenAndTheFileIsStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	start(t, path)
	text := read(t, path)
	for _, want := range []string{"# Go Savvy configuration", "[server]", "listen = 'localhost:8080'", "# Address to listen on.", "[mailer.smtp]", "port = 587"} {
		if !strings.Contains(text, want) {
			t.Errorf("new file lacks %q:\n%s", want, text)
		}
	}
	// A section without defaults is not written while the file lacks it, and a section that
	// only holds others gets no empty table of its own.
	if strings.Contains(text, "sendgrid") || strings.Contains(text, "[mailer]") {
		t.Errorf("unexpected tables:\n%s", text)
	}
	start(t, path)
	if read(t, path) != text {
		t.Error("file changed on the second start")
	}
}

func TestSectionWithoutDefaultsIsKeptFromTheFile(t *testing.T) {
	path := write(t, "[mailer.sendgrid]\nkey = \"k\"\n")
	h := start(t, path)
	if g, ok := h.sendgrid.Get(); !ok || g.Key != "k" {
		t.Errorf("sendgrid %+v %v", g, ok)
	}
	if !strings.Contains(read(t, path), "[mailer.sendgrid]\nkey = 'k'") {
		t.Errorf("section dropped:\n%s", read(t, path))
	}
}

func TestOptionalSectionPresence(t *testing.T) {
	path := write(t, "[mailer.sendgrid]\n")
	h := start(t, path)
	// An empty table is present, and stays in the file.
	if _, ok := h.sendgrid.Get(); !ok {
		t.Error("empty table is absent")
	}
	if !strings.Contains(read(t, path), "[mailer.sendgrid]") {
		t.Errorf("empty table dropped:\n%s", read(t, path))
	}
	// A section with defaults is present without its table.
	if c, ok := h.smtp.Get(); !ok || c.Addr != ":587" {
		t.Errorf("smtp %+v %v", c, ok)
	}

	h = start(t, write(t, ""))
	if g, ok := h.sendgrid.Get(); ok {
		t.Errorf("no table but present: %+v", g)
	}
}

func TestBrokenSyntaxIsRejected(t *testing.T) {
	if err := initCur(write(t, "[server\nlisten = \n")); err == nil {
		t.Fatal("broken TOML accepted")
	}
}

func TestFinishRejectsWhatASectionCannotUse(t *testing.T) {
	for name, body := range map[string]string{
		"wrong type":   "[server]\nlisten = 5\n",
		"decode error": "[mailer.smtp]\nport = 0\n",
	} {
		path := write(t, body)
		initAt(t, path)
		Of(cur, serverSection)
		OfOptional(cur, smtpSection)
		if err := cur.Finish(); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if read(t, path) != body {
			t.Errorf("%s: the file was written", name)
		}
	}
}

// A key a version dropped (or a typo) must not stop the server: it is named in the log and
// left out when the file is rewritten.
func TestUnknownKeysAndTablesAreIgnored(t *testing.T) {
	logs := captureLog(t)
	path := write(t, "[server]\napp_url = \"https://a.example\"\nlisten_addr = \"x\"\n[old]\nkey = 1\n[mailer.nothing]\nx = 1\n")
	h := start(t, path)
	if h.server.Get().AppURL != "https://a.example" {
		t.Error("known key lost")
	}
	for _, want := range []string{"server.listen_addr", "old", "mailer.nothing"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("%q not named in the log: %s", want, logs.String())
		}
	}
	text := read(t, path)
	for _, gone := range []string{"listen_addr", "[old]", "nothing"} {
		if strings.Contains(text, gone) {
			t.Errorf("%q kept in the rewritten file:\n%s", gone, text)
		}
	}
}

// A section inside another one is not the parent's key: the parent neither reads it nor
// complains about it, whatever order they are asked for in.
func TestNestedSectionIsNotAKeyOfItsParent(t *testing.T) {
	logs := captureLog(t)
	initAt(t, write(t, "[mailer]\n[mailer.smtp]\nhost = \"h\"\n"))
	smtp := OfOptional(cur, smtpSection)
	Of(cur, mailerSection)
	finish(t)
	if strings.Contains(logs.String(), "unknown") {
		t.Errorf("a nested section reported as unknown: %s", logs.String())
	}
	if c, ok := smtp.Get(); !ok || c.Addr != "h:587" {
		t.Errorf("smtp %+v %v", c, ok)
	}
}

func TestOfGivesOneValue(t *testing.T) {
	initAt(t, write(t, ""))
	a := Of(cur, serverSection)
	b := Of(cur, serverSection)
	finish(t)
	if err := serverSection.Update(cur, func(s *server) { s.AppURL = "u" }); err != nil {
		t.Fatal(err)
	}
	if a.Get().AppURL != "u" || b.Get().AppURL != "u" {
		t.Error("two handles to one section differ")
	}
}

func TestUpdatePersistsAndKeepsTheRest(t *testing.T) {
	path := write(t, "[server]\napp_url = \"https://a.example\"\n[mailer.sendgrid]\nkey = \"k\"\n")
	h := start(t, path)
	if err := serverSection.Update(cur, func(s *server) { s.AppURL = "https://b.example" }); err != nil {
		t.Fatal(err)
	}
	if h.server.Get().AppURL != "https://b.example" {
		t.Errorf("handle not updated: %q", h.server.Get().AppURL)
	}
	again := start(t, path)
	if again.server.Get().AppURL != "https://b.example" {
		t.Error("not persisted")
	}
	if g, ok := again.sendgrid.Get(); !ok || g.Key != "k" {
		t.Errorf("another section lost by the write: %+v %v", g, ok)
	}
}

func TestUpdateIsDecodedAndRejected(t *testing.T) {
	path := write(t, "")
	h := start(t, path)
	before := read(t, path)
	if err := smtpSection.Update(cur, func(c *smtpCfg) { c.Port = -1 }); err == nil {
		t.Fatal("a value Decode refuses was saved")
	}
	if c, _ := h.smtp.Get(); c.Addr != ":587" {
		t.Errorf("a rejected value was kept: %+v", c)
	}
	if read(t, path) != before {
		t.Error("a rejected value was written")
	}
	if err := smtpSection.Update(cur, func(c *smtpCfg) { c.Host = "h" }); err != nil {
		t.Fatal(err)
	}
	if c, _ := h.smtp.Get(); c.Addr != "h:587" {
		t.Errorf("not decoded: %+v", c)
	}
}

func TestOptionalUpdateAddsAndRemoveDrops(t *testing.T) {
	path := write(t, "")
	h := start(t, path)
	if err := sendgridSection.Update(cur, func(s *sendgrid) { s.Key = "k" }); err != nil {
		t.Fatal(err)
	}
	if g, ok := h.sendgrid.Get(); !ok || g.Key != "k" {
		t.Errorf("after add: %+v %v", g, ok)
	}
	if !strings.Contains(read(t, path), "[mailer.sendgrid]") {
		t.Errorf("added section not written:\n%s", read(t, path))
	}
	if err := sendgridSection.Remove(cur); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.sendgrid.Get(); ok {
		t.Error("still present after Remove")
	}
	if strings.Contains(read(t, path), "sendgrid") {
		t.Errorf("removed section still written:\n%s", read(t, path))
	}
	if err := sendgridSection.Remove(cur); err != nil {
		t.Errorf("removing an absent section: %v", err)
	}
}

func TestSubscribersHearChanges(t *testing.T) {
	h := start(t, write(t, ""))
	var got []string
	cancel := h.server.Subscribe(func(s server) { got = append(got, s.AppURL) })
	_ = serverSection.Update(cur, func(s *server) { s.AppURL = "a" })
	_ = serverSection.Update(cur, func(s *server) { s.Listen = "" }) // a refused change is not heard: none here
	cancel()
	_ = serverSection.Update(cur, func(s *server) { s.AppURL = "b" })
	if len(got) != 2 || got[0] != "a" || got[1] != "a" {
		t.Errorf("heard %v", got)
	}

	var present []bool
	h.sendgrid.Subscribe(func(_ sendgrid, ok bool) { present = append(present, ok) })
	_ = sendgridSection.Update(cur, func(*sendgrid) {})
	_ = sendgridSection.Remove(cur)
	if len(present) != 2 || !present[0] || present[1] {
		t.Errorf("heard %v", present)
	}
}

func TestReadOnlyFile(t *testing.T) {
	path := write(t, "[server]\napp_url = \"https://a.example\"\n")
	// A directory where the temp file goes makes every write fail, like a read-only mount.
	if err := os.Mkdir(path+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	h := start(t, path)
	if cur.Writable() {
		t.Error("write failed but the file is reported writable")
	}
	if err := serverSection.Update(cur, func(s *server) { s.AppURL = "https://b.example" }); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Update on a read-only file: %v", err)
	}
	if h.server.Get().AppURL != "https://a.example" {
		t.Errorf("value changed despite the failure: %q", h.server.Get().AppURL)
	}
}

func TestUpdateFailureKeepsTheValue(t *testing.T) {
	path := write(t, "")
	h := start(t, path)
	// The write fails after start, as when the disk fills up.
	if err := os.Mkdir(path+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := serverSection.Update(cur, func(s *server) { s.AppURL = "x" }); err == nil {
		t.Fatal("a failed write was reported as done")
	}
	if h.server.Get().AppURL != "" {
		t.Errorf("value kept after a failed write: %q", h.server.Get().AppURL)
	}
}

func TestUpdateBeforeFinishIsRefused(t *testing.T) {
	initAt(t, write(t, ""))
	Of(cur, serverSection)
	if err := serverSection.Update(cur, func(s *server) { s.AppURL = "x" }); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Update before Finish: %v", err)
	}
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: no panic", name)
		}
	}()
	fn()
}

func TestMistakesInWiringPanic(t *testing.T) {
	start(t, write(t, ""))
	mustPanic(t, "new section after Finish", func() { Of(cur, Section[server, server]{Name: "other"}) })
	mustPanic(t, "update of a section never asked for", func() {
		_ = Section[server, server]{Name: "other"}.Update(cur, func(*server) {})
	})

	initAt(t, write(t, ""))
	Of(cur, serverSection)
	mustPanic(t, "another type under the same name", func() { Of(cur, Section[mailer, mailer]{Name: "server"}) })
	mustPanic(t, "no Decode between different types", func() { Of(cur, Section[server, conn]{Name: "x"}) })
}

func TestFixedAndNilHandles(t *testing.T) {
	r := Fixed(server{AppURL: "u"})
	if r.Get().AppURL != "u" {
		t.Error("Fixed")
	}
	r.Subscribe(func(server) {})()
	var n *Ref[server]
	if n.Get() != (server{}) {
		t.Error("nil Ref")
	}
	var o *OptRef[server]
	if _, ok := o.Get(); ok {
		t.Error("nil OptRef is present")
	}
}

func TestConcurrentReadsAndUpdates(t *testing.T) {
	h := start(t, write(t, ""))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _ = h.server.Get() }()
		go func() { defer wg.Done(); _ = serverSection.Update(cur, func(s *server) { s.AppURL = "u" }) }()
	}
	wg.Wait()
	if got := h.server.Get().AppURL; got != "u" {
		t.Errorf("the update is lost: %q", got)
	}
}
