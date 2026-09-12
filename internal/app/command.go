package app

import (
	"context"
	"io"
)

// Command represents an executable CLI command.
type Command interface {
	Name() string
	Aliases() []string
	Description() string
	Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error
}
