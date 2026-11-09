package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"tools/jira/pkg"
)

// Cockpit represents the tview-based interactive Jira dashboard.
type Cockpit struct {
	app        *tview.Application
	pages      *tview.Pages
	space      string
	baseURL    string
	engine     *pkg.WatchEngine
	cacheDB    *pkg.CacheDB
	client     *pkg.Client

	issues           []pkg.Issue
	filteredIssues   []pkg.Issue
	selectedStatuses map[string]bool
	filterActive     bool

	activities []pkg.ActivityEvent
	digest     *pkg.CatchupDigest
	showDigest bool

	headerView   *tview.TextView
	digestView   *tview.TextView
	table        *tview.Table
	detailView   *tview.TextView
	activityView *tview.TextView
	footerView   *tview.TextView

	rootFlex      *tview.Flex
	workspaceFlex *tview.Flex

	leftRatio   int
	detailRatio int
	zoomMode    bool

	syncState    string
	lastSyncTime time.Time
	statusMsg    string
	mu           sync.RWMutex
}

// NewCockpit creates and initializes a Cockpit dashboard instance.
func NewCockpit(space, baseURL string, engine *pkg.WatchEngine, cache *pkg.CacheDB, client *pkg.Client) *Cockpit {
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
		if acts, err := cache.GetRecentActivities(30, space); err == nil {
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

	c := &Cockpit{
		app:              tview.NewApplication(),
		pages:            tview.NewPages(),
		space:            space,
		baseURL:          baseURL,
		engine:           engine,
		cacheDB:          cache,
		client:           client,
		issues:           initialIssues,
		filteredIssues:   initialIssues,
		selectedStatuses: make(map[string]bool),
		filterActive:     false,
		activities:       initialActivities,
		digest:           digest,
		showDigest:       showDigest,
		leftRatio:        42,
		detailRatio:      55,
		zoomMode:         false,
		syncState:        "LIVE WATCHING",
		lastSyncTime:     time.Now(),
		statusMsg:        "Ready. [j/k] navigate, [f] filter, [z] zoom, [r] sync, [q] quit",
	}

	c.initUI()
	c.pages.AddPage("main", c.rootFlex, true, true)
	return c
}

func (c *Cockpit) initUI() {
	// 1. Header
	c.headerView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	c.updateHeader()

	// 2. Digest View
	c.digestView = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	c.digestView.SetBorder(true).
		SetTitle(" 🏖️ Welcome Back Briefing (ESC to close) ").
		SetBorderColor(colorAmber)
	c.updateDigest()

	// 3. Issues Table (Left Panel)
	c.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	c.table.SetBorder(true).
		SetTitle(fmt.Sprintf(" 📋 Active Issues (%d) ", len(c.issues))).
		SetBorderColor(colorJiraBlue)

	c.table.SetSelectedStyle(tcell.StyleDefault.
		Background(colorDarkNavy).
		Foreground(tcell.ColorWhite).
		Bold(true))

	c.table.SetSelectionChangedFunc(func(row, column int) {
		c.onIssueSelectionChanged(row)
	})

	// Add j/k key navigation inside Table
	c.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		displayIssues := c.getDisplayIssues()
		switch event.Rune() {
		case 'j':
			row, col := c.table.GetSelection()
			if row < len(displayIssues)-1 {
				c.table.Select(row+1, col)
			}
			return nil
		case 'k':
			row, col := c.table.GetSelection()
			if row > 0 {
				c.table.Select(row-1, col)
			}
			return nil
		case 'g':
			c.table.Select(0, 0)
			return nil
		case 'G':
			if len(displayIssues) > 0 {
				c.table.Select(len(displayIssues)-1, 0)
			}
			return nil
		}
		return event
	})

	c.refreshIssuesTable()

	// 4. Detail View (Right Top)
	c.detailView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	c.detailView.SetBorder(true).
		SetTitle(" 📄 Issue Details ").
		SetBorderColor(colorMuted)

	// 5. Activity View (Right Bottom)
	c.activityView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	c.activityView.SetBorder(true).
		SetTitle(" ⚡ Live Activity Stream ").
		SetBorderColor(colorMuted)
	c.refreshActivities()

	// 6. Footer
	c.footerView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	c.updateFooter()

	// Assemble Workspace Flex
	c.workspaceFlex = tview.NewFlex().SetDirection(tview.FlexColumn)
	c.rebuildWorkspaceFlex()

	// Assemble Root Layout
	c.rootFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	c.rebuildRootLayout()

	// Global Key Bindings
	c.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if c.pages != nil && c.pages.HasPage("filterModal") {
			return event
		}

		switch event.Key() {
		case tcell.KeyCtrlC:
			c.app.Stop()
			return nil
		case tcell.KeyEsc:
			if c.zoomMode {
				c.zoomMode = false
				c.rebuildWorkspaceFlex()
				c.setStatus("Restored split view")
				return nil
			}
			if c.showDigest {
				c.showDigest = false
				c.rebuildRootLayout()
				c.setStatus("Briefing dismissed")
				return nil
			}
		}

		switch event.Rune() {
		case 'q':
			c.app.Stop()
			return nil
		case 'f':
			c.showStatusFilterModal()
			return nil
		case 'z':
			c.zoomMode = !c.zoomMode
			c.rebuildWorkspaceFlex()
			if c.zoomMode {
				c.setStatus("Zoomed into Issue Details ([z] or [ESC] to restore)")
			} else {
				c.setStatus("Restored split view")
			}
			return nil
		case '[':
			if !c.zoomMode && c.leftRatio > 20 {
				c.leftRatio -= 5
				c.rebuildWorkspaceFlex()
				c.setStatus(fmt.Sprintf("Panel width: List %d%% / Details %d%%", c.leftRatio, 100-c.leftRatio))
			}
			return nil
		case ']':
			if !c.zoomMode && c.leftRatio < 80 {
				c.leftRatio += 5
				c.rebuildWorkspaceFlex()
				c.setStatus(fmt.Sprintf("Panel width: List %d%% / Details %d%%", c.leftRatio, 100-c.leftRatio))
			}
			return nil
		case '{':
			if !c.zoomMode && c.detailRatio > 20 {
				c.detailRatio -= 5
				c.rebuildWorkspaceFlex()
				c.setStatus(fmt.Sprintf("Detail height: %d%% / Activity: %d%%", c.detailRatio, 100-c.detailRatio))
			}
			return nil
		case '}':
			if !c.zoomMode && c.detailRatio < 80 {
				c.detailRatio += 5
				c.rebuildWorkspaceFlex()
				c.setStatus(fmt.Sprintf("Detail height: %d%% / Activity: %d%%", c.detailRatio, 100-c.detailRatio))
			}
			return nil
		case 'r':
			c.triggerSync()
			return nil
		case 'o':
			c.openCurrentIssueInBrowser()
			return nil
		case 'c':
			c.setCurrentIssueActive()
			return nil
		}

		return event
	})

	if len(c.issues) > 0 {
		c.onIssueSelectionChanged(0)
	}
}

