package backup

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/mozillazg/go-unidecode"
)

func stamp(t time.Time) string { return t.UTC().Format("20060102-150405") }

// baseName is the readable start of a backup file name: srv for the server,
// otherwise the transliterated space name (space when nothing usable is left),
// plus -deleted for the final backup of a deleted space. Names stay ASCII and
// safe for any file system.
func baseName(m Manifest) string {
	if m.Kind != kindSpace {
		return "srv"
	}
	out := slugify(m.SpaceName)
	if out == "" {
		out = kindSpace
	}
	if m.Note == "deleted" {
		out += "-deleted"
	}
	return out
}

func slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(unidecode.Unidecode(name)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
			if b.Len() >= 40 {
				break
			}
			continue
		}
		dash = true
	}
	return b.String()
}

func validName(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.HasPrefix(name, ".") &&
		strings.HasSuffix(name, zipExt)
}
