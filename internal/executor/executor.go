package executor

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"time"

	desktoppkg "github.com/cirruslabs/mtell/internal/desktop"
	programpkg "github.com/cirruslabs/mtell/internal/program"
	"github.com/cirruslabs/mtell/internal/prompt"
	"github.com/cirruslabs/mtell/internal/vision"
	govnc "github.com/mitchellh/go-vnc"
)

func Execute(ctx context.Context, desktop desktoppkg.Desktop, program *programpkg.Program) error {
	for _, statement := range program.Statements {
		switch stmt := statement.(type) {
		case *programpkg.WaitDuration:
			slog.InfoContext(ctx, "waiting for duration", "duration", stmt.Duration)

			select {
			case <-time.After(stmt.Duration):
				// Proceed
			case <-ctx.Done():
				return ctx.Err()
			}
		case *programpkg.WaitText:
			slog.InfoContext(ctx, "waiting for text", "text", stmt.Text)

			_, err := waitForText(ctx, desktop, stmt.Text)
			if err != nil {
				return fmt.Errorf("failed to wait for text: %w", err)
			}
		case *programpkg.ClickText:
			slog.InfoContext(ctx, "clicking text", "text", stmt.Text)

			rectangle, err := waitForText(ctx, desktop, stmt.Text)
			if err != nil {
				return fmt.Errorf("failed to wait for text: %w", err)
			}

			centerX := (rectangle.Min.X + rectangle.Max.X) / 2
			centerY := (rectangle.Min.Y + rectangle.Max.Y) / 2

			slog.DebugContext(ctx, "clicking at the center point of the found text",
				"x", centerX, "y", centerY)

			if err := desktop.Mouse(ctx, govnc.ButtonLeft, uint16(centerX), uint16(centerY)); err != nil {
				return fmt.Errorf("failed to click at the center point of text: %w", err)
			}

			if err := desktop.Mouse(ctx, 0, uint16(centerX), uint16(centerY)); err != nil {
				return fmt.Errorf("failed to click at the center point of text: %w", err)
			}
		case *programpkg.PressKey:
			slog.InfoContext(ctx, "pressing key", "name", stmt.Name, "code",
				fmt.Sprintf("%#04x", stmt.Code), "mask", stmt.Mask)

			if stmt.Mask&programpkg.KeyOn != 0 {
				if err := desktop.Keyboard(ctx, stmt.Code, true); err != nil {
					return fmt.Errorf("failed to press key down: %w", err)
				}
			}

			if stmt.Mask&programpkg.KeyOff != 0 {
				if err := desktop.Keyboard(ctx, stmt.Code, false); err != nil {
					return fmt.Errorf("failed to press key up: %w", err)
				}
			}
		case *programpkg.TypeText:
			slog.InfoContext(ctx, "typing text", "text", stmt.Text)

			if err := desktoppkg.TypeText(ctx, desktop, stmt.Text); err != nil {
				return fmt.Errorf("failed to type text: %w", err)
			}
		case *programpkg.Prompt:
			slog.InfoContext(ctx, "prompting text", "text", stmt.Text)

			if err := prompt.Prompt(ctx, desktop, stmt.Text); err != nil {
				return fmt.Errorf("failed to prompt text: %w", err)
			}
		}
	}

	return nil
}

func waitForText(ctx context.Context, desktop desktoppkg.Desktop, text string) (*image.Rectangle, error) {
	for {
		image, err := desktop.Screen(ctx)
		if err != nil {
			return nil, err
		}

		rectangle, err := vision.FindTextCoordinates(ctx, image, text)
		if err != nil {
			if errors.Is(err, vision.ErrNotFound) {
				slog.DebugContext(ctx, "no text found", "text", text)

				// Wait for some time before trying again
				select {
				case <-time.After(time.Second):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}

			return nil, err
		}

		return rectangle, nil
	}
}
