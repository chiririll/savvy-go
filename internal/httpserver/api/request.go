package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// DecodeBody reads the JSON body into v. It answers 422 and returns false
// when the body is not valid.
func DecodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteValidation(w, map[string][]string{"body": {"The given data was invalid."}})
		return false
	}
	return true
}

// LooksLikeEmail is a cheap shape check: something@host.tld.
func LooksLikeEmail(s string) bool {
	s = strings.TrimSpace(s)
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-3 && strings.Contains(s[at:], ".")
}

// AsBool reads a flag a client may send as a bool, a number or a string.
func AsBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "1" || t == "true"
	default:
		return false
	}
}