func (c *Cockpit) rebuildWorkspaceFlex() {
	c.workspaceFlex.Clear()
	if c.zoomMode {
		c.workspaceFlex.AddItem(c.detailView, 0, 1, true)
		c.app.SetFocus(c.detailView)
	} else {
		rightFlex := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(c.detailView, 0, c.detailRatio, false).
			AddItem(c.activityView, 0, 100-c.detailRatio, false)

		c.workspaceFlex.
			AddItem(c.table, 0, c.leftRatio, true).
			AddItem(rightFlex, 0, 100-c.leftRatio, false)

		c.app.SetFocus(c.table)
	}
	c.updateFooter()
}

func (c *Cockpit) rebuildRootLayout() {
	c.rootFlex.Clear()
	c.rootFlex.AddItem(c.headerView, 1, 0, false)
	if c.showDigest && c.digest != nil {
		c.rootFlex.AddItem(c.digestView, 6, 0, false)
	}
	c.rootFlex.AddItem(c.workspaceFlex, 0, 1, true)
	c.rootFlex.AddItem(c.footerView, 1, 0, false)
}

func (c *Cockpit) updateHeader() {
	c.headerView.SetText(buildHeaderContent(c.space, c.syncState, c.lastSyncTime))
}

func (c *Cockpit) updateDigest() {
	if c.digest != nil {
		c.digestView.SetText(buildDigestText(c.digest))
	}
}

