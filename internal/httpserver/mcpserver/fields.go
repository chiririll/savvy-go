package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// jsonKeys is the JSON keys of a struct's fields, for the allowed set of parseFields.
func jsonKeys(v any) map[string]bool {
	t := reflect.TypeOf(v)
	keys := make(map[string]bool, t.NumField())
	for f := range t.Fields() {
		if name, _, _ := strings.Cut(f.Tag.Get("json"), ","); name != "" && name != "-" {
			keys[name] = true
		}
	}
	return keys
}

var fieldPathRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*$`)

// fieldTree is the paths of fields as a tree. A node without children takes the
// whole value.
type fieldTree map[string]fieldTree

// parseFields checks the paths of a tool's fields argument. Deeper segments
// name keys of nested objects (payments.accountId) and apply to each element
// of a list. With allowed, the first segment must be one of its keys; without,
// any key is accepted and a misspelled one just matches nothing.
func parseFields(paths []string, allowed map[string]bool) (fieldTree, error) {
	tree := fieldTree{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !fieldPathRe.MatchString(p) {
			return nil, fmt.Errorf("fields: %q is not a field name or a dotted path such as payments.accountId.", p)
		}
		segs := strings.Split(p, ".")
		if allowed != nil && !allowed[segs[0]] {
			return nil, fmt.Errorf("fields: unknown field %q.", segs[0])
		}
		node := tree
		for _, s := range segs {
			if node[s] == nil {
				node[s] = fieldTree{}
			}
			node = node[s]
		}
	}
	if len(tree) == 0 {
		return nil, nil
	}
	return tree, nil
}

// pick keeps of v only what the tree names. A picked parent that has a picked
// child keeps just that child; a parent picked alone is kept whole.
func (t fieldTree) pick(v any) any {
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = t.pick(e)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, sub := range t {
			val, ok := x[k]
			if !ok {
				continue
			}
			if len(sub) == 0 {
				out[k] = val
			} else {
				out[k] = sub.pick(val)
			}
		}
		return out
	}
	return v
}

// shapeData returns a shaper that trims the data of an answer to the tree: each
// row of a list, or the one object. Meta, summary and other keys stay.
func shapeData(tree fieldTree) func([]byte) ([]byte, error) {
	return func(body []byte) ([]byte, error) {
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.UseNumber() // money stays exact
		var page map[string]any
		if err := dec.Decode(&page); err != nil {
			return nil, err
		}
		if data, ok := page["data"]; ok {
			page["data"] = tree.pick(data)
		}
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetEscapeHTML(false) // as the REST API writes it: & stays &
		if err := enc.Encode(page); err != nil {
			return nil, err
		}
		return bytes.TrimSpace(out.Bytes()), nil
	}
}
