package pkg

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Issue represents a Jira issue.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains the metadata and content of an issue.
type IssueFields struct {
	Summary     string        `json:"summary"`
	Description interface{}   `json:"description,omitempty"`
	Status      Status        `json:"status"`
	IssueType   IssueType     `json:"issuetype"`
	Priority    *Priority     `json:"priority,omitempty"`
	Labels      []string      `json:"labels,omitempty"`
	Assignee    *User         `json:"assignee,omitempty"`
	Reporter    *User         `json:"reporter,omitempty"`
	DueDate     string        `json:"duedate,omitempty"`
	Created     string        `json:"created,omitempty"`
	Updated     string        `json:"updated,omitempty"`
	Project     Project       `json:"project"`
	Subtasks    []Issue       `json:"subtasks,omitempty"`
	Comment     *CommentBlock `json:"comment,omitempty"`
}

// Status represents the workflow status of an issue.
type Status struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	StatusCategory *StatusCategory `json:"statusCategory,omitempty"`
}

// StatusCategory classifies the status (To Do, In Progress, Done).
type StatusCategory struct {
	ID        int    `json:"id"`
	Key       string `json:"key"`
	ColorName string `json:"colorName"`
	Name      string `json:"name"`
}

// IssueType represents the type of issue (Task, Story, Bug, Epic, Subtask).
type IssueType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
}

// Priority represents priority level (High, Medium, Low).
type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// User represents an Atlassian user.
type User struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	Active       bool   `json:"active"`
}

// Project represents a Jira project.
type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// CommentBlock contains a list of comments.
type CommentBlock struct {
	Comments []Comment `json:"comments"`
	Total    int       `json:"total"`
}

// Comment represents a single comment on an issue.
type Comment struct {
	ID      string      `json:"id"`
	Author  User        `json:"author"`
	Body    interface{} `json:"body"`
	Created string      `json:"created"`
	Updated string      `json:"updated"`
}

