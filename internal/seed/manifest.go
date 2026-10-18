package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Manifest describes what Demo created, so tools that drive the app (such as the
// screenshot runner) can sign in and reach every seeded page without knowing ids.
type Manifest struct {
	// Now is the moment the data was placed relative to: the "today" of the seeded
	// data. It carries the time zone offset.
	Now         time.Time            `json:"now"`
	Users       []ManifestUser       `json:"users"`
	Spaces      []ManifestSpace      `json:"spaces"`
	Invitations []ManifestInvitation `json:"invitations"`
}

// ManifestUser is a seeded account. Role is the server role (admin, user, guest).
type ManifestUser struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	// Verified is whether the user proved the address; it only shows with mail on.
	Verified bool `json:"verified"`
}

// ManifestSpace is a seeded space. Members maps a user key to the role in it.
type ManifestSpace struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	Emoji       string            `json:"emoji,omitempty"`
	Color       string            `json:"color,omitempty"`
	Members     map[string]string `json:"members"`
	Automations []int64           `json:"automations,omitempty"`
	// SyncGroup is the id of the sync group the space is in, 0 for none.
	SyncGroup int64 `json:"sync_group,omitempty"`
}

// ManifestInvitation is an open invitation; Token is the secret in its link.
type ManifestInvitation struct {
	SpaceID int64  `json:"space_id"`
	Email   string `json:"email,omitempty"`
	Role    string `json:"role"`
	Token   string `json:"token"`
}

// WriteFile writes the manifest as indented JSON, creating the directory.
func (m *Manifest) WriteFile(path string) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
