package executor

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/cirruslabs/mtell/internal/desktop"
	"github.com/cirruslabs/mtell/internal/keymap"
	"github.com/cirruslabs/mtell/internal/program"
	programpkg "github.com/cirruslabs/mtell/internal/program"
	"github.com/cirruslabs/mtell/internal/vision"
	govnc "github.com/mitchellh/go-vnc"
)

func Execute(ctx context.Context, desktop desktop.Desktop, program *program.Program) error {
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

			for _, c := range stmt.Text {
				if err := typeRune(ctx, desktop, c); err != nil {
					return fmt.Errorf("failed to type text: %w", err)
				}

			}
		}
	}

	return nil
}

func waitForText(ctx context.Context, desktop desktop.Desktop, text string) (*image.Rectangle, error) {
	for {
		image, err := desktop.Screen(ctx)
		if err != nil {
			return nil, err
		}

		rectangle, err := vision.FindTextCoordinates(image, text)
		if err != nil {
			if errors.Is(err, vision.ErrNotFound) {
				continue
			}

			return nil, err
		}

		return rectangle, nil
	}
}

func typeRune(ctx context.Context, desktop desktop.Desktop, r rune) error {
	useShift := unicode.IsUpper(r) || strings.ContainsRune("!@#$%^&*()_+{}:\"|<>?", r)

	if useShift {
		if err := desktop.Keyboard(ctx, keymap.XK_Shift_L, true); err != nil {
			return err
		}
	}

	if err := desktop.Keyboard(ctx, uint32(r), true); err != nil {
		return err
	}

	if err := desktop.Keyboard(ctx, uint32(r), false); err != nil {
		return err
	}

	if useShift {
		if err := desktop.Keyboard(ctx, keymap.XK_Shift_L, false); err != nil {
			return err
		}
	}

	return nil
}
