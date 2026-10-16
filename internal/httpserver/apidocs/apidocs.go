// Package apidocs assembles the OpenAPI document. Each handler package embeds its own
// openapi.yaml (paths plus the schemas it owns) and passes it to Add from its Routes
// function; base.yaml holds what is shared. Spec merges them all.
package apidocs

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

//go:embed base.yaml
var base []byte

//go:embed swagger.html
var swaggerPage []byte

var (
	mu     sync.Mutex
	frags  [][]byte
	seen   = map[[32]byte]bool{}
	cached []byte
)

// Add registers a package's fragment. Adding the same bytes again is a no-op, so a package
// may call it from every one of its route functions.
func Add(fragment []byte) {
	sum := sha256.Sum256(fragment)
	mu.Lock()
	defer mu.Unlock()
	if seen[sum] {
		return
	}
	seen[sum] = true
	frags = append(frags, fragment)
	cached = nil
}

// Spec returns the merged document: base first, then fragments in registration order.
func Spec() ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	if cached != nil {
		return cached, nil
	}
	out, err := Merge(base, frags...)
	if err != nil {
		return nil, err
	}
	cached = out
	return out, nil
}

// Routes serves the Swagger UI page and the spec, relative to the /api mount.
func Routes(r chi.Router) {
	r.Get("/docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(swaggerPage)
	})
	r.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		out, err := Spec()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(out)
	})
}

// Merge adds the paths and components of each fragment to base. A path or component
// defined twice, a fragment key other than paths/components, or a $ref that resolves to
// nothing is an error.
func Merge(base []byte, fragments ...[]byte) ([]byte, error) {
	root, err := parse(base)
	if err != nil {
		return nil, fmt.Errorf("base: %w", err)
	}
	for i, f := range fragments {
		frag, err := parse(f)
		if err != nil {
			return nil, fmt.Errorf("fragment %d: %w", i, err)
		}
		if err := mergeInto(root, frag); err != nil {
			return nil, fmt.Errorf("fragment %d: %w", i, err)
		}
	}
	if err := checkRefs(root, root); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	return buf.Bytes(), enc.Close()
}

func parse(b []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("not a YAML mapping")
	}
	return doc.Content[0], nil
}

// get returns the value node of key in a mapping, or nil.
func get(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// section returns the mapping under key, creating it at the end when absent.
func section(m *yaml.Node, key string) *yaml.Node {
	if n := get(m, key); n != nil {
		n.Style = 0 // an empty `{}` placeholder in base becomes a block mapping
		return n
	}
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, n)
	return n
}

func appendUnique(dst, src *yaml.Node, what string) error {
	for i := 0; i+1 < len(src.Content); i += 2 {
		k := src.Content[i].Value
		if get(dst, k) != nil {
			return fmt.Errorf("duplicate %s %s", what, k)
		}
		dst.Content = append(dst.Content, src.Content[i], src.Content[i+1])
	}
	return nil
}

func mergeInto(root, frag *yaml.Node) error {
	for i := 0; i+1 < len(frag.Content); i += 2 {
		key, val := frag.Content[i].Value, frag.Content[i+1]
		switch key {
		case "paths":
			if err := appendUnique(section(root, "paths"), val, "path"); err != nil {
				return err
			}
		case "components":
			dst := section(root, "components")
			for j := 0; j+1 < len(val.Content); j += 2 {
				kind := val.Content[j].Value
				if err := appendUnique(section(dst, kind), val.Content[j+1], "component "+kind+"/"); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("top-level key %q is only allowed in base.yaml", key)
		}
	}
	return nil
}

// checkRefs verifies every local $ref in n points at a node of root.
func checkRefs(root, n *yaml.Node) error {
	if n.Kind == yaml.MappingNode {
		if ref := get(n, "$ref"); ref != nil && strings.HasPrefix(ref.Value, "#/") {
			cur := root
			for _, seg := range strings.Split(ref.Value[2:], "/") {
				if cur = get(cur, seg); cur == nil {
					return fmt.Errorf("unresolved $ref %s", ref.Value)
				}
			}
		}
	}
	for _, c := range n.Content {
		if err := checkRefs(root, c); err != nil {
			return err
		}
	}
	return nil
}
