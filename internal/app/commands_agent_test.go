package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAgentCommand_All(t *testing.T) {
	tempDir := t.TempDir()
	cmd := NewInitAgentCommand().WithBaseDir(tempDir)

	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{"--type", "all"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(".cursor", "rules", "jira.mdc"),
		"CLAUDE.md",
		filepath.Join(".agents", "skills", "jira", "SKILL.md"),
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(tempDir, file)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("expected file %q to be created, but got error: %v", file, err)
			continue
		}
		content := string(data)
		if !strings.Contains(content, "Agent Insight Block") && !strings.Contains(content, "jira") {
			t.Errorf("file %q should contain Jira Agent guidelines, got:\n%s", file, content)
		}
	}
}

func TestInitAgentCommand_SpecificType(t *testing.T) {
	tempDir := t.TempDir()
	cmd := NewInitAgentCommand().WithBaseDir(tempDir)

	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{"--type", "cursor"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cursorFile := filepath.Join(tempDir, ".cursor", "rules", "jira.mdc")
	if _, err := os.Stat(cursorFile); err != nil {
		t.Errorf("expected %q to exist, err: %v", cursorFile, err)
	}

	claudeFile := filepath.Join(tempDir, "CLAUDE.md")
	if _, err := os.Stat(claudeFile); !os.IsNotExist(err) {
		t.Errorf("expected %q NOT to exist when --type cursor is used", claudeFile)
	}
}

func TestInitAgentCommand_SkipExistingWithoutForce(t *testing.T) {
	tempDir := t.TempDir()
	claudePath := filepath.Join(tempDir, "CLAUDE.md")
	customContent := "Custom User Content"
	if err := os.WriteFile(claudePath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	cmd := NewInitAgentCommand().WithBaseDir(tempDir)
	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{"--type", "claude"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(claudePath)
	if string(data) != customContent {
		t.Errorf("existing file should not be overwritten without --force, got: %q", string(data))
	}
	if !strings.Contains(stdout.String(), "건너뜀") && !strings.Contains(stdout.String(), "Skipped") {
		t.Errorf("stdout should mention skipped file, got:\n%s", stdout.String())
	}
}

func TestInitAgentCommand_ForceOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	claudePath := filepath.Join(tempDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("Old Content"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	cmd := NewInitAgentCommand().WithBaseDir(tempDir)
	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{"--type", "claude", "--force"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(claudePath)
	if string(data) == "Old Content" {
		t.Errorf("file should have been overwritten with --force")
	}
}

func TestInitAgentCommand_InvalidType(t *testing.T) {
	tempDir := t.TempDir()
	cmd := NewInitAgentCommand().WithBaseDir(tempDir)

	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{"--type", "unknown_agent"}, &stdout, &stderr)
	if err == nil {
		t.Fatalf("expected error for invalid type, got nil")
	}
}

func TestInitAgentCommand_Help(t *testing.T) {
	cmd := NewInitAgentCommand()
	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{"--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error on --help: %v", err)
	}
	if !strings.Contains(stdout.String(), "init-agent") {
		t.Errorf("expected usage output, got:\n%s", stdout.String())
	}
}

func TestInitAgentCommand_Interactive_CustomSelection(t *testing.T) {
	tempDir := t.TempDir()
	// Input: select 1 (cursor) and 3 (antigravity)
	fakeInput := strings.NewReader("1,3\n")
	cmd := NewInitAgentCommand().WithBaseDir(tempDir).WithStdin(fakeInput)

	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cursorPath := filepath.Join(tempDir, ".cursor", "rules", "jira.mdc")
	if _, err := os.Stat(cursorPath); err != nil {
		t.Errorf("expected cursor rule to exist: %v", err)
	}

	agyPath := filepath.Join(tempDir, ".agents", "skills", "jira", "SKILL.md")
	if _, err := os.Stat(agyPath); err != nil {
		t.Errorf("expected antigravity skill to exist: %v", err)
	}

	claudePath := filepath.Join(tempDir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); !os.IsNotExist(err) {
		t.Errorf("expected CLAUDE.md NOT to exist when not selected")
	}
}

func TestInitAgentCommand_Interactive_DefaultAll(t *testing.T) {
	tempDir := t.TempDir()
	// Input: Enter (empty string) defaults to All
	fakeInput := strings.NewReader("\n")
	cmd := NewInitAgentCommand().WithBaseDir(tempDir).WithStdin(fakeInput)

	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, file := range []string{
		filepath.Join(".cursor", "rules", "jira.mdc"),
		"CLAUDE.md",
		filepath.Join(".agents", "skills", "jira", "SKILL.md"),
	} {
		if _, err := os.Stat(filepath.Join(tempDir, file)); err != nil {
			t.Errorf("expected %q to exist on Enter (all): %v", file, err)
		}
	}
}

func TestInitAgentCommand_Interactive_InvalidChoice(t *testing.T) {
	tempDir := t.TempDir()
	fakeInput := strings.NewReader("9,invalid\n")
	cmd := NewInitAgentCommand().WithBaseDir(tempDir).WithStdin(fakeInput)

	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{}, &stdout, &stderr)
	if err == nil {
		t.Fatalf("expected error on invalid choice, got nil")
	}
}
