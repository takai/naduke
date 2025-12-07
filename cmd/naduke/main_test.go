package main

import (
	"path/filepath"
	"testing"
)

func TestParseArgsDirMustExist(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")
	if _, _, _, _, err := parseArgs([]string{"-dir", missing, "file.txt"}); err == nil {
		t.Fatalf("expected error for missing dir")
	}

	existing := t.TempDir()
	if _, _, _, _, err := parseArgs([]string{"-dir", existing, "file.txt"}); err != nil {
		t.Fatalf("unexpected error for existing dir: %v", err)
	}
}
