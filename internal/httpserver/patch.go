package httpserver

// Updates decode the request body over the stored values, so a PATCH
// changes only the fields it sends and an explicit null clears a nullable
// one. Pointers are cloned first: decoding writes through them.

// clone copies *p, so decoding into the copy leaves the original alone.
func clone[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