// SearchResponse represents the response from /rest/api/3/search/jql.
type SearchResponse struct {
	Issues        []Issue `json:"issues"`
	IsLast        bool    `json:"isLast"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
}

// Transition represents an available workflow transition.
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
}

// TransitionsResponse represents the response from /transitions.
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// CreateIssueRequest is the payload to create an issue.
type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

// CreateIssueFields contains fields for issue creation.
type CreateIssueFields struct {
	Project     ProjectRef  `json:"project"`
	Summary     string      `json:"summary"`
	Description interface{} `json:"description,omitempty"`
	IssueType   TypeRef     `json:"issuetype"`
	Labels      []string    `json:"labels,omitempty"`
	DueDate     string      `json:"duedate,omitempty"`
	Parent      *ParentRef  `json:"parent,omitempty"`
}

// ProjectRef references a project by key or id.
type ProjectRef struct {
	Key string `json:"key,omitempty"`
	ID  string `json:"id,omitempty"`
}

// TypeRef references an issue type by name or id.
type TypeRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// ParentRef references a parent issue (for subtasks).
type ParentRef struct {
	Key string `json:"key,omitempty"`
	ID  string `json:"id,omitempty"`
}

// CreateIssueResponse is returned upon issue creation.
type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// JiraErrorResponse represents error responses from Jira REST API.
type JiraErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

// ADF (Atlassian Document Format) helpers

// BuildADFDocument constructs a rich ADF doc from markdown text.
func BuildADFDocument(text string) map[string]interface{} {
	if strings.TrimSpace(text) == "" {
		return map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []map[string]interface{}{
				{
					"type": "paragraph",
					"content": []map[string]interface{}{
						{"type": "text", "text": ""},
					},
				},
			},
		}
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var content []map[string]interface{}

	var inCodeBlock bool
	var codeLang string
	var codeLines []string

	var inBulletList bool
	var currentBulletItems [][]map[string]interface{}

	var inOrderedList bool
	var currentOrderedItems [][]map[string]interface{}

	flushLists := func() {
		if inBulletList && len(currentBulletItems) > 0 {
			var listItems []map[string]interface{}
			for _, itemNodes := range currentBulletItems {
				listItems = append(listItems, map[string]interface{}{
					"type": "listItem",
					"content": []map[string]interface{}{
						{
							"type":    "paragraph",
							"content": itemNodes,
						},
					},
				})
			}
			content = append(content, map[string]interface{}{
				"type":    "bulletList",
				"content": listItems,
			})
			inBulletList = false
			currentBulletItems = nil
		}

		if inOrderedList && len(currentOrderedItems) > 0 {
			var listItems []map[string]interface{}
			for _, itemNodes := range currentOrderedItems {
				listItems = append(listItems, map[string]interface{}{
					"type": "listItem",
					"content": []map[string]interface{}{
						{
							"type":    "paragraph",
							"content": itemNodes,
						},
					},
				})
			}
			content = append(content, map[string]interface{}{
				"type":    "orderedList",
				"content": listItems,
			})
			inOrderedList = false
			currentOrderedItems = nil
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 1. Code Block handling
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// End of code block
				codeText := strings.Join(codeLines, "\n")
				codeNode := map[string]interface{}{
					"type": "codeBlock",
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": codeText,
						},
					},
				}
				if codeLang != "" {
					codeNode["attrs"] = map[string]interface{}{
						"language": codeLang,
					}
				}
				content = append(content, codeNode)
				inCodeBlock = false
				codeLang = ""
				codeLines = nil
			} else {
				// Start of code block
				flushLists()
				inCodeBlock = true
				codeLang = strings.TrimPrefix(trimmed, "```")
				codeLines = nil
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// Empty line
		if trimmed == "" {
			flushLists()
			continue
		}

		// Horizontal rule
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			flushLists()
			content = append(content, map[string]interface{}{
				"type": "rule",
			})
			continue
		}

		// 2. Heading (#, ##, ###, ####, #####, ######)
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for level < len(trimmed) && trimmed[level] == '#' {
				level++
			}
			if level <= 6 && len(trimmed) > level && trimmed[level] == ' ' {
				flushLists()
				headingText := strings.TrimSpace(trimmed[level:])
				inlineNodes := parseInlineText(headingText)
				content = append(content, map[string]interface{}{
					"type": "heading",
					"attrs": map[string]interface{}{
						"level": level,
					},
					"content": inlineNodes,
				})
				continue
			}
		}

		// 3. Bullet List (- , * )
		if (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) && !isOrderedList(trimmed) {
			if inOrderedList {
				flushLists()
			}
			inBulletList = true
			itemText := strings.TrimSpace(trimmed[2:])
			currentBulletItems = append(currentBulletItems, parseInlineText(itemText))
			continue
		}

		// 4. Ordered List (1. , 2. )
		if isOrderedList(trimmed) {
			if inBulletList {
				flushLists()
			}
			inOrderedList = true
			dotIdx := strings.Index(trimmed, ".")
			itemText := strings.TrimSpace(trimmed[dotIdx+1:])
			currentOrderedItems = append(currentOrderedItems, parseInlineText(itemText))
			continue
		}

		// 5. Blockquote (> )
		if strings.HasPrefix(trimmed, "> ") {
			flushLists()
			quoteText := strings.TrimSpace(trimmed[2:])
			content = append(content, map[string]interface{}{
				"type": "blockquote",
				"content": []map[string]interface{}{
					{
						"type":    "paragraph",
						"content": parseInlineText(quoteText),
					},
				},
			})
			continue
		}

		// 6. Regular Paragraph
		flushLists()
		content = append(content, map[string]interface{}{
			"type":    "paragraph",
			"content": parseInlineText(trimmed),
		})
	}

	flushLists()

	if inCodeBlock && len(codeLines) > 0 {
		codeNode := map[string]interface{}{
			"type": "codeBlock",
			"content": []map[string]interface{}{
				{"type": "text", "text": strings.Join(codeLines, "\n")},
			},
		}
		if codeLang != "" {
			codeNode["attrs"] = map[string]interface{}{"language": codeLang}
		}
		content = append(content, codeNode)
	}

	if len(content) == 0 {
		content = append(content, map[string]interface{}{
			"type": "paragraph",
			"content": []map[string]interface{}{
				{"type": "text", "text": text},
			},
		})
	}

	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