func (c *Cockpit) updateFooter() {
	hints := "[#7A869A][↑/↓, j/k] Nav  [f] Filter  [z] Zoom  [[/]] Width  [{/}] Height  [r] Sync  [o] Browser  [c] Active  [q] Quit[-]"
	if c.zoomMode {
		hints = "[#7A869A][↑/↓, j/k] Scroll Details  [z/ESC] Restore  [o] Browser  [q] Quit[-]"
	}
	if c.statusMsg != "" {
		c.footerView.SetText(fmt.Sprintf("%s   [#36B37E:b] %s [-:-]", hints, c.statusMsg))
	} else {
		c.footerView.SetText(hints)
	}
}

func (c *Cockpit) setStatus(msg string) {
	c.statusMsg = msg
	c.updateFooter()
	go func() {
		time.Sleep(3 * time.Second)
		c.app.QueueUpdateDraw(func() {
			c.statusMsg = ""
			c.updateFooter()
		})
	}()
}

func (c *Cockpit) getDisplayIssues() []pkg.Issue {
	if !c.filterActive {
		return c.issues
	}
	return c.filteredIssues
}

func (c *Cockpit) applyFilter() {
	if !c.filterActive {
		c.filteredIssues = c.issues
		return
	}
	c.filteredIssues = FilterIssuesByStatus(c.issues, c.selectedStatuses)
}

func (c *Cockpit) refreshIssuesTable() {
	c.table.Clear()

	displayIssues := c.getDisplayIssues()
	if c.filterActive {
		var activeList []string
		for _, st := range GetUniqueStatuses(c.issues) {
			if c.selectedStatuses[st] {
				activeList = append(activeList, st)
			}
		}
		filterDesc := strings.Join(activeList, ", ")
		if len(filterDesc) > 30 {
			filterDesc = filterDesc[:27] + "..."
		}
		c.table.SetTitle(fmt.Sprintf(" 📋 Issues (%d/%d) [Filter: %s] ", len(displayIssues), len(c.issues), filterDesc))
	} else {
		c.table.SetTitle(fmt.Sprintf(" 📋 Active Issues (%d) ", len(c.issues)))
	}

	if len(displayIssues) == 0 {
		msg := "  No cached issues found. Waiting for sync..."
		if c.filterActive {
			msg = "  No issues match selected filter. ([f] to edit filter)"
		}
		c.table.SetCell(0, 0, tview.NewTableCell(msg).SetTextColor(colorMuted))
		c.detailView.SetText("[#7A869A]No issue selected[-]")
		return
	}

	for i, iss := range displayIssues {
		keyCell := tview.NewTableCell(fmt.Sprintf(" %-7s", iss.Key)).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignLeft)

		statusBadge := formatStatusBadgeTview(iss.Fields.Status.Name)
		statusCell := tview.NewTableCell(statusBadge).
			SetAlign(tview.AlignLeft)

		summaryCell := tview.NewTableCell(tview.Escape(iss.Fields.Summary)).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1).
			SetAlign(tview.AlignLeft)

		c.table.SetCell(i, 0, keyCell)
		c.table.SetCell(i, 1, statusCell)
		c.table.SetCell(i, 2, summaryCell)
	}
}

func (c *Cockpit) onIssueSelectionChanged(row int) {
	displayIssues := c.getDisplayIssues()
	if row < 0 || row >= len(displayIssues) {
		c.detailView.SetText("[#7A869A]No issue selected[-]")
		return
	}
	c.detailView.SetText(buildDetailCardText(&displayIssues[row]))
	c.detailView.ScrollToBeginning()
}

func (c *Cockpit) refreshActivities() {
	c.activityView.SetText(buildActivityFeedText(c.activities))
	c.activityView.ScrollToBeginning()
}

