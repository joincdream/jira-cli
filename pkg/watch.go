package pkg

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// StatusChange represents a status transition detected during diffing.
type StatusChange struct {
	IssueKey  string `json:"issue_key"`
	Summary   string `json:"summary"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Author    string `json:"author"`
}

// CatchupDigest holds a summary of changes accumulated during a long absence (e.g. vacation).
type CatchupDigest struct {
	IsLongAbsence     bool           `json:"is_long_absence"`
	AbsenceDuration   time.Duration  `json:"absence_duration"`
	LastSyncAt        time.Time      `json:"last_sync_at"`
	NewlyAssigned     []Issue        `json:"newly_assigned"`
	StatusChanges     []StatusChange `json:"status_changes"`
	NewCommentCount   int            `json:"new_comment_count"`
	TotalChangedCount int            `json:"total_changed_count"`
}

// FormatSummary returns a human-readable box report of the catch-up digest.
func (d *CatchupDigest) FormatSummary() string {
	if d == nil || !d.IsLongAbsence || d.TotalChangedCount == 0 {
		return ""
	}

	days := int(d.AbsenceDuration.Hours() / 24)
	if days < 1 {
		days = 1
	}

	var sb strings.Builder
	sb.WriteString("┌─ [Welcome Back! 부재 중 업무 변경 브리핑] ────────────────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│ 🏖️ 마지막 동기화(%d일 전) 이후 총 %d건의 변경사항이 감지되었습니다:\n", days, d.TotalChangedCount))
	sb.WriteString("│\n")

	if len(d.NewlyAssigned) > 0 {
		sb.WriteString(fmt.Sprintf("│  ✨ 내게 새로 할당된 티켓 (%d건):\n", len(d.NewlyAssigned)))
		for _, iss := range d.NewlyAssigned {
			sb.WriteString(fmt.Sprintf("│     • [%s] %s\n", iss.Key, iss.Fields.Summary))
		}
		sb.WriteString("│\n")
	}

	if len(d.StatusChanges) > 0 {
		sb.WriteString(fmt.Sprintf("│  🔄 상태 변경된 티켓 (%d건):\n", len(d.StatusChanges)))
		for _, sc := range d.StatusChanges {
			sb.WriteString(fmt.Sprintf("│     • [%s] %s ➔ %s\n", sc.IssueKey, sc.OldStatus, sc.NewStatus))
		}
		sb.WriteString("│\n")
	}

	if d.NewCommentCount > 0 {
		sb.WriteString(fmt.Sprintf("│  💬 확인해야 할 신규 코멘트: %d건\n", d.NewCommentCount))
	}

	sb.WriteString("└─────────────────────────────────────────────────────────────────────┘\n")
	return sb.String()
}

// DiffIssues computes the delta between old in-memory issues and newly fetched issues.
func DiffIssues(space string, oldMap map[string]Issue, newIssues []Issue, currentUserEmail string) ([]ActivityEvent, []StatusChange, []Issue, int) {
	var events []ActivityEvent
	var statusChanges []StatusChange
	var newlyAssigned []Issue
	newCommentCount := 0
	now := time.Now()

	for _, newIss := range newIssues {
		oldIss, exists := oldMap[newIss.Key]
		if !exists {
			// Newly discovered issue
			// If an issue is already completed when first discovered, do not emit CREATED event
			if isDoneStatus(newIss.Fields.Status) {
				continue
			}

			evt := ActivityEvent{
				Space:     space,
				IssueKey:  newIss.Key,
				Author:    getAuthorName(newIss),
				EventType: "CREATED",
				Details:   fmt.Sprintf("[%s] 생성됨: %s", newIss.Fields.IssueType.Name, newIss.Fields.Summary),
				CreatedAt: now,
			}
			events = append(events, evt)

			if isAssignedToUser(newIss, currentUserEmail) {
				newlyAssigned = append(newlyAssigned, newIss)
			}
			continue
		}

		// Idempotency: if updated timestamp is identical, state is completely unchanged (No-Op)
		if oldIss.Fields.Updated != "" && newIss.Fields.Updated != "" && oldIss.Fields.Updated == newIss.Fields.Updated {
			continue
		}

		// 1. Status change
		if oldIss.Fields.Status.Name != newIss.Fields.Status.Name {
			sc := StatusChange{
				IssueKey:  newIss.Key,
				Summary:   newIss.Fields.Summary,
				OldStatus: oldIss.Fields.Status.Name,
				NewStatus: newIss.Fields.Status.Name,
				Author:    getAuthorName(newIss),
			}
			statusChanges = append(statusChanges, sc)

			evt := ActivityEvent{
				Space:     space,
				IssueKey:  newIss.Key,
				Author:    getAuthorName(newIss),
				EventType: "STATUS_CHANGE",
				Details:   fmt.Sprintf("상태 전이: %s ➔ %s", oldIss.Fields.Status.Name, newIss.Fields.Status.Name),
				CreatedAt: now,
			}
			events = append(events, evt)
		}

		// 2. Comments added
		oldComments := 0
		if oldIss.Fields.Comment != nil {
			oldComments = len(oldIss.Fields.Comment.Comments)
		}
		newComments := 0
		if newIss.Fields.Comment != nil {
			newComments = len(newIss.Fields.Comment.Comments)
		}
		if newComments > oldComments {
			diff := newComments - oldComments
			newCommentCount += diff
			latestCommentAuthor := ""
			if newComments > 0 {
				latestCommentAuthor = newIss.Fields.Comment.Comments[newComments-1].Author.DisplayName
			}

			evt := ActivityEvent{
				Space:     space,
				IssueKey:  newIss.Key,
				Author:    latestCommentAuthor,
				EventType: "COMMENT_ADDED",
				Details:   fmt.Sprintf("신규 코멘트 %d건 등록됨", diff),
				CreatedAt: now,
			}
			events = append(events, evt)
		}

		// 3. Newly assigned to me
		if !isAssignedToUser(oldIss, currentUserEmail) && isAssignedToUser(newIss, currentUserEmail) {
			newlyAssigned = append(newlyAssigned, newIss)
		}
	}

	return events, statusChanges, newlyAssigned, newCommentCount
}

func getAuthorName(issue Issue) string {
	if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
		return issue.Fields.Assignee.DisplayName
	}
	if issue.Fields.Reporter != nil && issue.Fields.Reporter.DisplayName != "" {
		return issue.Fields.Reporter.DisplayName
	}
	return "Jira"
}

func isAssignedToUser(issue Issue, email string) bool {
	if email == "" || issue.Fields.Assignee == nil {
		return false
	}
	return strings.EqualFold(issue.Fields.Assignee.EmailAddress, email)
}

// WatchEngine handles incremental background polling, in-memory diffing, and adaptive backoff.
type WatchEngine struct {
	client      *Client
	cacheDB     *CacheDB
	space       string
	minInterval time.Duration
	maxInterval time.Duration
	curInterval time.Duration

	issueStore   map[string]Issue // Key: IssueKey -> Value: Issue (Single Source of Truth)
	cachedDigest *CatchupDigest
	mu           sync.RWMutex

	eventCh chan ActivityEvent
	stopCh  chan struct{}
}

// NewWatchEngine creates a WatchEngine for the target project space.
func NewWatchEngine(client *Client, cacheDB *CacheDB, space string, initialInterval time.Duration) *WatchEngine {
	if initialInterval <= 0 {
		initialInterval = 15 * time.Second
	}
	return &WatchEngine{
		client:          client,
		cacheDB:         cacheDB,
		space:           strings.ToUpper(strings.TrimSpace(space)),
		minInterval:     initialInterval,
		maxInterval:     60 * time.Second,
		curInterval:     initialInterval,
		issueStore:      make(map[string]Issue),
		eventCh:         make(chan ActivityEvent, 100),
		stopCh:          make(chan struct{}),
	}
}

// EventChannel returns the read-only channel for live activity events.
func (w *WatchEngine) EventChannel() <-chan ActivityEvent {
	return w.eventCh
}

// GetMemoryIssues returns active (non-done) issues for TUI display (View Layer Projection).
// It projects non-completed issues from the central issueStore without mutating state.
func (w *WatchEngine) GetMemoryIssues() []Issue {
	w.mu.RLock()
	defer w.mu.RUnlock()

	res := make([]Issue, 0, len(w.issueStore))
	for _, iss := range w.issueStore {
		if !isDoneStatus(iss.Fields.Status) {
			res = append(res, iss)
		}
	}
	return res
}

// GetCatchupDigest returns any long-absence digest calculated during InitState.
func (w *WatchEngine) GetCatchupDigest() *CatchupDigest {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cachedDigest
}

// PollNow triggers an immediate sync cycle synchronously.
func (w *WatchEngine) PollNow() ([]ActivityEvent, error) {
	return w.PollOnce(context.Background())
}

// InitState loads the cached issues into memory (0ms startup) and checks for catch-up needs.
func (w *WatchEngine) InitState(ctx context.Context) (*CatchupDigest, error) {
	// 1. Load from local SQLite cache instantly into issueStore
	cachedIssues, err := w.cacheDB.GetIssues(w.space, "all")
	if err == nil && len(cachedIssues) > 0 {
		w.mu.Lock()
		for _, iss := range cachedIssues {
			w.issueStore[iss.Key] = iss
		}
		w.mu.Unlock()
	}

	// 2. Check last sync time for long absence detection (default threshold: 7 days)
	spaceInfo, _ := w.cacheDB.GetSpace(w.space)
	var lastSync time.Time
	isLongAbsence := false
	var absenceDuration time.Duration

	if spaceInfo != nil && !spaceInfo.LastSyncAt.IsZero() {
		lastSync = spaceInfo.LastSyncAt
		absenceDuration = time.Since(lastSync)
		if absenceDuration > 7*24*time.Hour {
			isLongAbsence = true
		}
	}

	// 3. Execute first remote sync
	events, statusChanges, newlyAssigned, newCommentCount, err := w.syncRemote(ctx)
	if err != nil {
		// Network might be offline; return cached state without error
		return nil, nil
	}

	var digest *CatchupDigest
	if isLongAbsence && (len(events) > 0 || len(statusChanges) > 0) {
		digest = &CatchupDigest{
			IsLongAbsence:     true,
			AbsenceDuration:   absenceDuration,
			LastSyncAt:        lastSync,
			NewlyAssigned:     newlyAssigned,
			StatusChanges:     statusChanges,
			NewCommentCount:   newCommentCount,
			TotalChangedCount: len(events),
		}
	}

	w.mu.Lock()
	w.cachedDigest = digest
	w.mu.Unlock()

	return digest, nil
}

// PollOnce executes a single sync cycle, diffing and saving changes.
func (w *WatchEngine) PollOnce(ctx context.Context) ([]ActivityEvent, error) {
	events, _, _, _, err := w.syncRemote(ctx)
	return events, err
}

func isDoneStatus(s Status) bool {
	if s.StatusCategory != nil && strings.EqualFold(s.StatusCategory.Key, "done") {
		return true
	}
	norm := strings.ToLower(strings.TrimSpace(s.Name))
	return strings.Contains(norm, "완료") || strings.Contains(norm, "done") || strings.Contains(norm, "resolved")
}

func (w *WatchEngine) syncRemote(ctx context.Context) ([]ActivityEvent, []StatusChange, []Issue, int, error) {
	// Query non-done issues OR issues updated within the last 1 day so that transitions to Done are detected
	jql := fmt.Sprintf("project = '%s' AND (statusCategory != Done OR updated >= -1d) ORDER BY updated DESC", w.space)
	fetched, err := w.client.ListIssuesWithOptions(ctx, jql, ListIssuesOptions{Detail: true})
	if err != nil {
		return nil, nil, nil, 0, err
	}

	w.mu.RLock()
	oldMap := make(map[string]Issue, len(w.issueStore))
	for k, v := range w.issueStore {
		oldMap[k] = v
	}
	w.mu.RUnlock()

	userEmail := w.client.GetConfig().Email
	events, statusChanges, newlyAssigned, newComments := DiffIssues(w.space, oldMap, fetched, userEmail)

	// 1. Upsert all fetched issues into SQLite cache (persisting state, never deleting completed ones)
	if err := w.cacheDB.UpsertIssues(w.space, fetched); err != nil {
		return nil, nil, nil, 0, err
	}

	// 2. Log activities to SQLite and publish to channel
	for _, evt := range events {
		_ = w.cacheDB.LogActivity(evt.Space, evt.IssueKey, evt.Author, evt.EventType, evt.Details, evt.CreatedAt)
		select {
		case w.eventCh <- evt:
		default:
		}
	}

	_ = w.cacheDB.SaveSpace(w.space, w.space, time.Now())

	// 3. Upsert into in-memory Single Source of Truth (Key: IssueKey -> Value: Issue)
	w.mu.Lock()
	for _, iss := range fetched {
		w.issueStore[iss.Key] = iss
	}
	w.mu.Unlock()

	// Adaptive backoff adjustment
	if len(events) > 0 {
		w.curInterval = w.minInterval // Reset backoff on activity
	} else {
		// Gradually back off: 15s -> 30s -> 60s
		if w.curInterval < w.maxInterval {
			w.curInterval = w.curInterval + 15*time.Second
			if w.curInterval > w.maxInterval {
				w.curInterval = w.maxInterval
			}
		}
	}

	return events, statusChanges, newlyAssigned, newComments, nil
}

// Start begins background polling loop.
func (w *WatchEngine) Start(ctx context.Context) {
	go func() {
		for {
			w.mu.RLock()
			interval := w.curInterval
			w.mu.RUnlock()

			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-time.After(interval):
				_, _ = w.PollOnce(ctx)
			}
		}
	}()
}

// Stop terminates the background watch loop.
func (w *WatchEngine) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
}
