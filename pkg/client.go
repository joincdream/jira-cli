package pkg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config holds the Jira connection parameters.
type Config struct {
	InstanceURL string
	Email       string
	APIToken    string
	ProjectKey  string
	Language    string
}

// Client is a Jira API client.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

var (
	activeProfileLock sync.RWMutex
	activeProfile     string
)

// SetActiveProfile sets the active profile name for configuration loading.
func SetActiveProfile(name string) {
	activeProfileLock.Lock()
	defer activeProfileLock.Unlock()
	activeProfile = strings.TrimSpace(name)
}

// FindLocalProfile searches for a local profile configuration file (.jira-profile, .jira/profile)
// starting from the current working directory upwards.
func FindLocalProfile() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	curr := cwd
	for {
		candidates := []string{
			filepath.Join(curr, ".jira-profile"),
			filepath.Join(curr, ".jira", "profile"),
		}

		for _, p := range candidates {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				data, err := os.ReadFile(p)
				if err == nil {
					line := strings.TrimSpace(string(data))
					for _, l := range strings.Split(line, "\n") {
						trimmed := strings.TrimSpace(l)
						if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
							return trimmed
						}
					}
				}
			}
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return ""
}

// GetActiveProfile returns the active profile name.
// Resolution order:
// 1. Explicitly set active profile (via CLI flag --profile)
// 2. JIRA_PROFILE environment variable
// 3. Local project profile file (.jira-profile or .jira/profile)
// 4. Default "default"
func GetActiveProfile() string {
	activeProfileLock.RLock()
	cur := activeProfile
	activeProfileLock.RUnlock()

	if cur != "" {
		return cur
	}
	if env := os.Getenv("JIRA_PROFILE"); env != "" {
		return strings.TrimSpace(env)
	}
	if local := FindLocalProfile(); local != "" {
		return local
	}
	return "default"
}

// GetDefaultConfigPath returns the canonical path ~/.config/jira/config.
func GetDefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("홈 디렉토리를 찾을 수 없습니다: %w", err)
	}
	return filepath.Join(home, ".config", "jira", "config"), nil
}

// ReadProfileConfig reads a specific profile's configuration from an INI file.
// Returns (Config, found, error).
func ReadProfileConfig(configPath, profile string) (Config, bool, error) {
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, false, nil
		}
		return Config{}, false, err
	}
	defer file.Close()

	targetHeader := strings.ToLower(strings.TrimSpace(profile))
	currentSection := ""
	var cfg Config
	found := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if currentSection == targetHeader {
				found = true
			}
			continue
		}

		if currentSection == targetHeader {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				switch key {
				case "instance_url":
					cfg.InstanceURL = val
				case "email":
					cfg.Email = val
				case "api_token":
					cfg.APIToken = val
				case "project_key":
					cfg.ProjectKey = val
				case "language", "lang":
					cfg.Language = val
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Config{}, false, err
	}

	return cfg, found, nil
}

// ListProfiles returns all profile names defined in the INI file.
func ListProfiles(configPath string) ([]string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var profiles []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name != "" && !seen[strings.ToLower(name)] {
				profiles = append(profiles, name)
				seen[strings.ToLower(name)] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

// WriteProfileConfig creates or updates a profile section in an INI file.
// Other sections, comments, and structure are preserved.
func WriteProfileConfig(configPath, profile string, cfg Config) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패 (%s): %w", dir, err)
	}

	newSection := formatProfileSection(profile, cfg)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(configPath, []byte(newSection+"\n"), 0600)
		}
		return err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	targetHeader := "[" + strings.ToLower(strings.TrimSpace(profile)) + "]"
	startIdx := -1
	endIdx := -1

	for i, line := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if trimmed == targetHeader {
				startIdx = i
			} else if startIdx != -1 {
				endIdx = i
				break
			}
		}
	}

	var resultLines []string
	if startIdx != -1 {
		if endIdx == -1 {
			endIdx = len(lines)
		}
		resultLines = append(resultLines, lines[:startIdx]...)
		resultLines = append(resultLines, newSection)
		resultLines = append(resultLines, lines[endIdx:]...)
	} else {
		trimmedContent := strings.TrimRight(content, "\r\n")
		if trimmedContent != "" {
			resultLines = append(resultLines, trimmedContent, "", newSection)
		} else {
			resultLines = append(resultLines, newSection)
		}
	}

	output := strings.Join(resultLines, "\n")
	output = strings.TrimRight(output, "\r\n") + "\n"
	return os.WriteFile(configPath, []byte(output), 0600)
}

func formatProfileSection(profile string, cfg Config) string {
	projectKey := cfg.ProjectKey
	if projectKey == "" {
		projectKey = "KAN"
	}
	res := fmt.Sprintf("[%s]\ninstance_url = %s\nemail = %s\napi_token = %s\nproject_key = %s",
		strings.TrimSpace(profile),
		strings.TrimRight(cfg.InstanceURL, "/"),
		cfg.Email,
		cfg.APIToken,
		projectKey,
	)
	if cfg.Language != "" {
		res += fmt.Sprintf("\nlanguage = %s", cfg.Language)
	}
	return res
}

