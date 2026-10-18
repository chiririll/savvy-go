package httpx

import (
	"encoding/json"
	"io"
	"net/http"
)

// DecodeSent reads the JSON body over v, which holds the stored values on
// update, so fields the body leaves out keep them. It also returns the keys
// the body sent, for fields where an empty list differs from an absent one.
// On a bad body it answers 422 under errField and returns false.
func DecodeSent(w http.ResponseWriter, r *http.Request, v any, errField string) (sent map[string]json.RawMessage, ok bool) {
	data, err := io.ReadAll(r.Body)
	if err == nil {
		err = json.Unmarshal(data, &sent)
	}
	if err == nil {
		err = json.Unmarshal(data, v)
	}
	if err != nil {
		WriteValidation(w, map[string][]string{errField: {"The given data was invalid."}})
		return nil, false
	}
	return sent, true
}
