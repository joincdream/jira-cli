package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ConfigureCommand provides an interactive wizard to create/update Jira configuration.
type ConfigureCommand struct {
	stdin io.Reader
}

func NewConfigureCommand(stdin io.Reader) *ConfigureCommand {
	if stdin == nil {
		stdin = os.Stdin
	}
	return &ConfigureCommand{stdin: stdin}
}

func (c *ConfigureCommand) Name() string        { return "configure" }
func (c *ConfigureCommand) Aliases() []string  { return []string{"configuration", "config"} }
func (c *ConfigureCommand) Description() string { return "대화형으로 Jira 접속 정보(.jira.json)를 설정합니다." }

type configData struct {
	InstanceURL string `json:"instance_url"`
	Email       string `json:"email"`
	APIToken    string `json:"api_token"`
	ProjectKey  string `json:"project_key"`
}

func (c *ConfigureCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	isGlobal := false
	for _, arg := range args {
		if arg == "--global" || arg == "-g" {
			isGlobal = true
			break
		}
	}

	targetPath := ".jira.json"
	if isGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("홈 디렉토리를 찾을 수 없습니다: %w", err)
		}
		targetPath = filepath.Join(home, ".config", "jira", "config.json")
	}

	// Read existing configuration if available
	var existing configData
	if data, err := os.ReadFile(targetPath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	reader := bufio.NewReader(c.stdin)

	prompt := func(field, label, defaultValue, currentSecret string) (string, error) {
		displayDefault := defaultValue
		if currentSecret != "" {
			if len(currentSecret) > 8 {
				displayDefault = currentSecret[:4] + "..." + currentSecret[len(currentSecret)-4:]
			} else {
				displayDefault = "********"
			}
		}

		if displayDefault != "" {
			fmt.Fprintf(stdout, "%s [%s]: ", label, displayDefault)
		} else {
			fmt.Fprintf(stdout, "%s: ", label)
		}

		input, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		input = strings.TrimSpace(input)
		if input == "" {
			if currentSecret != "" {
				return currentSecret, nil
			}
			return defaultValue, nil
		}
		return input, nil
	}

	fmt.Fprintln(stdout, "🔧 Jira CLI 환경 설정 마법사")
	fmt.Fprintf(stdout, "설정 파일 대상: %s\n\n", targetPath)

	defaultURL := existing.InstanceURL
	if defaultURL == "" {
		defaultURL = "https://joincdream.atlassian.net"
	}
	instanceURL, err := prompt("instance_url", "Jira Instance URL", defaultURL, "")
	if err != nil {
		return err
	}

	email, err := prompt("email", "Jira Account Email", existing.Email, "")
	if err != nil {
		return err
	}

	apiToken, err := prompt("api_token", "Jira API Token", "", existing.APIToken)
	if err != nil {
		return err
	}

	defaultKey := existing.ProjectKey
	if defaultKey == "" {
		defaultKey = "KAN"
	}
	projectKey, err := prompt("project_key", "Default Project Key", defaultKey, "")
	if err != nil {
		return err
	}

	cfg := configData{
		InstanceURL: strings.TrimRight(instanceURL, "/"),
		Email:       email,
		APIToken:    apiToken,
		ProjectKey:  projectKey,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 인코딩 실패: %w", err)
	}
	data = append(data, '\n')

	// Ensure parent dir exists for global config
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil && filepath.Dir(targetPath) != "." {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	if err := os.WriteFile(targetPath, data, 0600); err != nil {
		return fmt.Errorf("설정 파일 저장 실패 (%s): %w", targetPath, err)
	}

	fmt.Fprintf(stdout, "\n✅ Jira 설정이 성공적으로 저장되었습니다: %s\n", targetPath)
	return nil
}
