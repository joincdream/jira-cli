package pkg

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateAndNormalizeLabels(t *testing.T) {
	catalog := getDefaultCatalog()

	t.Run("valid labels with exact match", func(t *testing.T) {
		input := []string{"DIVE", "AI", "Planning"}
		normalized, invalid, err := catalog.ValidateAndNormalizeLabels(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(invalid) > 0 {
			t.Errorf("expected 0 invalid labels, got %v", invalid)
		}
		if len(normalized) != 3 || normalized[0] != "DIVE" || normalized[1] != "AI" || normalized[2] != "Planning" {
			t.Errorf("unexpected normalized output: %v", normalized)
		}
	})

	t.Run("case-insensitive normalization", func(t *testing.T) {
		input := []string{"dive", "ai", "agentgo_studio"}
		normalized, invalid, err := catalog.ValidateAndNormalizeLabels(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(invalid) > 0 {
			t.Errorf("expected 0 invalid labels, got %v", invalid)
		}
		if normalized[0] != "DIVE" || normalized[1] != "AI" || normalized[2] != "AgentGo_Studio" {
			t.Errorf("normalization failed, got: %v", normalized)
		}
	})

	t.Run("invalid labels detected", func(t *testing.T) {
		input := []string{"DIVE", "RandomTag", "UnknownLabel"}
		normalized, invalid, err := catalog.ValidateAndNormalizeLabels(input)
		if err == nil {
			t.Fatal("expected error for invalid labels, got nil")
		}
		if len(invalid) != 2 || invalid[0] != "RandomTag" || invalid[1] != "UnknownLabel" {
			t.Errorf("expected invalid labels [RandomTag UnknownLabel], got %v", invalid)
		}
		if len(normalized) != 1 || normalized[0] != "DIVE" {
			t.Errorf("expected 1 valid normalized label, got %v", normalized)
		}
	})
}

func TestPrintLabelsCatalog(t *testing.T) {
	catalog := getDefaultCatalog()
	var buf bytes.Buffer
	PrintLabelsCatalog(&buf, catalog)

	out := buf.String()
	if !strings.Contains(out, "DIVE") || !strings.Contains(out, "AgentGo_Studio") || !strings.Contains(out, "Planning") {
		t.Errorf("catalog output missing expected labels: %s", out)
	}
}
