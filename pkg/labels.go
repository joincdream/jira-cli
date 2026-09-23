package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"tools/jira/internal/i18n"
)

// LabelItem represents a single label entry.
type LabelItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// LabelCategory represents a category of labels.
type LabelCategory struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Labels      []LabelItem `json:"labels"`
}

// LabelCatalog represents the full labels configuration.
type LabelCatalog struct {
	Version    string                   `json:"version"`
	Categories map[string]LabelCategory `json:"categories"`
}

// LoadLabelCatalog searches for .agents/labels.json or labels.json and parses it.
func LoadLabelCatalog() (*LabelCatalog, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return getDefaultCatalog(), nil
	}

	curr := cwd
	for {
		candidates := []string{
			filepath.Join(curr, ".agents", "labels.json"),
			filepath.Join(curr, "labels.json"),
		}

		for _, p := range candidates {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				data, err := os.ReadFile(p)
				if err == nil {
					var cat LabelCatalog
					if err := json.Unmarshal(data, &cat); err == nil && len(cat.Categories) > 0 {
						return &cat, nil
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

	return getDefaultCatalog(), nil
}

// GetValidLabelsSet returns a map of all allowed label names (case-sensitive and lowercase lookup).
func (c *LabelCatalog) GetValidLabelsSet() (map[string]string, []string) {
	validMap := make(map[string]string)
	var allList []string

	for _, cat := range c.Categories {
		for _, item := range cat.Labels {
			validMap[item.Name] = item.Name
			validMap[strings.ToLower(item.Name)] = item.Name
			allList = append(allList, item.Name)
		}
	}
	return validMap, allList
}

// ValidateAndNormalizeLabels validates the input labels against catalog and normalizes casing.
func (c *LabelCatalog) ValidateAndNormalizeLabels(inputLabels []string) ([]string, []string, error) {
	if len(inputLabels) == 0 {
		return nil, nil, nil
	}

	validMap, _ := c.GetValidLabelsSet()
	var normalized []string
	var invalid []string

	for _, l := range inputLabels {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}

		if canonical, ok := validMap[trimmed]; ok {
			normalized = append(normalized, canonical)
		} else if canonical, ok := validMap[strings.ToLower(trimmed)]; ok {
			normalized = append(normalized, canonical)
		} else {
			invalid = append(invalid, trimmed)
		}
	}

	if len(invalid) > 0 {
		return normalized, invalid, fmt.Errorf("%s", i18n.Sprintf("pkg.labels.err_invalid_label", strings.Join(invalid, ", ")))
	}

	return normalized, nil, nil
}

// PrintLabelsCatalog prints the catalog in a clean terminal format.
func PrintLabelsCatalog(w io.Writer, catalog *LabelCatalog) {
	fmt.Fprintln(w, i18n.T("pkg.labels.title"))
	fmt.Fprintln(w, "================================================================================")

	order := []string{"project", "tech", "activity"}
	// print ordered known categories first, then any extra
	seen := make(map[string]bool)

	for _, key := range order {
		if cat, ok := catalog.Categories[key]; ok {
			printCategory(w, key, cat)
			seen[key] = true
		}
	}

	for key, cat := range catalog.Categories {
		if !seen[key] {
			printCategory(w, key, cat)
		}
	}
	fmt.Fprintln(w, "================================================================================")
	fmt.Fprintln(w, i18n.T("pkg.labels.hint"))
}

func printCategory(w io.Writer, key string, cat LabelCategory) {
	fmt.Fprintf(w, "\n📁 [%s] %s\n", strings.ToUpper(key), cat.Name)
	if cat.Description != "" {
		fmt.Fprintf(w, i18n.Sprintf("pkg.labels.desc_prefix", cat.Description))
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	for _, item := range cat.Labels {
		fmt.Fprintf(tw, "   • %-18s\t# %s\n", item.Name, item.Description)
	}
	tw.Flush()
}

func getDefaultCatalog() *LabelCatalog {
	return &LabelCatalog{
		Version: "1.0.0",
		Categories: map[string]LabelCategory{
			"project": {
				Name:        "프로젝트 및 고객사",
				Description: "연관된 프로젝트 또는 고객사 구분",
				Labels: []LabelItem{
					{Name: "DIVE", Description: "DIVE MCP AI Agent 플랫폼 구축 사업"},
					{Name: "AgentGo_Studio", Description: "AgentGo Studio 서비스 개발"},
					{Name: "Fursys", Description: "퍼시스 관련 프로젝트 및 과제"},
					{Name: "아트리안", Description: "아트리안 고객사 연계 업무"},
					{Name: "FDE", Description: "FDE(Field Deploy Engineer) 조직 및 현장 프로젝트"},
				},
			},
			"tech": {
				Name:        "기술 및 도메인",
				Description: "관련 기술 스택 및 도메인",
				Labels: []LabelItem{
					{Name: "AI", Description: "인공지능, VLM, LLM, Agent 기술"},
					{Name: "MCP", Description: "Model Context Protocol 서버 및 연동"},
					{Name: "Cloud", Description: "AWS, GCP, 클라우드 인프라"},
					{Name: "Architecture", Description: "시스템 설계 및 데이터 아키텍처"},
				},
			},
			"activity": {
				Name:        "업무 성격 및 산출물",
				Description: "작업의 활동 성격 및 결과물 형태",
				Labels: []LabelItem{
					{Name: "Planning", Description: "기획, 일정 수립, MM 산정, R&R 정의"},
					{Name: "Development", Description: "기능 구현 및 코딩"},
					{Name: "PoC", Description: "개념 검증 및 프로토타이핑"},
					{Name: "QA", Description: "테스트 및 품질 검증"},
					{Name: "Report", Description: "보고서 및 발표 자료 작성"},
				},
			},
		},
	}
}
