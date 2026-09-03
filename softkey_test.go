package main

import (
	"strings"
	"testing"
)

func TestHashIdentifiers_Basic(t *testing.T) {
	h1 := hashIdentifiers("a", "b", "c")
	h2 := hashIdentifiers("a", "b", "c")
	if h1 != h2 {
		t.Fatalf("hashIdentifiers not deterministic: %s != %s", h1, h2)
	}
	if len(h1) != 64 { // SHA-256 hex = 64 chars
		t.Fatalf("expected 64 hex chars, got %d", len(h1))
	}
}

func TestHashIdentifiers_OrderMatters(t *testing.T) {
	h1 := hashIdentifiers("a", "b")
	h2 := hashIdentifiers("b", "a")
	if h1 == h2 {
		t.Fatal("hashIdentifiers should be order-sensitive")
	}
}

func TestHashIdentifiers_CollisionResistance(t *testing.T) {
	// "a" + "b" vs "ab" — NUL separator should prevent prefix collision
	h1 := hashIdentifiers("a", "b")
	h2 := hashIdentifiers("ab")
	if h1 == h2 {
		t.Fatal("NUL separator should prevent prefix collision")
	}
}

func TestHashIdentifiers_Empty(t *testing.T) {
	h := hashIdentifiers()
	if h != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("empty hash should be SHA-256 of empty string, got %s", h)
	}
}

func TestHashIdentifiers_DuplicateParts(t *testing.T) {
	h1 := hashIdentifiers("x", "y")
	h2 := hashIdentifiers("x", "y", "x", "y")
	if h1 == h2 {
		t.Fatal("duplicate parts in input should change hash")
	}
}

func TestAppendNonEmpty(t *testing.T) {
	cases := []struct {
		name     string
		initial  []string
		vals     []string
		expected []string
	}{
		{
			name:     "empty input",
			initial:  nil,
			vals:     []string{},
			expected: nil,
		},
		{
			name:     "all empty strings",
			initial:  nil,
			vals:     []string{"", "  ", "\t"},
			expected: nil,
		},
		{
			name:     "trim and deduplicate",
			initial:  []string{"a"},
			vals:     []string{"  b  ", "a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "whitespace-only values skipped",
			initial:  nil,
			vals:     []string{"", "  ", "\n", "valid"},
			expected: []string{"valid"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := appendNonEmpty(tc.initial, tc.vals...)
			if len(result) != len(tc.expected) {
				t.Fatalf("expected %v, got %v", tc.expected, result)
			}
			for i := range tc.expected {
				if result[i] != tc.expected[i] {
					t.Fatalf("expected %v, got %v", tc.expected, result)
				}
			}
		})
	}
}

func TestReadFileTrimmed(t *testing.T) {
	// Test with a known file (this test file itself)
	result := readFileTrimmed("/etc/hostname")
	// Should not error, result may or may not be empty depending on system
	_ = result

	// Test with non-existent file
	result = readFileTrimmed("/nonexistent/path/that/does/not/exist")
	if result != "" {
		t.Fatalf("expected empty string for non-existent file, got %q", result)
	}
}

func TestGetMachineId_NonEmpty(t *testing.T) {
	id := getMachineId()
	if id == "" {
		t.Fatal("getMachineId() returned empty string — at least one identifier source should work")
	}
	if len(id) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(id))
	}
	// Verify it's valid hex
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("invalid hex char %q in machine ID %q", string(c), id)
		}
	}
}

func TestGetMachineId_Stable(t *testing.T) {
	id1 := getMachineId()
	id2 := getMachineId()
	if id1 != id2 {
		t.Fatalf("getMachineId() not stable: %s != %s", id1, id2)
	}
}

func TestMachineIdUint32(t *testing.T) {
	u := machineIdUint32()
	id := getMachineId()
	if id == "" {
		if u != 0 {
			t.Fatal("machineIdUint32 should be 0 when getMachineId is empty")
		}
		return
	}
	// Should be non-zero for a valid machine
	if u == 0 {
		t.Logf("machineIdUint32 returned 0 for non-empty machine ID %s (possible if first 8 hex chars are 0)", id)
	}
}

func TestDebugIdentifiers(t *testing.T) {
	parts := debugIdentifiers()
	// Should return at least one identifier on any supported platform
	if len(parts) == 0 {
		t.Fatal("debugIdentifiers() returned empty slice — at least one source should work")
	}
	for _, p := range parts {
		if p == "" {
			t.Fatal("debugIdentifiers() should not contain empty strings")
		}
	}
}

func TestCollectIdentifiers_PartsFormat(t *testing.T) {
	parts := debugIdentifiers()
	for _, p := range parts {
		// Parts should be trimmed (no leading/trailing whitespace)
		if strings.TrimSpace(p) != p {
			t.Fatalf("identifier part has whitespace: %q", p)
		}
	}
}
