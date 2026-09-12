package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"tools/jira/internal/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	application := app.NewDefaultApp()
	exitCode := application.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(exitCode)
}
