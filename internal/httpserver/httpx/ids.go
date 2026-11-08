package httpx

import (
	"net/url"
	"strconv"
	"strings"
)

// QueryIDs reads a list of ids from key or key[], each value comma-separated.
// ok is false when a part is not a positive id, so a typo is not read as
// "no filter".
func QueryIDs(q url.Values, key string) (ids []int64, ok bool) {
	for _, raw := range append(q[key], q[key+"[]"]...) {
		for _, part := range strings.Split(raw, ",") {
			if part = strings.TrimSpace(part); part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				return nil, false
			}
			ids = append(ids, id)
		}
	}
	return ids, true
}
