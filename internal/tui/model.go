package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"tools/jira/pkg"
)

// ActivityMsg wraps a pkg.ActivityEvent for the Bubble Tea loop.
type ActivityMsg pkg.ActivityEvent

// IssuesUpdatedMsg conveys a refreshed list of issues.
type IssuesUpdatedMsg []pkg.Issue

// DigestMsg conveys a detected catch-up digest.
type DigestMsg *pkg.CatchupDigest

// ClearStatusMsg is sent after a delay to clear temporary status messages.
type ClearStatusMsg struct{}

// Model represents the state of the TUI Jira Cockpit.
type Model struct {
	space         string
	baseURL       string
	watchEngine   *pkg.WatchEngine
	cacheDB       *pkg.CacheDB
	client        *pkg.Client

	issues        []pkg.Issue
	cursor        int
	activities    []pkg.ActivityEvent
	digest        *pkg.CatchupDigest
	showDigest    bool

	width         int
	height        int

	syncState     string
	lastSyncTime  time.Time
	statusMessage string
	quitting      bool
}

// NewModel creates an initialized TUI Model.
func NewModel(space, baseURL string, engine *pkg.WatchEngine, cache *pkg.CacheDB, client *pkg.Client) Model {
	initialIssues := []pkg.Issue{}
	if cache != nil {
		if cached, err := cache.GetIssues(space, ""); err == nil && len(cached) > 0 {
			initialIssues = cached
		}
	}
	if engine != nil && len(initialIssues) == 0 {
		initialIssues = engine.GetMemoryIssues()
	}

	initialActivities := []pkg.ActivityEvent{}
	if cache != nil {
		if acts, err := cache.GetRecentActivities(20, space); err == nil {
			initialActivities = acts
		}
	}

	var digest *pkg.CatchupDigest
	showDigest := false
	if engine != nil {
		digest = engine.GetCatchupDigest()
		if digest != nil {
			showDigest = true
		}
	}

	return Model{
		space:         space,
		baseURL:       baseURL,
		watchEngine:   engine,
		cacheDB:       cache,
		client:        client,
		issues:        initialIssues,
		cursor:        0,
		activities:    initialActivities,
		digest:        digest,
		showDigest:    showDigest,
		syncState:     "LIVE WATCHING",
		lastSyncTime:  time.Now(),
		statusMessage: "Ready. [j/k] navigate, [r] sync, [o] browser, [c] active, [q] quit",
	}
}

// Init sets up initial commands for the Bubble Tea loop.
func (m Model) Init() tea.Cmd {
	return waitForActivityEvent(m.watchEngine)
}

// waitForActivityEvent returns a tea.Cmd that waits for the next ActivityEvent.
func waitForActivityEvent(engine *pkg.WatchEngine) tea.Cmd {
	if engine == nil {
		return nil
	}
	return func() tea.Msg {
		ch := engine.EventChannel()
		if ch == nil {
			return nil
		}
		evt, ok := <-ch
		if !ok {
			return nil
		}
		return ActivityMsg(evt)
	}
}

// clearStatusAfter creates a command that clears the status after a duration.
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// pollSyncCmd executes an on-demand sync cycle.
func pollSyncCmd(engine *pkg.WatchEngine, space string, cache *pkg.CacheDB) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		_, _ = engine.PollNow()
		var updated []pkg.Issue
		if cache != nil {
			updated, _ = cache.GetIssues(space, "")
		} else {
			updated = engine.GetMemoryIssues()
		}
		return IssuesUpdatedMsg(updated)
	}
}