func (c *Cockpit) openCurrentIssueInBrowser() {
	displayIssues := c.getDisplayIssues()
	row, _ := c.table.GetSelection()
	if row >= 0 && row < len(displayIssues) {
		url := formatIssueURL(c.baseURL, displayIssues[row].Key)
		err := openInBrowser(url)
		if err != nil {
			c.setStatus(fmt.Sprintf("Failed to open browser: %v", err))
		} else {
			c.setStatus(fmt.Sprintf("Opened %s in web browser", displayIssues[row].Key))
		}
	}
}

func (c *Cockpit) setCurrentIssueActive() {
	displayIssues := c.getDisplayIssues()
	row, _ := c.table.GetSelection()
	if row >= 0 && row < len(displayIssues) {
		key := displayIssues[row].Key
		if c.cacheDB != nil {
			_ = c.cacheDB.SetActiveIssue(key)
		}
		c.setStatus(fmt.Sprintf("Active ticket set to [%s] (Saved to local cache)", key))
	}
}

func (c *Cockpit) showStatusFilterModal() {
	statuses := GetUniqueStatuses(c.issues)
	if len(statuses) == 0 {
		c.setStatus("No issues/statuses available to filter")
		return
	}

	tempSelection := make(map[string]bool)
	if c.filterActive {
		for k, v := range c.selectedStatuses {
			tempSelection[k] = v
		}
	} else {
		for _, s := range statuses {
			tempSelection[s] = false
		}
	}

	filterTable := tview.NewTable().
		SetSelectable(true, false).
		SetBorders(false)

	renderRows := func() {
		filterTable.Clear()
		for i, st := range statuses {
			checkText := tview.Escape("[ ]")
			textColor := colorMuted
			if tempSelection[st] {
				checkText = tview.Escape("[x]")
				textColor = colorGreen
			}
			badge := formatStatusBadgeTview(st)

			cellCheck := tview.NewTableCell(fmt.Sprintf(" %s ", checkText)).
				SetTextColor(textColor).
				SetAlign(tview.AlignLeft)
			cellBadge := tview.NewTableCell(badge).
				SetAlign(tview.AlignLeft)
			cellName := tview.NewTableCell(fmt.Sprintf(" %s", tview.Escape(st))).
				SetTextColor(tcell.ColorWhite).
				SetExpansion(1).
				SetAlign(tview.AlignLeft)

			filterTable.SetCell(i, 0, cellCheck)
			filterTable.SetCell(i, 1, cellBadge)
			filterTable.SetCell(i, 2, cellName)
		}
	}
	renderRows()

	filterTable.SetSelectedStyle(tcell.StyleDefault.
		Background(colorDarkNavy).
		Foreground(tcell.ColorWhite).
		Bold(true))

	closeModal := func() {
		c.pages.RemovePage("filterModal")
		c.app.SetFocus(c.table)
	}

	filterTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := filterTable.GetSelection()
		if row < 0 {
			row = 0
		}
		if row >= len(statuses) {
			row = len(statuses) - 1
		}

		switch event.Key() {
		case tcell.KeyEsc:
			closeModal()
			c.setStatus("Filter unchanged")
			return nil
		case tcell.KeyEnter:
			selectedCount := 0
			for _, st := range statuses {
				if tempSelection[st] {
					selectedCount++
				}
			}

			if selectedCount == len(statuses) || selectedCount == 0 {
				c.filterActive = false
				c.selectedStatuses = make(map[string]bool)
				c.applyFilter()
				closeModal()
				c.refreshIssuesTable()
				c.setStatus(fmt.Sprintf("Filter cleared (showing all %d issues)", len(c.issues)))
			} else {
				c.filterActive = true
				c.selectedStatuses = make(map[string]bool)
				for k, v := range tempSelection {
					c.selectedStatuses[k] = v
				}
				c.applyFilter()
				closeModal()
				c.refreshIssuesTable()
				c.setStatus(fmt.Sprintf("Filter applied: %d/%d issues matching", len(c.filteredIssues), len(c.issues)))
			}
			displayIssues := c.getDisplayIssues()
			if len(displayIssues) > 0 {
				c.table.Select(0, 0)
				c.onIssueSelectionChanged(0)
			}
			return nil
		}

		switch event.Rune() {
		case ' ':
			if row >= 0 && row < len(statuses) {
				st := statuses[row]
				tempSelection[st] = !tempSelection[st]
				renderRows()
				filterTable.Select(row, 0)
			}
			return nil
		case 'j':
			if row < len(statuses)-1 {
				filterTable.Select(row+1, 0)
			}
			return nil
		case 'k':
			if row > 0 {
				filterTable.Select(row-1, 0)
			}
			return nil
		case 'u':
			for _, st := range statuses {
				tempSelection[st] = IsUnresolvedStatus(st)
			}
			renderRows()
			filterTable.Select(row, 0)
			return nil
		case 'a':
			for _, st := range statuses {
				tempSelection[st] = true
			}
			renderRows()
			filterTable.Select(row, 0)
			return nil
		case 'c':
			for _, st := range statuses {
				tempSelection[st] = false
			}
			renderRows()
			filterTable.Select(row, 0)
			return nil
		}

		return event
	})

	helpView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[#7A869A][Space] Toggle  [u] Unresolved  [a] All  [c] Clear\n[Enter] Apply  [ESC] Cancel[-]")

	modalContent := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(filterTable, 0, 1, true).
		AddItem(helpView, 2, 0, false)

	modalContent.SetBorder(true).
		SetTitle(" 🔍 Status Filter (OR Selection) ").
		SetBorderColor(colorJiraBlue)

	modalHeight := len(statuses) + 6
	if modalHeight > 18 {
		modalHeight = 18
	}
	if modalHeight < 9 {
		modalHeight = 9
	}
	modalWidth := 56

	modalWrapper := centerModal(modalContent, modalWidth, modalHeight)
	c.pages.AddPage("filterModal", modalWrapper, true, true)
	c.app.SetFocus(filterTable)
}