// LoadConfig reads configuration for the active profile from ~/.config/jira/config.
func LoadConfig() (Config, error) {
	return LoadConfigForProfile(GetActiveProfile())
}

// LoadConfigForProfile loads configuration for a specified profile.
func LoadConfigForProfile(profile string) (Config, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return Config{}, err
	}
	return LoadConfigFileForProfile(configPath, profile)
}

// LoadConfigFileForProfile loads configuration for a specified profile from a specific file path.
func LoadConfigFileForProfile(configPath, profile string) (Config, error) {
	cfg, found, err := ReadProfileConfig(configPath, profile)
	if err != nil {
		return Config{}, fmt.Errorf("설정 파일 읽기 실패 (%s): %w", configPath, err)
	}

	// Environment variable overrides
	if envVal := os.Getenv("JIRA_INSTANCE_URL"); envVal != "" {
		cfg.InstanceURL = envVal
	}
	if envVal := os.Getenv("JIRA_EMAIL"); envVal != "" {
		cfg.Email = envVal
	}
	if envVal := os.Getenv("JIRA_API_TOKEN"); envVal != "" {
		cfg.APIToken = envVal
	}
	if envVal := os.Getenv("JIRA_PROJECT_KEY"); envVal != "" {
		cfg.ProjectKey = envVal
	}

	cfg.InstanceURL = strings.TrimRight(cfg.InstanceURL, "/")
	if cfg.ProjectKey == "" {
		cfg.ProjectKey = "KAN"
	}

	hasEnvCredentials := os.Getenv("JIRA_INSTANCE_URL") != "" && os.Getenv("JIRA_EMAIL") != "" && os.Getenv("JIRA_API_TOKEN") != ""

	if !found && !hasEnvCredentials {
		return cfg, fmt.Errorf("Jira 프로필 [%s]을(를) 찾을 수 없습니다 (%s).\n'jira configure --profile %s' 명령어로 프로필을 설정하세요.", profile, configPath, profile)
	}

	if cfg.InstanceURL == "" || cfg.Email == "" || cfg.APIToken == "" {
		return cfg, fmt.Errorf("Jira 인증 정보가 누락되었습니다 (프로필: [%s], 파일: %s).\n'jira configure --profile %s' 명령어로 설정하세요.", profile, configPath, profile)
	}

	return cfg, nil
}

// NewClient initializes a Client using configuration loaded from ~/.config/jira/config.
func NewClient() (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return NewClientWithConfig(cfg), nil
}

// NewClientWithConfig initializes a Client with explicit Config.
func NewClientWithConfig(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// GetConfig returns the client's configuration.
func (c *Client) GetConfig() Config {
	return c.cfg
}

// formatJiraAPIError parses and formats Jira REST API errors.
func formatJiraAPIError(statusCode int, body []byte) error {
	var jiraErr JiraErrorResponse
	if err := json.Unmarshal(body, &jiraErr); err == nil {
		var messages []string
		if len(jiraErr.ErrorMessages) > 0 {
			messages = append(messages, strings.Join(jiraErr.ErrorMessages, "; "))
		}
		for field, msg := range jiraErr.Errors {
			messages = append(messages, fmt.Sprintf("%s: %s", field, msg))
		}
		if len(messages) > 0 {
			return fmt.Errorf("Jira API error (status %d): %s", statusCode, strings.Join(messages, " | "))
		}
	}
	return fmt.Errorf("Jira API error (status %d): %s", statusCode, string(body))
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	fullURL := fmt.Sprintf("%s%s", c.cfg.InstanceURL, path)

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.cfg.Email, c.cfg.APIToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return respBytes, resp.StatusCode, formatJiraAPIError(resp.StatusCode, respBytes)
	}

	return respBytes, resp.StatusCode, nil
}

// ListIssues searches for issues matching JQL query.
func (c *Client) ListIssues(ctx context.Context, jql string) ([]Issue, error) {
	if jql == "" {
		jql = fmt.Sprintf("project = %s ORDER BY created DESC", c.cfg.ProjectKey)
	}

	fields := "summary,status,issuetype,priority,labels,assignee,duedate,created,updated,project,parent,subtasks"
	endpoint := fmt.Sprintf("/rest/api/3/search/jql?jql=%s&fields=%s", url.QueryEscape(jql), url.QueryEscape(fields))

	respBytes, _, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(respBytes, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search response: %w", err)
	}

	return searchResp.Issues, nil
}

// GetIssue retrieves a single issue by key or id.
func (c *Client) GetIssue(ctx context.Context, key string) (*Issue, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key))
	respBytes, _, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(respBytes, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue: %w", err)
	}

	return &issue, nil
}

