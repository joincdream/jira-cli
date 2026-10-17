package pkg

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ActivityEvent represents a recorded activity in the live activity stream.
type ActivityEvent struct {
	ID        int64     `json:"id"`
	Space     string    `json:"space"`
	IssueKey  string    `json:"issue_key"`
	Author    string    `json:"author"`
	EventType string    `json:"event_type"` // STATUS_CHANGE, COMMENT_ADDED, CREATED, UPDATED
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

// SpaceInfo represents metadata about a Jira project/space stored in cache.
type SpaceInfo struct {
	Key        string    `json:"key"`
	Name       string    `json:"name"`
	LastSyncAt time.Time `json:"last_sync_at"`
}

// CacheDB wraps a SQLite connection for local state caching.
type CacheDB struct {
	db *sql.DB
}

// GetDefaultCachePath returns the standard cache DB path ~/.cache/jira/state.db.
func GetDefaultCachePath() (string, error) {
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheHome = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheHome, "jira", "state.db"), nil
}

// OpenCacheDB opens or creates the SQLite cache database with WAL mode and schemas.
func OpenCacheDB(dbPath string) (*CacheDB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Limit idle connections for embedded use
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	c := &CacheDB{db: db}
	if err := c.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return c, nil
}

func (c *CacheDB) initSchema() error {
	ddl := `
	CREATE TABLE IF NOT EXISTS spaces (
		key TEXT PRIMARY KEY,
		name TEXT,
		last_sync_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS issues (
		key TEXT PRIMARY KEY,
		space TEXT NOT NULL,
		summary TEXT NOT NULL,
		status TEXT NOT NULL,
		assignee TEXT,
		issue_type TEXT,
		due_date TEXT,
		raw_json TEXT NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_issues_space ON issues(space);
	CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
	CREATE INDEX IF NOT EXISTS idx_issues_updated ON issues(updated_at DESC);

	CREATE TABLE IF NOT EXISTS activities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		space TEXT NOT NULL,
		issue_key TEXT NOT NULL,
		author TEXT,
		event_type TEXT NOT NULL,
		details TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_activities_created ON activities(created_at DESC);

	CREATE TABLE IF NOT EXISTS active_context (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err := c.db.Exec(ddl)
	return err
}

// Close closes the underlying SQLite database connection.
func (c *CacheDB) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// SaveSpace updates metadata for a space.
func (c *CacheDB) SaveSpace(key, name string, lastSyncAt time.Time) error {
	query := `
	INSERT INTO spaces (key, name, last_sync_at)
	VALUES (?, ?, ?)
	ON CONFLICT(key) DO UPDATE SET
		name = excluded.name,
		last_sync_at = excluded.last_sync_at;
	`
	_, err := c.db.Exec(query, strings.ToUpper(key), name, lastSyncAt)
	return err
}

// GetSpace retrieves space metadata.
func (c *CacheDB) GetSpace(key string) (*SpaceInfo, error) {
	row := c.db.QueryRow("SELECT key, name, last_sync_at FROM spaces WHERE key = ?", strings.ToUpper(key))
	var s SpaceInfo
	var lastSync sql.NullTime
	if err := row.Scan(&s.Key, &s.Name, &lastSync); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastSync.Valid {
		s.LastSyncAt = lastSync.Time
	}
	return &s, nil
}

// UpsertIssues batches multiple issues into the database inside a transaction.
func (c *CacheDB) UpsertIssues(space string, issues []Issue) error {
	if len(issues) == 0 {
		return nil
	}

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO issues (key, space, summary, status, assignee, issue_type, due_date, raw_json, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(key) DO UPDATE SET
		space = excluded.space,
		summary = excluded.summary,
		status = excluded.status,
		assignee = excluded.assignee,
		issue_type = excluded.issue_type,
		due_date = excluded.due_date,
		raw_json = excluded.raw_json,
		updated_at = excluded.updated_at;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	spaceUpper := strings.ToUpper(space)
	for _, issue := range issues {
		rawJSON, err := json.Marshal(issue)
		if err != nil {
			continue
		}

		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}

		updatedAt := time.Now()
		if issue.Fields.Updated != "" {
			if t, err := time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Updated); err == nil {
				updatedAt = t
			} else if t, err := time.Parse(time.RFC3339, issue.Fields.Updated); err == nil {
				updatedAt = t
			}
		}

		_, err = stmt.Exec(
			issue.Key,
			spaceUpper,
			issue.Fields.Summary,
			issue.Fields.Status.Name,
			assignee,
			issue.Fields.IssueType.Name,
			issue.Fields.DueDate,
			string(rawJSON),
			updatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetIssues returns cached issues for the given space (or all spaces if space is "").
func (c *CacheDB) GetIssues(space string, statusFilter string) ([]Issue, error) {
	var query strings.Builder
	query.WriteString("SELECT raw_json FROM issues WHERE 1=1")
	var args []interface{}

	if strings.TrimSpace(space) != "" {
		query.WriteString(" AND space = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(space)))
	}

	trimmedStatus := strings.ToLower(strings.TrimSpace(statusFilter))
	switch trimmedStatus {
	case "open":
		query.WriteString(" AND LOWER(status) NOT IN ('완료', 'done', 'closed', 'resolved')")
	case "done":
		query.WriteString(" AND LOWER(status) IN ('완료', 'done', 'closed', 'resolved')")
	case "", "all", "*":
		// no filter
	default:
		query.WriteString(" AND LOWER(status) = ?")
		args = append(args, trimmedStatus)
	}

	query.WriteString(" ORDER BY updated_at DESC, key ASC")

	rows, err := c.db.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Issue
	for rows.Next() {
		var rawJSON string
		if err := rows.Scan(&rawJSON); err != nil {
			continue
		}
		var issue Issue
		if err := json.Unmarshal([]byte(rawJSON), &issue); err == nil {
			result = append(result, issue)
		}
	}

	return result, nil
}

// GetIssue returns a single issue by key from cache.
func (c *CacheDB) GetIssue(key string) (*Issue, error) {
	row := c.db.QueryRow("SELECT raw_json FROM issues WHERE key = ?", strings.ToUpper(strings.TrimSpace(key)))
	var rawJSON string
	if err := row.Scan(&rawJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal([]byte(rawJSON), &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// PruneDeletedIssues removes cached issues for space that are not in currentKeys.
func (c *CacheDB) PruneDeletedIssues(space string, currentKeys []string) error {
	if len(currentKeys) == 0 {
		return nil
	}

	placeholders := make([]string, len(currentKeys))
	args := make([]interface{}, 0, len(currentKeys)+1)
	args = append(args, strings.ToUpper(strings.TrimSpace(space)))

	for i, k := range currentKeys {
		placeholders[i] = "?"
		args = append(args, strings.ToUpper(k))
	}

	query := fmt.Sprintf("DELETE FROM issues WHERE space = ? AND key NOT IN (%s)", strings.Join(placeholders, ","))
	_, err := c.db.Exec(query, args...)
	return err
}

// LogActivity records a new activity event and trims older activities past 500 records.
func (c *CacheDB) LogActivity(space, issueKey, author, eventType, details string, createdAt time.Time) error {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	query := `
	INSERT INTO activities (space, issue_key, author, event_type, details, created_at)
	VALUES (?, ?, ?, ?, ?, ?);
	`
	_, err := c.db.Exec(query, strings.ToUpper(space), strings.ToUpper(issueKey), author, eventType, details, createdAt)
	if err != nil {
		return err
	}

	// Keep rolling buffer of latest 500 activities
	_, _ = c.db.Exec(`DELETE FROM activities WHERE id NOT IN (SELECT id FROM activities ORDER BY id DESC LIMIT 500);`)
	return nil
}

// GetRecentActivities retrieves up to limit recent activities.
func (c *CacheDB) GetRecentActivities(limit int, spaceFilter string) ([]ActivityEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	var query strings.Builder
	query.WriteString("SELECT id, space, issue_key, author, event_type, details, created_at FROM activities WHERE 1=1")
	var args []interface{}

	if strings.TrimSpace(spaceFilter) != "" {
		query.WriteString(" AND space = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(spaceFilter)))
	}

	query.WriteString(" ORDER BY id DESC LIMIT ?")
	args = append(args, limit)

	rows, err := c.db.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ActivityEvent
	for rows.Next() {
		var e ActivityEvent
		if err := rows.Scan(&e.ID, &e.Space, &e.IssueKey, &e.Author, &e.EventType, &e.Details, &e.CreatedAt); err == nil {
			events = append(events, e)
		}
	}
	return events, nil
}

// SetActiveIssue records the currently active issue key in active_context.
func (c *CacheDB) SetActiveIssue(key string) error {
	trimmed := strings.ToUpper(strings.TrimSpace(key))
	query := `
	INSERT INTO active_context (key, value)
	VALUES ('active_issue', ?)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value;
	`
	_, err := c.db.Exec(query, trimmed)
	return err
}

// GetActiveIssue retrieves the active issue key from active_context.
func (c *CacheDB) GetActiveIssue() (string, error) {
	row := c.db.QueryRow("SELECT value FROM active_context WHERE key = 'active_issue'")
	var val string
	if err := row.Scan(&val); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

// ClearActiveIssue deletes the active issue setting from active_context.
func (c *CacheDB) ClearActiveIssue() error {
	_, err := c.db.Exec("DELETE FROM active_context WHERE key = 'active_issue'")
	return err
}

// GetCachedActiveIssue returns the active issue key from the default cache file if present.
func GetCachedActiveIssue() string {
	path, err := GetDefaultCachePath()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}
	cache, err := OpenCacheDB(path)
	if err != nil {
		return ""
	}
	defer cache.Close()
	key, _ := cache.GetActiveIssue()
	return key
}
