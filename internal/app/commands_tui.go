package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"tools/jira/internal/i18n"
	"tools/jira/internal/tui"
	"tools/jira/pkg"
)

// TUICommand launches the interactive terminal cockpit dashboard.
type TUICommand struct {
	clientProvider ClientProvider
}

func NewTUICommand(cp ClientProvider) *TUICommand {
	return &TUICommand{clientProvider: cp}
}

func (c *TUICommand) Name() string        { return "tui" }
func (c *TUICommand) Aliases() []string  { return []string{"cockpit", "dashboard"} }
func (c *TUICommand) Description() string { return i18n.T("cmd.tui.desc") }

func (c *TUICommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var project string
	var intervalSec int

	fs.StringVar(&project, "p", "", "Jira project space key")
	fs.StringVar(&project, "project", "", "Jira project space key")
	fs.IntVar(&intervalSec, "interval", 15, "Background sync polling interval in seconds")

	fs.Usage = func() {
		fmt.Fprintln(stdout, "Usage: jira tui [-p/--project <KEY>] [--interval <sec>]")
		fs.SetOutput(stdout)
		fs.PrintDefaults()
		fs.SetOutput(stderr)
	}

	if isHelpRequested(args) {
		fs.Usage()
		return nil
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	if strings.TrimSpace(project) == "" {
		project = client.GetConfig().ProjectKey
		if project == "" {
			project = "KAN"
		}
	}

	cachePath, err := pkg.GetDefaultCachePath()
	if err != nil {
		return fmt.Errorf("failed to get cache path: %w", err)
	}

	cacheDB, err := pkg.OpenCacheDB(cachePath)
	if err != nil {
		return fmt.Errorf("failed to initialize SQLite cache: %w", err)
	}
	defer cacheDB.Close()

	pollInterval := time.Duration(intervalSec) * time.Second
	engine := pkg.NewWatchEngine(client, cacheDB, project, pollInterval)

	// Load cached state and check for catch-up digest (0ms startup)
	_, _ = engine.InitState(ctx)

	// Start background watcher
	engine.Start(ctx)
	defer engine.Stop()

	baseURL := client.GetConfig().InstanceURL
	return tui.RunTUI(ctx, project, baseURL, engine, cacheDB, client)
}

// WatchCommand streams real-time issue updates and activity events to stdout.
type WatchCommand struct {
	clientProvider ClientProvider
}

func NewWatchCommand(cp ClientProvider) *WatchCommand {
	return &WatchCommand{clientProvider: cp}
}

func (c *WatchCommand) Name() string        { return "watch" }
func (c *WatchCommand) Aliases() []string  { return []string{"tail"} }
func (c *WatchCommand) Description() string { return i18n.T("cmd.watch.desc") }

func (c *WatchCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var project string
	var intervalSec int
	var jsonOutput bool

	fs.StringVar(&project, "p", "", "Jira project space key")
	fs.StringVar(&project, "project", "", "Jira project space key")
	fs.IntVar(&intervalSec, "interval", 15, "Polling interval in seconds")
	fs.BoolVar(&jsonOutput, "json", false, "Output events as JSON stream")

	fs.Usage = func() {
		fmt.Fprintln(stdout, "Usage: jira watch [-p/--project <KEY>] [--interval <sec>] [--json]")
		fs.SetOutput(stdout)
		fs.PrintDefaults()
		fs.SetOutput(stderr)
	}

	if isHelpRequested(args) {
		fs.Usage()
		return nil
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	if strings.TrimSpace(project) == "" {
		project = client.GetConfig().ProjectKey
		if project == "" {
			project = "KAN"
		}
	}

	cachePath, err := pkg.GetDefaultCachePath()
	if err != nil {
		return fmt.Errorf("failed to get cache path: %w", err)
	}

	cacheDB, err := pkg.OpenCacheDB(cachePath)
	if err != nil {
		return fmt.Errorf("failed to initialize SQLite cache: %w", err)
	}
	defer cacheDB.Close()

	pollInterval := time.Duration(intervalSec) * time.Second
	engine := pkg.NewWatchEngine(client, cacheDB, project, pollInterval)

	digest, _ := engine.InitState(ctx)
	if digest != nil && !jsonOutput {
		fmt.Fprint(stdout, digest.FormatSummary())
	}

	if !jsonOutput {
		fmt.Fprintf(stdout, "⚡ Watching project [%s] (Polling every %ds). Press Ctrl+C to exit.\n", project, intervalSec)
	}

	engine.Start(ctx)
	defer engine.Stop()

	ch := engine.EventChannel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				return nil
			}
			if jsonOutput {
				data, _ := json.Marshal(evt)
				fmt.Fprintln(stdout, string(data))
			} else {
				fmt.Fprintf(stdout, "[%s] %-14s %s: %s\n",
					evt.CreatedAt.Format("15:04:05"),
					"["+evt.EventType+"]",
					evt.IssueKey,
					evt.Details,
				)
			}
		}
	}
}

