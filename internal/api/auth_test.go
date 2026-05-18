package api

import "testing"

func TestBearerToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		header   string
		wantTok  string
		wantOK   bool
	}{
		{"canonical scheme", "Bearer abc123", "abc123", true},
		{"lowercase scheme", "bearer abc123", "abc123", true},
		{"uppercase scheme", "BEARER abc123", "abc123", true},
		{"mixed case scheme", "BeArEr abc123", "abc123", true},
		{"trailing whitespace on token", "Bearer abc123   ", "abc123", true},
		{"leading whitespace on token", "Bearer    abc123", "abc123", true},
		{"empty header", "", "", false},
		{"scheme only", "Bearer", "", false},
		{"scheme with only spaces", "Bearer    ", "", false},
		{"wrong scheme", "Basic abc123", "", false},
		{"no space separator", "Bearerabc123", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotTok, gotOK := bearerToken(tc.header)
			if gotOK != tc.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tc.wantOK)
			}
			if gotTok != tc.wantTok {
				t.Fatalf("token = %q, want %q", gotTok, tc.wantTok)
			}
		})
	}
}
