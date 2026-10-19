package app

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tools/jira/pkg"
)

func TestBuildTargetURL(t *testing.T) {
	tests := []struct {
		name        string
		instanceURL string
		projectKey  string
		issueKey    string
		expected    string
	}{
		{
			name:        "Issue key provided",
			instanceURL: "https://my-company.atlassian.net",
			projectKey:  "PROJ",
			issueKey:    "PROJ-123",
			expected:    "https://my-company.atlassian.net/browse/PROJ-123",
		},
		{
			name:        "Issue key lowercase normalization",
			instanceURL: "https://my-company.atlassian.net/",
			projectKey:  "PROJ",
			issueKey:    "proj-123",
			expected:    "https://my-company.atlassian.net/browse/PROJ-123",
		},
		{
			name:        "No issue key, uses project browse URL",
			instanceURL: "https://my-company.atlassian.net",
			projectKey:  "TEAM",
			issueKey:    "",
			expected:    "https://my-company.atlassian.net/browse/TEAM",
		},
		{
			name:        "No issue key, empty project key defaults to KAN",
			instanceURL: "https://my-company.atlassian.net",
			projectKey:  "",
			issueKey:    "",
			expected:    "https://my-company.atlassian.net/browse/KAN",
		},
		{
			name:        "Trailing slash in instance URL handled correctly",
			instanceURL: "https://my-company.atlassian.net///",
			projectKey:  "JC",
			issueKey:    "JC-3",
			expected:    "https://my-company.atlassian.net/browse/JC-3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildTargetURL(tc.instanceURL, tc.projectKey, tc.issueKey)
			if got != tc.expected {
				t.Errorf("BuildTargetURL() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestBuildBrowserCommand(t *testing.T) {
	targetURL := "https://my-company.atlassian.net/browse/JC-3"

	t.Run("OS defaults", func(t *testing.T) {
		cmdDarwin, argsDarwin := BuildBrowserCommand(targetURL, "", "darwin")
		if cmdDarwin != "open" || !reflect.DeepEqual(argsDarwin, []string{targetURL}) {
			t.Errorf("darwin: got %s %v, want open [%s]", cmdDarwin, argsDarwin, targetURL)
		}

		cmdLinux, argsLinux := BuildBrowserCommand(targetURL, "", "linux")
		if cmdLinux != "xdg-open" || !reflect.DeepEqual(argsLinux, []string{targetURL}) {
			t.Errorf("linux: got %s %v, want xdg-open [%s]", cmdLinux, argsLinux, targetURL)
		}

		cmdWin, argsWin := BuildBrowserCommand(targetURL, "", "windows")
		if cmdWin != "cmd" || !reflect.DeepEqual(argsWin, []string{"/c", "start", "", targetURL}) {
			t.Errorf("windows: got %s %v, want cmd [/c start \"\" %s]", cmdWin, argsWin, targetURL)
		}
	})

	t.Run("Custom browser flag takes highest priority", func(t *testing.T) {
		t.Setenv("JIRA_BROWSER", "env-browser")
		t.Setenv("BROWSER", "system-browser")

		cmd, args := BuildBrowserCommand(targetURL, "firefox --private-window", "linux")
		if cmd != "firefox" {
			t.Errorf("expected cmd 'firefox', got %q", cmd)
		}
		expectedArgs := []string{"--private-window", targetURL}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Errorf("expected args %v, got %v", expectedArgs, args)
		}
	})

	t.Run("JIRA_BROWSER takes priority over BROWSER and OS default", func(t *testing.T) {
		t.Setenv("JIRA_BROWSER", "google-chrome")
		t.Setenv("BROWSER", "firefox")

		cmd, args := BuildBrowserCommand(targetURL, "", "linux")
		if cmd != "google-chrome" || !reflect.DeepEqual(args, []string{targetURL}) {
			t.Errorf("expected google-chrome [%s], got %s %v", targetURL, cmd, args)
		}
	})

	t.Run("BROWSER environment variable used if set", func(t *testing.T) {
		t.Setenv("JIRA_BROWSER", "")
		t.Setenv("BROWSER", "chromium")

		cmd, args := BuildBrowserCommand(targetURL, "", "linux")
		if cmd != "chromium" || !reflect.DeepEqual(args, []string{targetURL}) {
			t.Errorf("expected chromium [%s], got %s %v", targetURL, cmd, args)
		}
	})
}

func TestOpenCommand_Execute(t *testing.T) {
	origResolver := pkg.GitBranchResolver
	pkg.GitBranchResolver = func(ctx context.Context) (string, error) { return "main", nil }
	defer func() { pkg.GitBranchResolver = origResolver }()

	origCacheResolver := pkg.CachedActiveIssueResolver
	defer func() { pkg.CachedActiveIssueResolver = origCacheResolver }()
	pkg.CachedActiveIssueResolver = func() string { return "" }

	mockClientProvider := func() (*pkg.Client, error) {
		return pkg.NewClientWithConfig(pkg.Config{
			InstanceURL: "https://example.atlassian.net",
			ProjectKey:  "TEST",
		}), nil
	}

	t.Run("URL-only flag outputs URL without launching browser", func(t *testing.T) {
		var launched bool
		cmd := NewOpenCommand(mockClientProvider).WithLauncher(func(ctx context.Context, name string, args []string) error {
			launched = true
			return nil
		})

		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"--url-only", "TEST-42"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if launched {
			t.Error("launcher should not have been called when --url-only is set")
		}

		expectedOutput := "https://example.atlassian.net/browse/TEST-42\n"
		if stdout.String() != expectedOutput {
			t.Errorf("stdout = %q, want %q", stdout.String(), expectedOutput)
		}
	})

	t.Run("Positional arg before flag (e.g. KEY --url-only)", func(t *testing.T) {
		cmd := NewOpenCommand(mockClientProvider)
		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"TEST-42", "--url-only"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedOutput := "https://example.atlassian.net/browse/TEST-42\n"
		if stdout.String() != expectedOutput {
			t.Errorf("stdout = %q, want %q", stdout.String(), expectedOutput)
		}
	})

	t.Run("Help flag returns 0 without calling client or launcher", func(t *testing.T) {
		failingClientProvider := func() (*pkg.Client, error) {
			return nil, fmt.Errorf("should not be called")
		}
		cmd := NewOpenCommand(failingClientProvider)
		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"--help"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("expected nil error on --help, got: %v", err)
		}
		if !strings.Contains(stdout.String(), "jira open") {
			t.Errorf("expected help output, got: %q", stdout.String())
		}
	})

	t.Run("URL-only without issue key prints project board URL", func(t *testing.T) {
		cmd := NewOpenCommand(mockClientProvider)
		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"--url-only"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedOutput := "https://example.atlassian.net/browse/TEST\n"
		if stdout.String() != expectedOutput {
			t.Errorf("stdout = %q, want %q", stdout.String(), expectedOutput)
		}
	})

	t.Run("Launches browser with issue URL", func(t *testing.T) {
		var executedCmd string
		var executedArgs []string
		cmd := NewOpenCommand(mockClientProvider).
			WithGOOS("linux").
			WithLauncher(func(ctx context.Context, name string, args []string) error {
				executedCmd = name
				executedArgs = args
				return nil
			})

		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"TEST-100"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if executedCmd != "xdg-open" {
			t.Errorf("executedCmd = %q, want 'xdg-open'", executedCmd)
		}
		expectedURL := "https://example.atlassian.net/browse/TEST-100"
		if len(executedArgs) != 1 || executedArgs[0] != expectedURL {
			t.Errorf("executedArgs = %v, want [%s]", executedArgs, expectedURL)
		}
		if !strings.Contains(stdout.String(), expectedURL) {
			t.Errorf("stdout should contain target URL notice, got %q", stdout.String())
		}
	})

	t.Run("Graceful fallback on launcher failure", func(t *testing.T) {
		cmd := NewOpenCommand(mockClientProvider).
			WithGOOS("linux").
			WithLauncher(func(ctx context.Context, name string, args []string) error {
				return fmt.Errorf("no display available")
			})

		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"TEST-99"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("execute should succeed gracefully (exit code 0), but returned: %v", err)
		}

		expectedURL := "https://example.atlassian.net/browse/TEST-99"
		if !strings.Contains(stderr.String(), expectedURL) {
			t.Errorf("stderr should contain fallback URL message, got %q", stderr.String())
		}
	})
}