// UseCommand sets or inspects the active ticket key in the local cache.
type UseCommand struct {
	clientProvider ClientProvider
}

func NewUseCommand(cp ClientProvider) *UseCommand {
	return &UseCommand{clientProvider: cp}
}

func (c *UseCommand) Name() string        { return "use" }
func (c *UseCommand) Aliases() []string  { return []string{"set-active", "current"} }
func (c *UseCommand) Description() string { return i18n.T("cmd.use.desc") }

func (c *UseCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("use", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var clear bool
	fs.BoolVar(&clear, "clear", false, "Clear current active issue context")

	fs.Usage = func() {
		fmt.Fprintln(stdout, "Usage: jira use [KEY] [--clear]")
		fs.SetOutput(stdout)
		fs.PrintDefaults()
		fs.SetOutput(stderr)
	}

	if isHelpRequested(args) {
		fs.Usage()
		return nil
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	cachePath, err := pkg.GetDefaultCachePath()
	if err != nil {
		return fmt.Errorf("failed to get cache path: %w", err)
	}

	cacheDB, err := pkg.OpenCacheDB(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open local cache: %w", err)
	}
	defer cacheDB.Close()

	if clear {
		if err := cacheDB.ClearActiveIssue(); err != nil {
			return fmt.Errorf("failed to clear active issue: %w", err)
		}
		fmt.Fprintln(stdout, "Active issue context cleared.")
		return nil
	}

	posArgs := fs.Args()
	if len(posArgs) == 0 {
		active, err := cacheDB.GetActiveIssue()
		if err != nil {
			return fmt.Errorf("failed to retrieve active issue: %w", err)
		}
		if active == "" {
			fmt.Fprintln(stdout, "No active issue is set in local cache.")
		} else {
			fmt.Fprintf(stdout, "Current active issue: %s\n", active)
		}
		return nil
	}

	targetKey := strings.ToUpper(strings.TrimSpace(posArgs[0]))
	if err := cacheDB.SetActiveIssue(targetKey); err != nil {
		return fmt.Errorf("failed to set active issue: %w", err)
	}

	fmt.Fprintf(stdout, "Active issue set to: %s\n", targetKey)
	return nil
}

// CacheCommand manages the local SQLite cache.
type CacheCommand struct {
	clientProvider ClientProvider
}

func NewCacheCommand(cp ClientProvider) *CacheCommand {
	return &CacheCommand{clientProvider: cp}
}

func (c *CacheCommand) Name() string        { return "cache" }
func (c *CacheCommand) Aliases() []string  { return []string{} }
func (c *CacheCommand) Description() string { return i18n.T("cmd.cache.desc") }

func (c *CacheCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || isHelpRequested(args) {
		fmt.Fprintln(stdout, "Usage: jira cache <command> [arguments]")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Commands:")
		fmt.Fprintln(stdout, "  clean    Reset and clean the local SQLite cache")
		fmt.Fprintln(stdout, "  status   Show cache DB location and statistics")
		return nil
	}

	cachePath, err := pkg.GetDefaultCachePath()
	if err != nil {
		return fmt.Errorf("failed to get cache path: %w", err)
	}

	subcmd := args[0]
	switch subcmd {
	case "clean", "clear":
		cacheDB, err := pkg.OpenCacheDB(cachePath)
		if err == nil {
			_ = cacheDB.Close()
		}
		if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete cache database: %w", err)
		}
		// Also clean WAL files
		_ = os.Remove(cachePath + "-wal")
		_ = os.Remove(cachePath + "-shm")

		fmt.Fprintln(stdout, "Local SQLite cache database cleaned successfully.")
		return nil

	case "status", "info":
		fi, err := os.Stat(cachePath)
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "Cache DB does not exist yet (Run 'jira list' or 'jira tui' to populate).")
			return nil
		}
		if err != nil {
			return err
		}

		cacheDB, err := pkg.OpenCacheDB(cachePath)
		if err != nil {
			return fmt.Errorf("failed to open cache: %w", err)
		}
		defer cacheDB.Close()

		issues, _ := cacheDB.GetIssues("", "")
		active, _ := cacheDB.GetActiveIssue()
		activities, _ := cacheDB.GetRecentActivities(100, "")

		fmt.Fprintf(stdout, "Path:            %s\n", cachePath)
		fmt.Fprintf(stdout, "Size:            %.2f KB\n", float64(fi.Size())/1024.0)
		fmt.Fprintf(stdout, "Cached Issues:   %d\n", len(issues))
		fmt.Fprintf(stdout, "Recent Events:   %d\n", len(activities))
		if active != "" {
			fmt.Fprintf(stdout, "Active Ticket:   %s\n", active)
		} else {
			fmt.Fprintln(stdout, "Active Ticket:   (None)")
		}
		return nil

	default:
		return fmt.Errorf("unknown cache subcommand: %s (use 'clean' or 'status')", subcmd)
	}
}
