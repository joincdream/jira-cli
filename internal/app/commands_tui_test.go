package app

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"tools/jira/pkg"
)

func TestUseCommand_And_ResolveIssueKeyFallback(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tempDir)

	// Ensure git branch resolver returns "main" (no issue key in branch)
	oldResolver := pkg.GitBranchResolver
	pkg.GitBranchResolver = func(ctx context.Context) (string, error) {
		return "main", nil
	}
	defer func() { pkg.GitBranchResolver = oldResolver }()

	clientProvider := func() (*pkg.Client, error) {
		return nil, nil
	}

	useCmd := NewUseCommand(clientProvider)

	// 1. Initial use (no active issue set)
	var outBuf, errBuf bytes.Buffer
	err := useCmd.Execute(context.Background(), []string{}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "No active issue is set") {
		t.Errorf("expected 'No active issue is set', got: %s", outBuf.String())
	}

	// 2. Set active issue to JC-17
	outBuf.Reset()
	err = useCmd.Execute(context.Background(), []string{"jc-17"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error setting active: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Active issue set to: JC-17") {
		t.Errorf("expected active issue confirmation, got: %s", outBuf.String())
	}

	// 3. Inspect active issue
	outBuf.Reset()
	err = useCmd.Execute(context.Background(), []string{}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error checking active: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Current active issue: JC-17") {
		t.Errorf("expected current active issue JC-17, got: %s", outBuf.String())
	}

	// 4. Test ResolveIssueKey fallback: on "main" branch with zero arguments, it should resolve to JC-17!
	resolvedKey, err := pkg.ResolveIssueKey(context.Background(), "")
	if err != nil {
		t.Fatalf("expected fallback to active cache key, got err: %v", err)
	}
	if resolvedKey != "JC-17" {
		t.Errorf("expected resolved key JC-17, got: %s", resolvedKey)
	}

	// 5. Clear active issue
	outBuf.Reset()
	err = useCmd.Execute(context.Background(), []string{"--clear"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error clearing active: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Active issue context cleared") {
		t.Errorf("expected cleared notice, got: %s", outBuf.String())
	}

	// 6. After clear, ResolveIssueKey on "main" should fail with no key
	_, err = pkg.ResolveIssueKey(context.Background(), "")
	if err == nil {
		t.Errorf("expected error when no key is in branch or cache")
	}
}

func TestCacheCommand_Status_And_Clean(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tempDir)

	cachePath, _ := pkg.GetDefaultCachePath()
	cacheDB, err := pkg.OpenCacheDB(cachePath)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	_ = cacheDB.UpsertIssues("JC", []pkg.Issue{
		{Key: "JC-1", Fields: pkg.IssueFields{Summary: "Test 1", Status: pkg.Status{Name: "To Do"}}},
	})
	_ = cacheDB.SetActiveIssue("JC-1")
	cacheDB.Close()

	cacheCmd := NewCacheCommand(func() (*pkg.Client, error) { return nil, nil })

	// Test status
	var outBuf, errBuf bytes.Buffer
	err = cacheCmd.Execute(context.Background(), []string{"status"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error in cache status: %v", err)
	}
	outStr := outBuf.String()
	if !strings.Contains(outStr, "Cached Issues:   1") || !strings.Contains(outStr, "Active Ticket:   JC-1") {
		t.Errorf("unexpected status output:\n%s", outStr)
	}

	// Test clean
	outBuf.Reset()
	err = cacheCmd.Execute(context.Background(), []string{"clean"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error in cache clean: %v", err)
	}
	if !strings.Contains(outBuf.String(), "cleaned successfully") {
		t.Errorf("unexpected clean output: %s", outBuf.String())
	}

	// Verify file was deleted
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Errorf("expected cache file to be deleted, but still exists")
	}
}

func TestListAndGet_WithCachedFlag(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tempDir)

	cachePath, _ := pkg.GetDefaultCachePath()
	cacheDB, err := pkg.OpenCacheDB(cachePath)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	_ = cacheDB.UpsertIssues("JC", []pkg.Issue{
		{
			Key: "JC-100",
			Fields: pkg.IssueFields{
				Summary: "Cached Offline Issue",
				Status:  pkg.Status{Name: "진행 중"},
				Updated: "2026-10-17T12:00:00.000+0900",
			},
		},
	})
	cacheDB.Close()

	mockClient := pkg.NewClientWithConfig(pkg.Config{
		ProjectKey: "JC",
	})
	clientProvider := func() (*pkg.Client, error) {
		return mockClient, nil
	}

	listCmd := NewListCommand(clientProvider)
	getCmd := NewGetCommand(clientProvider)

	// 1. List with --cached
	var outBuf, errBuf bytes.Buffer
	err = listCmd.Execute(context.Background(), []string{"--cached"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error with list --cached: %v", err)
	}
	if !strings.Contains(outBuf.String(), "JC-100") || !strings.Contains(outBuf.String(), "Cached Offline Issue") {
		t.Errorf("expected JC-100 in list --cached output, got:\n%s", outBuf.String())
	}

	// 2. Get with --cached
	outBuf.Reset()
	err = getCmd.Execute(context.Background(), []string{"JC-100", "--cached"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error with get --cached: %v", err)
	}
	if !strings.Contains(outBuf.String(), "JC-100") || !strings.Contains(outBuf.String(), "Cached Offline Issue") {
		t.Errorf("expected JC-100 in get --cached output, got:\n%s", outBuf.String())
	}
}
