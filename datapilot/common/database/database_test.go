package database

import (
	"strings"
	"testing"
)

// TestEnsureDSNParams verifies that required DSN parameters are appended correctly.
func TestEnsureDSNParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSubs []string
	}{
		{
			name:     "no query string — appends all params",
			input:    "user:pass@tcp(localhost:3306)/dbname",
			wantSubs: []string{"charset=utf8mb4", "parseTime=True", "loc=Local"},
		},
		{
			name:     "already has all params — unchanged",
			input:    "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
			wantSubs: []string{"charset=utf8mb4", "parseTime=True", "loc=Local"},
		},
		{
			name:     "partial params — missing ones are appended",
			input:    "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4",
			wantSubs: []string{"charset=utf8mb4", "parseTime=True", "loc=Local"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ensureDSNParams(tc.input)
			for _, sub := range tc.wantSubs {
				if !strings.Contains(result, sub) {
					t.Errorf("expected DSN to contain %q, got: %s", sub, result)
				}
			}
		})
	}
}

// TestEnsureDSNParamsIdempotent verifies that calling ensureDSNParams twice is idempotent.
func TestEnsureDSNParamsIdempotent(t *testing.T) {
	dsn := "user:pass@tcp(localhost:3306)/dbname"
	once := ensureDSNParams(dsn)
	twice := ensureDSNParams(once)
	if once != twice {
		t.Errorf("ensureDSNParams is not idempotent:\nfirst:  %s\nsecond: %s", once, twice)
	}
}