func isOrderedList(s string) bool {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return false
	}
	numStr := strings.TrimSpace(parts[0])
	if numStr == "" {
		return false
	}
	for _, ch := range numStr {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return strings.HasPrefix(parts[1], " ")
}

// parseInlineText parses bold (**), code (`), and links into ADF text nodes with marks.
func parseInlineText(text string) []map[string]interface{} {
	if text == "" {
		return []map[string]interface{}{
			{"type": "text", "text": " "},
		}
	}

	var nodes []map[string]interface{}
	i := 0
	runes := []rune(text)
	n := len(runes)

	var current strings.Builder

	flushCurrent := func() {
		if current.Len() > 0 {
			nodes = append(nodes, map[string]interface{}{
				"type": "text",
				"text": current.String(),
			})
			current.Reset()
		}
	}

	for i < n {
		// Bold: **text**
		if i+1 < n && runes[i] == '*' && runes[i+1] == '*' {
			end := -1
			for j := i + 2; j+1 < n; j++ {
				if runes[j] == '*' && runes[j+1] == '*' {
					end = j
					break
				}
			}
			if end != -1 {
				flushCurrent()
				boldText := string(runes[i+2 : end])
				nodes = append(nodes, map[string]interface{}{
					"type": "text",
					"text": boldText,
					"marks": []map[string]interface{}{
						{"type": "strong"},
					},
				})
				i = end + 2
				continue
			}
		}

		// Inline Code: `text`
		if runes[i] == '`' {
			end := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == '`' {
					end = j
					break
				}
			}
			if end != -1 {
				flushCurrent()
				codeText := string(runes[i+1 : end])
				nodes = append(nodes, map[string]interface{}{
					"type": "text",
					"text": codeText,
					"marks": []map[string]interface{}{
						{"type": "code"},
					},
				})
				i = end + 1
				continue
			}
		}

		current.WriteRune(runes[i])
		i++
	}

	flushCurrent()

	if len(nodes) == 0 {
		nodes = append(nodes, map[string]interface{}{
			"type": "text",
			"text": text,
		})
	}

	return nodes
}

// ExtractTextFromADF parses ADF structure or string into plain text with terminal formatting.
func ExtractTextFromADF(body interface{}) string {
	if body == nil {
		return ""
	}
	if str, ok := body.(string); ok {
		return str
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return ""
	}

	var node map[string]interface{}
	if err := json.Unmarshal(raw, &node); err != nil {
		return fmt.Sprintf("%v", body)
	}

	var sb strings.Builder
	extractTextRecursive(node, &sb, 0)
	return strings.TrimSpace(sb.String())
}

func extractTextRecursive(node map[string]interface{}, sb *strings.Builder, depth int) {
	nodeType, _ := node["type"].(string)

	if nodeType == "text" {
		if text, ok := node["text"].(string); ok {
			sb.WriteString(text)
		}
		return
	}

	if nodeType == "rule" {
		sb.WriteString("--------------------------------------------------------------------------------\n")
		return
	}

	if nodeType == "listItem" {
		sb.WriteString("  • ")
	}

	if content, ok := node["content"].([]interface{}); ok {
		for _, child := range content {
			if childMap, ok := child.(map[string]interface{}); ok {
				extractTextRecursive(childMap, sb, depth+1)
			}
		}

		switch nodeType {
		case "paragraph", "heading", "codeBlock", "blockquote":
			sb.WriteString("\n")
		case "listItem":
			// Handled by child paragraph
		}
	}
}
