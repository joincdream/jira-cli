package pkg

import (
	"strings"
	"testing"
)

func TestBuildADFDocument(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLines int
	}{
		{
			name:      "single line",
			input:     "Hello Jira",
			wantLines: 1,
		},
		{
			name:      "multi lines with empty lines",
			input:     "Line 1\n\nLine 2\nLine 3\n",
			wantLines: 3,
		},
		{
			name:      "empty string",
			input:     "",
			wantLines: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := BuildADFDocument(tt.input)
			if doc["type"] != "doc" {
				t.Errorf("expected doc type 'doc', got %v", doc["type"])
			}
			if doc["version"] != 1 {
				t.Errorf("expected doc version 1, got %v", doc["version"])
			}

			content, ok := doc["content"].([]map[string]interface{})
			if !ok {
				t.Fatalf("expected []map[string]interface{} content, got %T", doc["content"])
			}

			if len(content) != tt.wantLines {
				t.Errorf("expected %d paragraphs, got %d", tt.wantLines, len(content))
			}
		})
	}
}

func TestExtractTextFromADF(t *testing.T) {
	t.Run("nil body", func(t *testing.T) {
		res := ExtractTextFromADF(nil)
		if res != "" {
			t.Errorf("expected empty string for nil, got %q", res)
		}
	})

	t.Run("plain string body", func(t *testing.T) {
		input := "Plain text description"
		res := ExtractTextFromADF(input)
		if res != input {
			t.Errorf("expected %q, got %q", input, res)
		}
	})

	t.Run("valid ADF structure", func(t *testing.T) {
		inputDoc := BuildADFDocument("First line\nSecond line")
		res := ExtractTextFromADF(inputDoc)

		expectedLines := []string{"First line", "Second line"}
		for _, line := range expectedLines {
			if !strings.Contains(res, line) {
				t.Errorf("expected result to contain %q, got %q", line, res)
			}
		}
	})
}
