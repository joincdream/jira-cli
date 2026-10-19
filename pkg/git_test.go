package pkg

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestIsIssueKey(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"JC-14", true},
		{"kan-25", true},
		{"PROJ_1-999", true},
		{"A-1", true},
		{"main", false},
		{"feature/JC-14", false},
		{"123-ABC", false},
		{"JC", false},
		{"-123", false},
		{"", false},
		{"  JC-14  ", true},
	}

	for _, tt := range tests {
		got := IsIssueKey(tt.input)
		if got != tt.expected {
			t.Errorf("IsIssueKey(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestExtractIssueKey(t *testing.T) {
	tests := []struct {
		branch   string
		expected string
		found    bool
	}{
		{"feature/JC-14-git-binding", "JC-14", true},
		{"fix/kan-25_hotfix", "KAN-25", true},
		{"JC-1", "JC-1", true},
		{"feat/PROJ_9-1234/desc", "PROJ_9-1234", true},
		{"bugfix/jc-99", "JC-99", true},
		{"main", "", false},
		{"master", "", false},
		{"develop", "", false},
		{"release/v1.0.0", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		got, found := ExtractIssueKey(tt.branch)
		if found != tt.found {
			t.Errorf("ExtractIssueKey(%q) found = %v, want %v", tt.branch, found, tt.found)
		}
		if got != tt.expected {
			t.Errorf("ExtractIssueKey(%q) = %q, want %q", tt.branch, got, tt.expected)
		}
	}
}

func TestResolveIssueKey(t *testing.T) {
	ctx := context.Background()

	// 1. Explicit key given -> should return explicit key in upper case
	key, err := ResolveIssueKey(ctx, "kan-10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "KAN-10" {
		t.Fatalf("expected KAN-10, got %s", key)
	}

	// 2. Explicit key empty, mock branch returns feature/JC-14
	origResolver := GitBranchResolver
	origCacheResolver := CachedActiveIssueResolver
	defer func() {
		GitBranchResolver = origResolver
		CachedActiveIssueResolver = origCacheResolver
	}()
	CachedActiveIssueResolver = func() string { return "" }

	GitBranchResolver = func(ctx context.Context) (string, error) {
		return "feature/JC-14-auto-binding", nil
	}

	key, err = ResolveIssueKey(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "JC-14" {
		t.Fatalf("expected JC-14, got %s", key)
	}

	// 3. Explicit key empty, mock branch returns main (no issue key)
	GitBranchResolver = func(ctx context.Context) (string, error) {
		return "main", nil
	}

	_, err = ResolveIssueKey(ctx, "")
	if err == nil {
		t.Fatal("expected error when branch has no issue key, got nil")
	}

	// 4. Git error (e.g. not a git repo)
	GitBranchResolver = func(ctx context.Context) (string, error) {
		return "", errors.New("not a git repo")
	}

	_, err = ResolveIssueKey(ctx, "")
	if err == nil {
		t.Fatal("expected error when git fails, got nil")
	}
}

func TestParseHeadFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid branch ref
	headFile1 := filepath.Join(tmpDir, "HEAD1")
	if err := os.WriteFile(headFile1, []byte("ref: refs/heads/feature/JC-14\n"), 0644); err != nil {
		t.Fatal(err)
	}
	branch, err := parseHeadFile(headFile1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "feature/JC-14" {
		t.Fatalf("expected feature/JC-14, got %s", branch)
	}

	// Detached HEAD (commit hash)
	headFile2 := filepath.Join(tmpDir, "HEAD2")
	if err := os.WriteFile(headFile2, []byte("d9a8c1f938a19283740192830192830192830192\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = parseHeadFile(headFile2)
	if err == nil || err.Error() != "detached HEAD" {
		t.Fatalf("expected detached HEAD error, got %v", err)
	}

	// Empty HEAD
	headFile3 := filepath.Join(tmpDir, "HEAD3")
	if err := os.WriteFile(headFile3, []byte("   \n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = parseHeadFile(headFile3)
	if err == nil {
		t.Fatal("expected error for empty HEAD, got nil")
	}
}