// Update handles incoming messages and events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.issues)-1 {
				m.cursor++
			}

		case "g", "home":
			m.cursor = 0

		case "G", "end":
			if len(m.issues) > 0 {
				m.cursor = len(m.issues) - 1
			}

		case "esc":
			if m.showDigest {
				m.showDigest = false
				m.statusMessage = "Welcome back briefing dismissed"
				cmds = append(cmds, clearStatusAfter(3*time.Second))
			}

		case "r":
			m.syncState = "SYNCING..."
			m.statusMessage = "Manual sync triggered..."
			cmds = append(cmds, pollSyncCmd(m.watchEngine, m.space, m.cacheDB))

		case "o":
			if len(m.issues) > 0 && m.cursor >= 0 && m.cursor < len(m.issues) {
				selected := m.issues[m.cursor]
				url := formatIssueURL(m.baseURL, selected.Key)
				err := openInBrowser(url)
				if err != nil {
					m.statusMessage = fmt.Sprintf("Failed to open browser: %v", err)
				} else {
					m.statusMessage = fmt.Sprintf("Opened %s in web browser", selected.Key)
				}
				cmds = append(cmds, clearStatusAfter(3*time.Second))
			}

		case "c":
			if len(m.issues) > 0 && m.cursor >= 0 && m.cursor < len(m.issues) {
				selected := m.issues[m.cursor]
				if m.cacheDB != nil {
					_ = m.cacheDB.SetActiveIssue(selected.Key)
				}
				m.statusMessage = fmt.Sprintf("Active ticket set to [%s] (Saved to local cache)", selected.Key)
				cmds = append(cmds, clearStatusAfter(3*time.Second))
			}
		}

	case ActivityMsg:
		evt := pkg.ActivityEvent(msg)
		m.lastSyncTime = time.Now()
		m.syncState = "LIVE WATCHING"

		// Prepend new activity
		m.activities = append([]pkg.ActivityEvent{evt}, m.activities...)
		if len(m.activities) > 50 {
			m.activities = m.activities[:50]
		}

		// Refresh issues from cache
		if m.cacheDB != nil {
			if fresh, err := m.cacheDB.GetIssues(m.space, ""); err == nil && len(fresh) > 0 {
				m.updateIssuesList(fresh)
			}
		} else if m.watchEngine != nil {
			m.updateIssuesList(m.watchEngine.GetMemoryIssues())
		}

		m.statusMessage = fmt.Sprintf("Live Event: %s (%s)", evt.IssueKey, evt.EventType)
		cmds = append(cmds, clearStatusAfter(4*time.Second))

		// Continue listening
		cmds = append(cmds, waitForActivityEvent(m.watchEngine))

	case IssuesUpdatedMsg:
		m.syncState = "LIVE WATCHING"
		m.lastSyncTime = time.Now()
		fresh := []pkg.Issue(msg)
		if len(fresh) > 0 {
			m.updateIssuesList(fresh)
			m.statusMessage = fmt.Sprintf("Synced with Jira (%d active issues)", len(fresh))
		} else {
			m.statusMessage = "Sync completed: no issues found"
		}
		cmds = append(cmds, clearStatusAfter(3*time.Second))

	case DigestMsg:
		d := (*pkg.CatchupDigest)(msg)
		if d != nil {
			m.digest = d
			m.showDigest = true
		}

	case ClearStatusMsg:
		m.statusMessage = ""
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateIssuesList(fresh []pkg.Issue) {
	currentSelectedKey := ""
	if len(m.issues) > 0 && m.cursor >= 0 && m.cursor < len(m.issues) {
		currentSelectedKey = m.issues[m.cursor].Key
	}
	m.issues = fresh
	newCursor := 0
	if currentSelectedKey != "" {
		for i, it := range m.issues {
			if it.Key == currentSelectedKey {
				newCursor = i
				break
			}
		}
	}
	if newCursor >= len(m.issues) && len(m.issues) > 0 {
		newCursor = len(m.issues) - 1
	}
	m.cursor = newCursor
}

func formatIssueURL(baseURL, key string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/browse/%s", baseURL, strings.ToUpper(key))
}

func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// RunTUI launches the Bubble Tea Jira Cockpit with the alternative screen buffer.
func RunTUI(ctx context.Context, space, baseURL string, engine *pkg.WatchEngine, cache *pkg.CacheDB, client *pkg.Client) error {
	m := NewModel(space, baseURL, engine, cache, client)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