func (c *Cockpit) triggerSync() {
	c.syncState = "SYNCING..."
	c.updateHeader()
	c.setStatus("Manual sync triggered...")

	go func() {
		if c.engine != nil {
			_, _ = c.engine.PollNow()
		}

		var fresh []pkg.Issue
		if c.cacheDB != nil {
			fresh, _ = c.cacheDB.GetIssues(c.space, "")
		} else if c.engine != nil {
			fresh = c.engine.GetMemoryIssues()
		}

		c.app.QueueUpdateDraw(func() {
			c.syncState = "LIVE WATCHING"
			c.lastSyncTime = time.Now()
			c.updateHeader()

			if len(fresh) > 0 {
				c.issues = fresh
				c.applyFilter()
				c.refreshIssuesTable()
				row, _ := c.table.GetSelection()
				c.onIssueSelectionChanged(row)
				c.setStatus(fmt.Sprintf("Synced with Jira (%d active issues)", len(fresh)))
			} else {
				c.setStatus("Sync completed: no issues found")
			}
		})
	}()
}

func (c *Cockpit) startEventListener(ctx context.Context) {
	if c.engine == nil {
		return
	}
	ch := c.engine.EventChannel()
	if ch == nil {
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				c.app.QueueUpdateDraw(func() {
					c.lastSyncTime = time.Now()
					c.syncState = "LIVE WATCHING"
					c.updateHeader()

					c.activities = append([]pkg.ActivityEvent{evt}, c.activities...)
					if len(c.activities) > 50 {
						c.activities = c.activities[:50]
					}
					c.refreshActivities()

					// Refresh issues list
					if c.cacheDB != nil {
						if fresh, err := c.cacheDB.GetIssues(c.space, ""); err == nil && len(fresh) > 0 {
							c.issues = fresh
							c.applyFilter()
							c.refreshIssuesTable()
							row, _ := c.table.GetSelection()
							c.onIssueSelectionChanged(row)
						}
					}
					c.setStatus(fmt.Sprintf("Live Event: %s (%s)", evt.IssueKey, evt.EventType))
				})
			}
		}
	}()
}

// RunTUI launches the rivo/tview Jira Cockpit dashboard.
func RunTUI(ctx context.Context, space, baseURL string, engine *pkg.WatchEngine, cache *pkg.CacheDB, client *pkg.Client) error {
	cockpit := NewCockpit(space, baseURL, engine, cache, client)
	cockpit.startEventListener(ctx)
	return cockpit.app.SetRoot(cockpit.pages, true).EnableMouse(true).Run()
}
