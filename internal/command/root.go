package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cirruslabs/mtell/internal/desktop"
	vncpkg "github.com/cirruslabs/mtell/internal/desktop/vnc"
	"github.com/cirruslabs/mtell/internal/executor"
	"github.com/cirruslabs/mtell/internal/logginglevel"
	programpkg "github.com/cirruslabs/mtell/internal/program"
	"github.com/cirruslabs/mtell/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	VNC        string
	InputDelay time.Duration
	Debug      bool
}

func NewRootCommand() *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:           "mtell PROGRAM",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.FullVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Debug {
				logginglevel.Level.Set(slog.LevelDebug)
			}

			return run(cmd, args, opts)
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&opts.VNC, "vnc", "", "machine's VNC server address "+
		"or URL (e.g. \"example.com:5900\" or \"vnc://:password@example.com:5900\")")
	cmd.Flags().DurationVar(&opts.InputDelay, "input-delay", 100*time.Millisecond,
		"delay between input actions (mouse clicks, keyboard key presses, etc.)")
	cmd.Flags().BoolVar(&opts.Debug, "debug", false, "enable verbose output")

	return cmd
}

func run(cmd *cobra.Command, args []string, opts *Options) error {
	// Parse the program
	program, err := programpkg.ParseString(args[0])
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	// Connect to a desktop
	var desktop desktop.Desktop

	switch {
	case opts.VNC != "":
		desktop, err = vncpkg.New(cmd.Context(), opts.VNC, opts.InputDelay)
	default:
		return fmt.Errorf("please specify \"--vnc\"")
	}
	if err != nil {
		return fmt.Errorf("failed to connect to a desktop: %w", err)
	}
	defer desktop.Close()

	slog.Info("connected to a desktop using VNC", "vnc", opts.VNC)

	// Run the desktop session
	subCtx, subCtxCancel := context.WithCancel(cmd.Context())
	group, groupCtx := errgroup.WithContext(subCtx)

	group.Go(func() error {
		if err := desktop.Run(groupCtx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}

		return nil
	})

	// Execute the program
	group.Go(func() error {
		defer subCtxCancel()

		return executor.Execute(groupCtx, desktop, program)
	})

	return group.Wait()
}
