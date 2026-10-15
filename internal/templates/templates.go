package templates

import (
	"embed"
	"fmt"
)

//go:embed cursor.mdc claude.md skill.md
var templateFS embed.FS

// GetTemplate reads an embedded template file by its name.
func GetTemplate(name string) ([]byte, error) {
	data, err := templateFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template %q: %w", name, err)
	}
	return data, nil
}
