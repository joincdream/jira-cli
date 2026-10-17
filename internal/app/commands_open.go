package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"tools/jira/internal/i18n"
	"tools/jira/pkg"
)

// BrowserLauncher defines a function signature for launching the system browser.
type BrowserLauncher func(ctx context.Context, name string, args []string) error

// DefaultBrowserLauncher starts the process asynchronously using os/exec.
func DefaultBrowserLauncher(ctx context.Context, name string, args []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Start()
}

// OpenCommand opens Jira tickets or boards in the web browser.
type OpenCommand struct {
	clientProvider ClientProvider
	launcher       BrowserLauncher
	goos           string
}

// NewOpenCommand creates an OpenCommand with the given ClientProvider.
func NewOpenCommand(cp ClientProvider) *OpenCommand {
	return &OpenCommand{
		clientProvider: cp,
		launcher:       DefaultBrowserLauncher,
		goos:           runtime.GOOS,
	}
}

// WithLauncher sets a custom browser launcher (useful for testing).
func (c *OpenCommand) WithLauncher(launcher BrowserLauncher) *OpenCommand {
	c.launcher = launcher
	return c
}

// WithGOOS sets a custom GOOS value (useful for testing cross-platform commands).
func (c *OpenCommand) WithGOOS(goos string) *OpenCommand {
	c.goos = goos
	return c
}

func (c *OpenCommand) Name() string        { return "open" }
func (c *OpenCommand) Aliases() []string  { return []string{"browse"} }
func (c *OpenCommand) Description() string { return i18n.T("cmd.open.desc") }

func (c *OpenCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.open.usage"))
		return nil
	}

	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var customBrowser string
	var urlOnly bool

	fs.StringVar(&customBrowser, "browser", "", i18n.T("cmd.open.flag_browser"))
	fs.BoolVar(&urlOnly, "url-only", false, i18n.T("cmd.open.flag_url_only"))

	posArgs, err := parseFlagsAndPositional(fs, args)
	if err != nil {
		return err
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cmd.common.err_client_init"), err)
	}

	cfg := client.GetConfig()
	var targetKey string
	if len(posArgs) > 0 {
		targetKey = posArgs[0].Value
	} else {
		// Zero-argument: check if current git branch contains an issue key
		branch, err := pkg.GetCurrentBranch(ctx)
		if err == nil {
			if key, ok := pkg.ExtractIssueKey(branch); ok {
				targetKey = key
			}
		}
	}

	targetURL := BuildTargetURL(cfg.InstanceURL, cfg.ProjectKey, targetKey)

	if urlOnly {
		fmt.Fprintln(stdout, targetURL)
		return nil
	}

	cmdName, cmdArgs := BuildBrowserCommand(targetURL, customBrowser, c.goos)

	fmt.Fprintf(stdout, i18n.Sprintf("cmd.open.opening", targetURL))

	if err := c.launcher(ctx, cmdName, cmdArgs); err != nil {
		fmt.Fprintf(stderr, i18n.Sprintf("cmd.open.fallback", targetURL))
		return nil
	}

	return nil
}

// BuildTargetURL constructs the web destination URL based on ticket key and project key.
func BuildTargetURL(instanceURL, projectKey, issueKey string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(instanceURL), "/")
	key := strings.TrimSpace(issueKey)
	if key == "" {
		key = strings.TrimSpace(projectKey)
		if key == "" {
			key = "KAN"
		}
	}

	return fmt.Sprintf("%s/browse/%s", baseURL, strings.ToUpper(key))
}

// BuildBrowserCommand resolves the executable and arguments according to flags, env, and OS.
func BuildBrowserCommand(targetURL, customBrowser, goos string) (string, []string) {
	// 1. Explicit CLI flag
	if strings.TrimSpace(customBrowser) != "" {
		return parseCustomBrowserCommand(customBrowser, targetURL)
	}

	// 2. JIRA_BROWSER environment variable
	if env := strings.TrimSpace(os.Getenv("JIRA_BROWSER")); env != "" {
		return parseCustomBrowserCommand(env, targetURL)
	}

	// 3. Standard BROWSER environment variable
	if env := strings.TrimSpace(os.Getenv("BROWSER")); env != "" {
		return parseCustomBrowserCommand(env, targetURL)
	}

	// 4. OS-specific default launcher
	switch goos {
	case "darwin":
		return "open", []string{targetURL}
	case "windows":
		return "cmd", []string{"/c", "start", "", targetURL}
	default: // "linux", "freebsd", "openbsd", "netbsd", etc.
		return "xdg-open", []string{targetURL}
	}
}

func parseCustomBrowserCommand(commandStr, targetURL string) (string, []string) {
	fields := strings.Fields(commandStr)
	if len(fields) == 0 {
		return "xdg-open", []string{targetURL}
	}
	if len(fields) == 1 {
		return fields[0], []string{targetURL}
	}
	args := append(fields[1:], targetURL)
	return fields[0], args
}
