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
	"time"
)

// Config holds the Jira connection parameters.
type Config struct {
	InstanceURL string
	Email       string
	APIToken    string
	ProjectKey  string
}

// Client is a Jira API client.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// jiraConfigFile represents the structure of .jira.json or config.json.
type jiraConfigFile struct {
	InstanceURL string `json:"instance_url"`
	Email       string `json:"email"`
	APIToken    string `json:"api_token"`
	ProjectKey  string `json:"project_key"`
}

// LoadConfig reads configuration in priority:
// 1. OS environment variables
// 2. Local project config file (.jira.json, .jira/config.json, .agents/jira.json)
// 3. Global user config file (~/.config/jira/config.json, ~/.jira/config.json)
// 4. Local .env file (fallback)
func LoadConfig() (Config, error) {
	localCfg := loadLocalConfigFile()
	globalCfg := loadGlobalConfigFile()
	dotEnvMap := loadDotEnv()

	getVal := func(envKey, localVal, globalVal string) string {
		if val := os.Getenv(envKey); val != "" {
			return val
		}
		if localVal != "" {
			return localVal
		}
		if globalVal != "" {
			return globalVal
		}
		return dotEnvMap[envKey]
	}

	cfg := Config{
		InstanceURL: strings.TrimRight(getVal("JIRA_INSTANCE_URL", localCfg.InstanceURL, globalCfg.InstanceURL), "/"),
		Email:       getVal("JIRA_EMAIL", localCfg.Email, globalCfg.Email),
		APIToken:    getVal("JIRA_API_TOKEN", localCfg.APIToken, globalCfg.APIToken),
		ProjectKey:  getVal("JIRA_PROJECT_KEY", localCfg.ProjectKey, globalCfg.ProjectKey),
	}

	if cfg.ProjectKey == "" {
		cfg.ProjectKey = "KAN"
	}

	if cfg.InstanceURL == "" || cfg.Email == "" || cfg.APIToken == "" {
		return cfg, fmt.Errorf("missing Jira credentials. Please run 'jira configuration' to configure your credentials in .jira.json")
	}

	return cfg, nil
}

// loadLocalConfigFile traverses up from current working directory to find local Jira config files.
func loadLocalConfigFile() jiraConfigFile {
	cwd, err := os.Getwd()
	if err != nil {
		return jiraConfigFile{}
	}

	curr := cwd
	for {
		candidates := []string{
			filepath.Join(curr, ".jira.json"),
			filepath.Join(curr, ".jira", "config.json"),
			filepath.Join(curr, ".agents", "jira.json"),
		}

		for _, p := range candidates {
			if cfg, ok := readJiraConfigFile(p); ok {
				return cfg
			}
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return jiraConfigFile{}
}

// loadGlobalConfigFile loads configuration from ~/.config/jira/config.json or ~/.jira/config.json.
func loadGlobalConfigFile() jiraConfigFile {
	home, err := os.UserHomeDir()
	if err != nil {
		return jiraConfigFile{}
	}

	candidates := []string{
		filepath.Join(home, ".config", "jira", "config.json"),
		filepath.Join(home, ".jira", "config.json"),
	}

	for _, p := range candidates {
		if cfg, ok := readJiraConfigFile(p); ok {
			return cfg
		}
	}

	return jiraConfigFile{}
}

// readJiraConfigFile attempts to read and unmarshal a JSON file into jiraConfigFile.
func readJiraConfigFile(path string) (jiraConfigFile, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return jiraConfigFile{}, false
	}
	var cfg jiraConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return jiraConfigFile{}, false
	}
	return cfg, true
}

// NewClient initializes a Client using configuration loaded from environment/.env.
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

// loadDotEnv walks up the directory tree to find and parse .env files.
func loadDotEnv() map[string]string {
	cwd, err := os.Getwd()
	if err != nil {
		return map[string]string{}
	}

	curr := cwd
	for {
		envPath := filepath.Join(curr, ".env")
		if info, err := os.Stat(envPath); err == nil && !info.IsDir() {
			return parseEnvFileToMap(envPath)
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return map[string]string{}
}

// parseEnvFileToMap parses key-value pairs from .env file into a map.
func parseEnvFileToMap(filename string) map[string]string {
	res := make(map[string]string)
	file, err := os.Open(filename)
	if err != nil {
		return res
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if key != "" {
				res[key] = val
			}
		}
	}
	return res
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

