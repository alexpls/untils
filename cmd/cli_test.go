package main

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "strips trailing slash",
			input: "http://untils.localhost:4200/",
			want:  "http://untils.localhost:4200",
		},
		{
			name:  "keeps path",
			input: "https://untils.example.com/base/",
			want:  "https://untils.example.com/base",
		},
		{
			name:    "rejects missing host",
			input:   "http:///app",
			wantErr: true,
		},
		{
			name:    "rejects unsupported scheme",
			input:   "mailto:test@example.com",
			wantErr: true,
		},
		{
			name:    "rejects query",
			input:   "https://untils.example.com?foo=bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeBaseURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
