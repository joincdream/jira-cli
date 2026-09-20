package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"tools/jira/pkg"
)

// ConfigureCommand provides an interactive wizard to create/update Jira configuration.
type ConfigureCommand struct {
	stdin      io.Reader
	configPath string
}

func NewConfigureCommand(stdin io.Reader) *ConfigureCommand {
	if stdin == nil {
		stdin = os.Stdin
	}
	return &ConfigureCommand{stdin: stdin}
}

func NewConfigureCommandWithPath(stdin io.Reader, configPath string) *ConfigureCommand {
	if stdin == nil {
		stdin = os.Stdin
	}
	return &ConfigureCommand{stdin: stdin, configPath: configPath}
}

func (c *ConfigureCommand) Name() string        { return "configure" }
func (c *ConfigureCommand) Aliases() []string  { return []string{"configuration", "config"} }
func (c *ConfigureCommand) Description() string { return "Jira 계정 프로필(~/.config/jira/config)을 대화형으로 설정합니다." }

func (c *ConfigureCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	targetPath := c.configPath
	if targetPath == "" {
		p, err := pkg.GetDefaultConfigPath()
		if err != nil {
			return err
		}
		targetPath = p
	}

	profile := pkg.GetActiveProfile()
	// Parse args for subcommands or flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "list" || arg == "ls" {
			return c.listProfiles(targetPath, stdout)
		}
		if arg == "--profile" && i+1 < len(args) {
			profile = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--profile=") {
			profile = strings.TrimPrefix(arg, "--profile=")
		}
	}

	if profile == "" {
		profile = "default"
	}

	// Read existing profile config if available
	existing, _, _ := pkg.ReadProfileConfig(targetPath, profile)

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
	fmt.Fprintf(stdout, "설정 파일 대상: %s\n", targetPath)
	fmt.Fprintf(stdout, "대상 프로필:   [%s]\n\n", profile)

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

	cfg := pkg.Config{
		InstanceURL: strings.TrimRight(instanceURL, "/"),
		Email:       email,
		APIToken:    apiToken,
		ProjectKey:  projectKey,
	}

	if err := pkg.WriteProfileConfig(targetPath, profile, cfg); err != nil {
		return fmt.Errorf("설정 파일 저장 실패 (%s): %w", targetPath, err)
	}

	fmt.Fprintf(stdout, "\n✅ Jira 설정이 성공적으로 저장되었습니다: %s (프로필: [%s])\n", targetPath, profile)
	return nil
}

func (c *ConfigureCommand) listProfiles(configPath string, stdout io.Writer) error {
	profiles, err := pkg.ListProfiles(configPath)
	if err != nil {
		return fmt.Errorf("프로필 목록 조회 실패 (%s): %w", configPath, err)
	}

	if len(profiles) == 0 {
		fmt.Fprintf(stdout, "등록된 Jira 프로필이 없습니다 (%s).\n'jira configure' 명령어로 프로필을 생성하세요.\n", configPath)
		return nil
	}

	active := pkg.GetActiveProfile()
	fmt.Fprintf(stdout, "설정 파일: %s\n\n", configPath)
	fmt.Fprintln(stdout, "등록된 프로필 목록:")
	for _, p := range profiles {
		cfg, _, _ := pkg.ReadProfileConfig(configPath, p)
		marker := "  "
		if strings.EqualFold(p, active) {
			marker = "* "
		}
		summary := ""
		if cfg.InstanceURL != "" || cfg.Email != "" {
			summary = fmt.Sprintf(" (%s, %s)", cfg.InstanceURL, cfg.Email)
		}
		if marker == "* " {
			fmt.Fprintf(stdout, "%s[%s]%s [현재 활성]\n", marker, p, summary)
		} else {
			fmt.Fprintf(stdout, "%s[%s]%s\n", marker, p, summary)
		}
	}
	return nil
}
