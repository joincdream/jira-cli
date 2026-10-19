package app

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tools/jira/internal/i18n"
	"tools/jira/internal/templates"
)

// AgentTarget represents a target AI agent configuration file.
type AgentTarget struct {
	Type        string
	Path        string
	TemplateKey string
	Desc        string
}

// AvailableAgentTargets returns the list of all supported agent targets.
func AvailableAgentTargets() []AgentTarget {
	return []AgentTarget{
		{
			Type:        "cursor",
			Path:        filepath.Join(".cursor", "rules", "jira.mdc"),
			TemplateKey: "cursor.mdc",
			Desc:        "Cursor IDE Agent Rule",
		},
		{
			Type:        "claude",
			Path:        "CLAUDE.md",
			TemplateKey: "claude.md",
			Desc:        "Claude Code Project Guidelines",
		},
		{
			Type:        "antigravity",
			Path:        filepath.Join(".agents", "skills", "jira", "SKILL.md"),
			TemplateKey: "skill.md",
			Desc:        "Antigravity Agent SKILL",
		},
	}
}

// InitAgentCommand creates AI agent configuration and rules files in the current repository.
type InitAgentCommand struct {
	baseDir string    // Working directory for file creation (empty means current directory)
	stdin   io.Reader // Input stream for interactive prompts (defaults to os.Stdin)
}

// NewInitAgentCommand creates an InitAgentCommand.
func NewInitAgentCommand() *InitAgentCommand {
	return &InitAgentCommand{
		baseDir: "",
		stdin:   os.Stdin,
	}
}

// WithBaseDir sets a custom base directory (useful for unit testing).
func (c *InitAgentCommand) WithBaseDir(dir string) *InitAgentCommand {
	c.baseDir = dir
	return c
}

// WithStdin sets a custom stdin reader (useful for testing interactive prompts).
func (c *InitAgentCommand) WithStdin(stdin io.Reader) *InitAgentCommand {
	c.stdin = stdin
	return c
}

func (c *InitAgentCommand) Name() string        { return "init-agent" }
func (c *InitAgentCommand) Aliases() []string  { return []string{"agent-init", "setup-agent"} }
func (c *InitAgentCommand) Description() string { return i18n.T("cmd.init_agent.desc") }

func (c *InitAgentCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.init_agent.usage"))
		return nil
	}

	fs := flag.NewFlagSet("init-agent", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var agentType string
	var force bool

	fs.StringVar(&agentType, "type", "", i18n.T("cmd.init_agent.flag_type"))
	fs.BoolVar(&force, "force", false, i18n.T("cmd.init_agent.flag_force"))
	fs.BoolVar(&force, "f", false, i18n.T("cmd.init_agent.flag_force"))

	if _, err := parseFlagsAndPositional(fs, args); err != nil {
		return err
	}

	allTargets := AvailableAgentTargets()
	var targets []AgentTarget

	// Determine targets: Interactive prompt if --type was NOT provided and stdin is available
	typeExplicitlyPassed := false
	for _, arg := range args {
		if arg == "--type" || strings.HasPrefix(arg, "--type=") || strings.HasPrefix(arg, "-type=") {
			typeExplicitlyPassed = true
			break
		}
	}

	if !typeExplicitlyPassed && c.stdin != nil {
		// Run interactive selection prompt
		fmt.Fprintf(stdout, "%s", i18n.T("cmd.init_agent.prompt_title"))

		reader := bufio.NewReader(c.stdin)
		input, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read interactive input: %w", err)
		}

		selected, err := parseInteractiveSelection(strings.TrimSpace(input), allTargets)
		if err != nil {
			return err
		}
		targets = selected
		fmt.Fprintln(stdout)
	} else {
		// Non-interactive / explicit --type flag
		normType := strings.ToLower(strings.TrimSpace(agentType))
		if normType == "all" || normType == "" {
			targets = allTargets
		} else {
			for _, t := range allTargets {
				if t.Type == normType {
					targets = append(targets, t)
					break
				}
			}
			if len(targets) == 0 {
				return fmt.Errorf("%s", i18n.Sprintf("cmd.init_agent.err_invalid_type", normType))
			}
		}
	}

	base := c.baseDir
	if base == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		base = cwd
	}

	fmt.Fprint(stdout, i18n.Sprintf("cmd.init_agent.header", base))

	for _, target := range targets {
		destPath := filepath.Join(base, target.Path)

		// Check if file already exists
		if _, err := os.Stat(destPath); err == nil && !force {
			fmt.Fprint(stdout, i18n.Sprintf("cmd.init_agent.skipped", target.Desc, target.Path))
			continue
		}

		data, err := templates.GetTemplate(target.TemplateKey)
		if err != nil {
			return fmt.Errorf("failed to load template %q: %w", target.TemplateKey, err)
		}

		// Ensure directory exists
		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %q: %w", destPath, err)
		}

		fmt.Fprint(stdout, i18n.Sprintf("cmd.init_agent.created", target.Desc, target.Path))
	}

	fmt.Fprint(stdout, i18n.Sprintf("cmd.init_agent.footer"))
	return nil
}

// parseInteractiveSelection parses comma or space separated numbers (1, 2, 3, 4) into AgentTargets.
func parseInteractiveSelection(input string, allTargets []AgentTarget) ([]AgentTarget, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == "4" || strings.EqualFold(trimmed, "all") {
		return allTargets, nil
	}

	// Split by comma or whitespace
	tokens := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})

	seen := make(map[string]bool)
	var result []AgentTarget

	for _, token := range tokens {
		t := strings.TrimSpace(token)
		if t == "" {
			continue
		}

		var targetType string
		switch t {
		case "1", "cursor":
			targetType = "cursor"
		case "2", "claude":
			targetType = "claude"
		case "3", "antigravity":
			targetType = "antigravity"
		case "4", "all":
			return allTargets, nil
		default:
			return nil, fmt.Errorf("%s", i18n.Sprintf("cmd.init_agent.prompt_invalid", t))
		}

		if !seen[targetType] {
			seen[targetType] = true
			for _, item := range allTargets {
				if item.Type == targetType {
					result = append(result, item)
					break
				}
			}
		}
	}

	if len(result) == 0 {
		return allTargets, nil
	}
	return result, nil
}
