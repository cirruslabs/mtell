package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cirruslabs/mtell/internal/command"
	"github.com/cirruslabs/mtell/internal/logginglevel"
	"github.com/lmittmann/tint"
)

func main() {
	if err := mainImpl(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func mainImpl() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Use a colored logging handler for log/slog
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level: logginglevel.Level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return command.NewRootCommand().ExecuteContext(ctx)
}
