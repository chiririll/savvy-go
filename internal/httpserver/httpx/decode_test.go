package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeSent(t *testing.T) {
	type body struct {
		Name string  `json:"name"`
		IDs  []int64 `json:"ids"`
	}
	cases := []struct {
		name     string
		in       string
		prefill  body
		want     body
		sentIDs  bool
		wantCode int
	}{
		{"omitted keeps prefill", `{"name":"b"}`, body{"a", []int64{1}}, body{"b", []int64{1}}, false, 0},
		{"sent replaces", `{"ids":[]}`, body{"a", []int64{1}}, body{"a", []int64{}}, true, 0},
		{"null is sent", `{"ids":null}`, body{"a", []int64{1}}, body{"a", nil}, true, 0},
		{"not json", `nope`, body{}, body{}, false, 422},
		{"wrong type", `{"name":5}`, body{}, body{}, false, 422},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("PATCH", "/x", strings.NewReader(tc.in))
			got := tc.prefill
			sent, ok := DecodeSent(w, r, &got, "name")
			if tc.wantCode != 0 {
				if ok || w.Code != tc.wantCode {
					t.Fatalf("ok=%v code=%d, want failure %d", ok, w.Code, tc.wantCode)
				}
				return
			}
			if !ok {
				t.Fatalf("failed: %d %s", w.Code, w.Body)
			}
			if got.Name != tc.want.Name || len(got.IDs) != len(tc.want.IDs) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
			if _, has := sent["ids"]; has != tc.sentIDs {
				t.Errorf("ids sent = %v, want %v", has, tc.sentIDs)
			}
		})
	}
}
