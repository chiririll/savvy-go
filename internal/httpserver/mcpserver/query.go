package mcpserver

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func setDate(q url.Values, key, v string) error {
	if v == "" {
		return nil
	}
	if !dateRe.MatchString(v) {
		return fmt.Errorf("%s must be YYYY-MM-DD.", key)
	}
	q.Set(key, v)
	return nil
}

func setEnum(q url.Values, key, v string, allowed ...string) error {
	if v == "" {
		return nil
	}
	for _, a := range allowed {
		if v == a {
			q.Set(key, v)
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s.", key, strings.Join(allowed, ", "))
}

func setID(q url.Values, key string, id int64) {
	if id > 0 {
		q.Set(key, strconv.FormatInt(id, 10))
	}
}

func setIDs(q url.Values, key string, ids []int64) {
	var parts []string
	for _, id := range ids {
		if id > 0 {
			parts = append(parts, strconv.FormatInt(id, 10))
		}
	}
	if len(parts) > 0 {
		q.Set(key, strings.Join(parts, ","))
	}
}
