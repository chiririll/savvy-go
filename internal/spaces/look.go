package spaces

import (
	"regexp"
	"strings"
	"unicode"

	repo "savvy-go/internal/db/repo/spaces"
)

// Look is how a space is drawn beside its name: one emoji and a #rrggbb color,
// each optional. They are display only and never part of anything signed.
type Look = repo.Look

// LookError is a bad emoji or color, reported as a 422 on Field.
type LookError struct{ Field, Msg string }

func (e *LookError) Error() string { return e.Field + ": " + e.Msg }

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// maxEmojiBytes bounds the longest real sequence (a ZWJ family with skin tones)
// with room to spare.
const maxEmojiBytes = 48

// Normalize checks a look and returns it trimmed, with the color in lower case
// and an empty value turned into none.
func Normalize(l Look) (Look, error) {
	if l.Emoji != nil {
		e := strings.TrimSpace(*l.Emoji)
		l.Emoji = nil
		if e != "" {
			if !isEmojiSequence(e) {
				return l, &LookError{Field: "emoji", Msg: "The emoji must be a single emoji."}
			}
			l.Emoji = &e
		}
	}
	if l.Color != nil {
		c := strings.TrimSpace(*l.Color)
		l.Color = nil
		if c != "" {
			if !colorPattern.MatchString(c) {
				return l, &LookError{Field: "color", Msg: "The color must look like #rrggbb."}
			}
			c = strings.ToLower(c)
			l.Color = &c
		}
	}
	return l, nil
}

// isEmojiSequence reports whether s is one user-perceived symbol: a base
// character followed only by modifiers, joiners and what they join, or a pair
// of regional indicators (a flag). Letters are not emoji; a digit is only the
// base of a keycap (1️⃣).
func isEmojiSequence(s string) bool {
	rs := []rune(s)
	if len(rs) == 0 || len(s) > maxEmojiBytes {
		return false
	}
	first := rs[0]
	if unicode.IsLetter(first) || unicode.IsSpace(first) || unicode.IsControl(first) {
		return false
	}
	isRegional := func(r rune) bool { return r >= 0x1F1E6 && r <= 0x1F1FF }
	if isRegional(first) {
		return len(rs) == 2 && isRegional(rs[1])
	}
	joined := false
	for _, r := range rs[1:] {
		switch {
		case joined:
			joined = false
		case r == 0x200D:
			joined = true
		case r == 0xFE0E || r == 0xFE0F || r == 0x20E3,
			r >= 0x1F3FB && r <= 0x1F3FF, // skin tones
			r >= 0xE0020 && r <= 0xE007F: // tag characters (subdivision flags)
		default:
			return false
		}
	}
	return !joined
}