// CreateIssue creates a new issue in Jira.
func (c *Client) CreateIssue(ctx context.Context, projectKey, summary, description, issueTypeName, dueDate, parentKey string, labels []string) (*CreateIssueResponse, error) {
	if projectKey == "" {
		projectKey = c.cfg.ProjectKey
	}
	if issueTypeName == "" {
		issueTypeName = "작업"
	}

	req := CreateIssueRequest{
		Fields: CreateIssueFields{
			Project: ProjectRef{
				Key: projectKey,
			},
			Summary: summary,
			IssueType: TypeRef{
				Name: issueTypeName,
			},
			Labels: labels,
		},
	}

	if description != "" {
		req.Fields.Description = BuildADFDocument(description)
	}
	if dueDate != "" {
		req.Fields.DueDate = dueDate
	}
	if parentKey != "" {
		req.Fields.Parent = &ParentRef{
			Key: parentKey,
		}
	}

	respBytes, _, err := c.doRequest(ctx, "POST", "/rest/api/3/issue", req)
	if err != nil {
		return nil, err
	}

	var createResp CreateIssueResponse
	if err := json.Unmarshal(respBytes, &createResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create response: %w", err)
	}

	return &createResp, nil
}

// GetTransitions fetches possible workflow transitions for an issue.
func (c *Client) GetTransitions(ctx context.Context, key string) ([]Transition, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(key))
	respBytes, _, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var transResp TransitionsResponse
	if err := json.Unmarshal(respBytes, &transResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transitions: %w", err)
	}

	return transResp.Transitions, nil
}

// TransitionIssue changes the status of an issue.
func (c *Client) TransitionIssue(ctx context.Context, key, targetStatusNameOrID string) error {
	transitions, err := c.GetTransitions(ctx, key)
	if err != nil {
		return err
	}

	var matchedID string
	var available []string

	targetLower := strings.ToLower(strings.TrimSpace(targetStatusNameOrID))
	for _, t := range transitions {
		available = append(available, fmt.Sprintf("%s (id:%s, to:%s)", t.Name, t.ID, t.To.Name))
		if t.ID == targetStatusNameOrID ||
			strings.ToLower(t.Name) == targetLower ||
			strings.ToLower(t.To.Name) == targetLower {
			matchedID = t.ID
			break
		}
	}

	if matchedID == "" {
		return fmt.Errorf("no matching transition for '%s'. Available transitions: %s", targetStatusNameOrID, strings.Join(available, ", "))
	}

	payload := map[string]interface{}{
		"transition": map[string]string{
			"id": matchedID,
		},
	}

	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(key))
	_, _, err = c.doRequest(ctx, "POST", endpoint, payload)
	return err
}

// AddComment posts a comment on an issue.
func (c *Client) AddComment(ctx context.Context, key, text string) error {
	payload := map[string]interface{}{
		"body": BuildADFDocument(text),
	}

	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment", url.PathEscape(key))
	_, _, err := c.doRequest(ctx, "POST", endpoint, payload)
	return err
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(ctx context.Context, key string, deleteSubtasks bool) error {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s?deleteSubtasks=%t", url.PathEscape(key), deleteSubtasks)
	_, _, err := c.doRequest(ctx, "DELETE", endpoint, nil)
	return err
}

// UpdateIssueOptions holds options for updating an issue.
type UpdateIssueOptions struct {
	Summary      *string
	Description  *string
	DueDate      *string
	ParentKey    *string
	Labels       []string
	UpdateLabels bool
}

// UpdateIssue updates fields of an existing issue.
func (c *Client) UpdateIssue(ctx context.Context, key string, opts UpdateIssueOptions) error {
	fields := make(map[string]interface{})

	if opts.Summary != nil {
		fields["summary"] = *opts.Summary
	}
	if opts.Description != nil {
		fields["description"] = BuildADFDocument(*opts.Description)
	}
	if opts.DueDate != nil {
		if *opts.DueDate == "" || *opts.DueDate == "none" || *opts.DueDate == "null" {
			fields["duedate"] = nil
		} else {
			fields["duedate"] = *opts.DueDate
		}
	}
	if opts.ParentKey != nil {
		if *opts.ParentKey == "" || *opts.ParentKey == "none" || *opts.ParentKey == "null" {
			fields["parent"] = nil
		} else {
			fields["parent"] = map[string]string{
				"key": *opts.ParentKey,
			}
		}
	}
	if opts.UpdateLabels {
		fields["labels"] = opts.Labels
	}

	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}

	payload := map[string]interface{}{
		"fields": fields,
	}

	endpoint := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key))
	_, _, err := c.doRequest(ctx, "PUT", endpoint, payload)
	return err
}

// UpdateIssueLabels updates the labels of an existing issue.
func (c *Client) UpdateIssueLabels(ctx context.Context, key string, labels []string) error {
	return c.UpdateIssue(ctx, key, UpdateIssueOptions{
		Labels:       labels,
		UpdateLabels: true,
	})
}

