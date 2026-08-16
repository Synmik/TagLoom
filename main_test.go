package main

import (
	"testing"
)

// TestProductVersionFromWailsJSON guards the embedded wails.json: it must
// parse and carry a non-empty info.productVersion, because that value is
// what the app reports as its version (single source of truth — see
// productVersionFromWailsJSON).
func TestProductVersionFromWailsJSON(t *testing.T) {
	v, err := productVersionFromWailsJSON(wailsJSON)
	if err != nil {
		t.Fatalf("productVersionFromWailsJSON: %v", err)
	}
	if v == "" {
		t.Fatal("productVersion is empty")
	}
	t.Logf("productVersion = %q", v)
}

// TestProductVersionFromWailsJSONGarbage ensures malformed input is a hard
// error rather than a silent empty version.
func TestProductVersionFromWailsJSONGarbage(t *testing.T) {
	if _, err := productVersionFromWailsJSON([]byte("{not json")); err == nil {
		t.Fatal("expected error for malformed wails.json")
	}
	if _, err := productVersionFromWailsJSON([]byte(`{"info":{}}`)); err == nil {
		t.Fatal("expected error when info.productVersion is missing")
	}
}
