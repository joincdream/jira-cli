package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"tools/jira/internal/i18n"
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
func (c *ConfigureCommand) Description() string { return i18n.T("cmd.configure.desc") }

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

	fmt.Fprintln(stdout, i18n.T("cmd.configure.wizard_title"))
	fmt.Fprint(stdout, i18n.Sprintf("cmd.configure.target_file", targetPath))
	fmt.Fprint(stdout, i18n.Sprintf("cmd.configure.target_profile", profile))

	existingProfiles, _ := pkg.ListProfiles(targetPath)
	hasBaseProfiles := false
	for _, p := range existingProfiles {
		if !strings.EqualFold(p, profile) {
			hasBaseProfiles = true
			break
		}
	}

	// If configuring a non-default profile and other profiles exist, ask if user wants to inherit credentials
	if !strings.EqualFold(profile, "default") && hasBaseProfiles {
		defaultInherit := "Y"
		if existing.InstanceURL != "" && existing.SourceProfile == "" {
			defaultInherit = "N"
		}

		inheritChoice, err := prompt("inherit", i18n.T("cmd.configure.prompt_inherit"), defaultInherit, "")
		if err != nil {
			return err
		}

		if strings.EqualFold(inheritChoice, "y") || strings.EqualFold(inheritChoice, "yes") {
			defaultSource := existing.SourceProfile
			if defaultSource == "" {
				defaultSource = "default"
			}
			sourceProf, err := prompt("source_profile", i18n.T("cmd.configure.prompt_source_profile"), defaultSource, "")
			if err != nil {
				return err
			}

			fmt.Fprintf(stdout, "%s", i18n.Sprintf("cmd.configure.skip_credentials_notice", sourceProf))

			defaultKey := existing.ProjectKey
			if defaultKey == "" {
				defaultKey = "KAN"
			}
			projectKey, err := prompt("project_key", i18n.T("cmd.configure.prompt_project_key"), defaultKey, "")
			if err != nil {
				return err
			}

			cfg := pkg.Config{
				SourceProfile: sourceProf,
				ProjectKey:    projectKey,
				Language:      existing.Language,
			}

			if err := pkg.WriteProfileConfig(targetPath, profile, cfg); err != nil {
				return fmt.Errorf("%s", i18n.Sprintf("cmd.configure.err_save", targetPath, err))
			}

			fmt.Fprint(stdout, i18n.Sprintf("cmd.configure.save_success", targetPath, profile))
			return nil
		}
	}

	defaultURL := existing.InstanceURL
	if defaultURL == "" {
		defaultURL = "https://joincdream.atlassian.net"
	}
	instanceURL, err := prompt("instance_url", i18n.T("cmd.configure.prompt_instance_url"), defaultURL, "")
	if err != nil {
		return err
	}

	email, err := prompt("email", i18n.T("cmd.configure.prompt_email"), existing.Email, "")
	if err != nil {
		return err
	}

	apiToken, err := prompt("api_token", i18n.T("cmd.configure.prompt_api_token"), "", existing.APIToken)
	if err != nil {
		return err
	}

	defaultKey := existing.ProjectKey
	if defaultKey == "" {
		defaultKey = "KAN"
	}
	projectKey, err := prompt("project_key", i18n.T("cmd.configure.prompt_project_key"), defaultKey, "")
	if err != nil {
		return err
	}

	cfg := pkg.Config{
		InstanceURL: strings.TrimRight(instanceURL, "/"),
		Email:       email,
		APIToken:    apiToken,
		ProjectKey:  projectKey,
		Language:    existing.Language,
	}

	if err := pkg.WriteProfileConfig(targetPath, profile, cfg); err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.configure.err_save", targetPath, err))
	}

	fmt.Fprint(stdout, i18n.Sprintf("cmd.configure.save_success", targetPath, profile))
	return nil
}

func (c *ConfigureCommand) listProfiles(configPath string, stdout io.Writer) error {
	profiles, err := pkg.ListProfiles(configPath)
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.configure.err_list_profiles", configPath, err))
	}

	if len(profiles) == 0 {
		fmt.Fprint(stdout, i18n.Sprintf("cmd.configure.no_profiles", configPath))
		return nil
	}

	active := pkg.GetActiveProfile()
	fmt.Fprintf(stdout, "%s\n", i18n.Sprintf("cmd.configure.list_header", configPath))
	for _, p := range profiles {
		cfg, _, _ := pkg.ReadProfileConfig(configPath, p)
		marker := "  "
		if strings.EqualFold(p, active) {
			marker = "* "
		}
		summary := ""
		if cfg.SourceProfile != "" {
			summary = fmt.Sprintf(" (inherits: %s, project: %s)", cfg.SourceProfile, cfg.ProjectKey)
		} else if cfg.InstanceURL == "" && cfg.APIToken == "" && cfg.ProjectKey != "" {
			summary = fmt.Sprintf(" (inherits: default, project: %s)", cfg.ProjectKey)
		} else if cfg.InstanceURL != "" || cfg.Email != "" {
			summary = fmt.Sprintf(" (%s, %s, project: %s)", cfg.InstanceURL, cfg.Email, cfg.ProjectKey)
		}
		if marker == "* " {
			fmt.Fprintf(stdout, "%s[%s]%s %s\n", marker, p, summary, i18n.T("cmd.configure.active_profile_marker"))
		} else {
			fmt.Fprintf(stdout, "%s[%s]%s\n", marker, p, summary)
		}
	}
	return nil
}
