package pkg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"tools/jira/internal/i18n"
)

var (
	issueKeyRegex  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*-[0-9]+$`)
	branchKeyRegex = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])([A-Za-z][A-Za-z0-9_]*-[0-9]+)(?:[^0-9]|$)`)
)

// IsIssueKey checks if a string matches the standard Jira issue key pattern (e.g., PROJ-123).
func IsIssueKey(str string) bool {
	return issueKeyRegex.MatchString(strings.TrimSpace(str))
}

// ExtractIssueKey extracts the first Jira issue key from a git branch name or string.
// Returns the upper-cased issue key and true if found, otherwise ("", false).
func ExtractIssueKey(branch string) (string, bool) {
	matches := branchKeyRegex.FindStringSubmatch(branch)
	if len(matches) > 1 {
		return strings.ToUpper(matches[1]), true
	}
	return "", false
}

// GitBranchResolver is the function used by GetCurrentBranch to find the active branch.
// Exported for testing purposes.
var GitBranchResolver = GetCurrentBranchDefault

// GetCurrentBranch resolves the active Git branch name.
func GetCurrentBranch(ctx context.Context) (string, error) {
	return GitBranchResolver(ctx)
}

// GetCurrentBranchDefault tries git CLI first, then falls back to direct .git/HEAD parsing.
func GetCurrentBranchDefault(ctx context.Context) (string, error) {
	// 1. Try git rev-parse with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		if branch != "" && branch != "HEAD" {
			return branch, nil
		}
	}

	// 2. Fallback: Parse .git/HEAD directly
	return readGitHeadFromDisk()
}

func readGitHeadFromDisk() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepath.Join(dir, ".git")
		fi, err := os.Stat(gitPath)
		if err == nil {
			if fi.IsDir() {
				headPath := filepath.Join(gitPath, "HEAD")
				return parseHeadFile(headPath)
			}
			// Worktree or submodule where .git is a file
			content, err := os.ReadFile(gitPath)
			if err == nil {
				text := strings.TrimSpace(string(content))
				if strings.HasPrefix(text, "gitdir:") {
					gitDir := strings.TrimSpace(strings.TrimPrefix(text, "gitdir:"))
					if !filepath.IsAbs(gitDir) {
						gitDir = filepath.Join(dir, gitDir)
					}
					return parseHeadFile(filepath.Join(gitDir, "HEAD"))
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("not a git repository")
}

func parseHeadFile(headPath string) (string, error) {
	content, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(content))
	if strings.HasPrefix(text, "ref: refs/heads/") {
		branch := strings.TrimPrefix(text, "ref: refs/heads/")
		return branch, nil
	}
	if text != "" {
		return "", errors.New("detached HEAD")
	}
	return "", errors.New("empty HEAD file")
}

// ResolveIssueKey returns explicitKey if provided (uppercased), or automatically extracts
// ResolveIssueKey returns explicitKey if provided (uppercased), or automatically extracts
// the issue key from the current Git branch, or falls back to the locally cached active issue.
func ResolveIssueKey(ctx context.Context, explicitKey string) (string, error) {
	trimmed := strings.TrimSpace(explicitKey)
	if trimmed != "" {
		return strings.ToUpper(trimmed), nil
	}

	branch, branchErr := GetCurrentBranch(ctx)
	if branchErr == nil {
		if key, ok := ExtractIssueKey(branch); ok {
			return key, nil
		}
	}

	// 3. Fallback to active issue stored in local SQLite cache
	if cachedKey := GetCachedActiveIssue(); cachedKey != "" {
		return cachedKey, nil
	}

	if branchErr != nil {
		return "", fmt.Errorf("%s", i18n.T("cmd.common.err_git_context_required"))
	}

	return "", fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_no_issue_key_in_branch", branch))
}
