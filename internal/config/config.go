package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is process-wide runtime configuration loaded from the environment.
type Config struct {
	AppURL        string
	ListenAddr    string
	DataDir       string
	Database      string
	UploadsDir    string
	BackupsDir    string
	PublicDir     string
	TZ            string
	Location      *time.Location
	SessionTTL    time.Duration
	RememberTTL   time.Duration
	ChallengeTTL  time.Duration
	SessionCookie string
	CSRFCookie    string
	CSRFHeader    string
	SeedDemo      bool
	AppKey        string
}

// FromEnv loads configuration. DATA_DIR defaults to /data or /var/lib/savvy
// when those deploy paths exist, otherwise ./data.
func FromEnv() Config {
	dataDir := firstNonEmpty(os.Getenv("DATA_DIR"), detectDataDir())
	tz := firstNonEmpty(os.Getenv("TZ"), "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
		tz = "UTC"
	}

	cfg := Config{
		AppURL:        strings.TrimRight(firstNonEmpty(os.Getenv("APP_URL"), "http://localhost:8080"), "/"),
		ListenAddr:    firstNonEmpty(os.Getenv("LISTEN_ADDR"), os.Getenv("HTTP_ADDR"), "localhost:8080"),
		DataDir:       dataDir,
		Database:      firstNonEmpty(os.Getenv("DB_DATABASE"), filepath.Join(dataDir, "database.sqlite")),
		UploadsDir:    firstNonEmpty(os.Getenv("UPLOAD_ROOT"), filepath.Join(dataDir, "uploads")),
		BackupsDir:    firstNonEmpty(os.Getenv("BACKUP_PATH"), filepath.Join(dataDir, "backups")),
		PublicDir:     firstNonEmpty(os.Getenv("PUBLIC_DIR"), "public"),
		TZ:            tz,
		Location:      loc,
		SessionTTL:    minutesEnv("AUTH_SESSION_TTL", 60*24),
		RememberTTL:   minutesEnv("AUTH_SESSION_REMEMBER_TTL", 60*24*7),
		ChallengeTTL:  minutesEnv("AUTH_SESSION_CHALLENGE_TTL", 5),
		SessionCookie: firstNonEmpty(os.Getenv("AUTH_SESSION_COOKIE"), "svy_session"),
		CSRFCookie:    firstNonEmpty(os.Getenv("AUTH_SESSION_CSRF_COOKIE"), "svy_csrf"),
		CSRFHeader:    firstNonEmpty(os.Getenv("AUTH_SESSION_CSRF_HEADER"), "X-CSRF-Token"),
		SeedDemo:      truthy(os.Getenv("SEED_DEMO")),
	}
	cfg.AppKey = loadAppKey(dataDir)
	return cfg
}

func loadAppKey(dataDir string) string {
	if v := strings.TrimSpace(os.Getenv("APP_KEY")); v != "" {
		return v
	}
	for _, p := range []string{
		filepath.Join(dataDir, ".env_config"),
		filepath.Join(dataDir, ".env"),
		".env",
	} {
		if k := appKeyFromDotenv(p); k != "" {
			return k
		}
	}
	return ""
}

func appKeyFromDotenv(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "APP_KEY" {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		return val
	}
	return ""
}

func detectDataDir() string {
	for _, candidate := range []string{"/data", "/var/lib/savvy-go"} {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}
	return "./data"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func minutesEnv(key string, fallback int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(fallback) * time.Minute
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return time.Duration(fallback) * time.Minute
	}
	return time.Duration(n) * time.Minute
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
