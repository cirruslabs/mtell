package prompt

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"log/slog"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	desktoppkg "github.com/cirruslabs/mtell/internal/desktop"
	"github.com/cirruslabs/mtell/internal/desktop/vnc"
	govnc "github.com/mitchellh/go-vnc"
	"golang.org/x/exp/constraints"
)

const waitDuration = 5 * time.Second

func click(ctx context.Context, desktop desktoppkg.Desktop, button string, x int64, y int64) error {
	var mask govnc.ButtonMask

	switch strings.ToLower(button) {
	case "left":
		mask = vnc.ButtonLeft
	case "right":
		mask = vnc.ButtonRight
	case "wheel":
		mask = vnc.ButtonMiddle
	default:
		return fmt.Errorf("unsupported mouse button %q", button)
	}

	if err := mouse(ctx, desktop, mask, x, y); err != nil {
		return err
	}

	return mouse(ctx, desktop, 0, x, y)
}

func drag(ctx context.Context, desktop desktoppkg.Desktop, points []point) error {
	slog.DebugContext(ctx, "dragging")

	if len(points) < 2 {
		return fmt.Errorf("drag path is too short, expected at least 2 elements, got %d", len(points))
	}

	// Start dragging
	start := points[0]

	if err := mouse(ctx, desktop, vnc.ButtonLeft, start.X, start.Y); err != nil {
		return err
	}

	// Continue dragging
	for _, p := range points[1:] {
		if err := mouse(ctx, desktop, vnc.ButtonLeft, p.X, p.Y); err != nil {
			return err
		}
	}

	// Stop dragging
	end := points[len(points)-1]

	return mouse(ctx, desktop, 0, end.X, end.Y)
}

// point represents an x,y coordinate for drag operations.
type point struct {
	X int64
	Y int64
}

func mouse(
	ctx context.Context,
	desktop desktoppkg.Desktop,
	mask govnc.ButtonMask,
	x int64,
	y int64,
	opts ...desktoppkg.MouseOption,
) error {
	if x < 0 || x > math.MaxUint16 {
		return fmt.Errorf("x coordinate for mouse is out of bounds: %d", x)
	}

	if y < 0 || y > math.MaxUint16 {
		return fmt.Errorf("y coordinate for mouse is out of bounds: %d", y)
	}

	return desktop.Mouse(ctx, mask, uint16(x), uint16(y), opts...)
}

func keypress(ctx context.Context, desktop desktoppkg.Desktop, keys []string) error {
	var keyCodes []uint32

	for _, key := range keys {
		// Similarly to normalizePlaywrightKey()[1]
		//
		// [1]: https://github.com/openai/openai-cua-sample-app/blob/3751c8baa6376c0bbf6cceea2cdc0c0b42996e03/packages/runner-core/src/responses-loop.ts#L153
		key = strings.TrimSpace(key)

		// If key is a single character, it is case-sensitive
		if utf8.ValidString(key) && utf8.RuneCountInString(key) == 1 {
			r, _ := utf8.DecodeRuneInString(key)

			keyCodes = append(keyCodes, uint32(r))

			continue
		}

		// Otherwise, it's a key symbol, and it can be case-insensitive
		keyCode, ok := keyMap[strings.ToLower(key)]
		if !ok {
			return fmt.Errorf("unknown key %q", key)
		}

		keyCodes = append(keyCodes, uint32(keyCode))
	}

	for i := range keyCodes {
		slog.DebugContext(ctx, "pressing key", "key", keys[i], "code",
			fmt.Sprintf("%#x", keyCodes[i]))

		if err := desktop.Keyboard(ctx, keyCodes[i], true); err != nil {
			return fmt.Errorf("failed to press key %q: %w", keys[i], err)
		}
	}

	for i := len(keyCodes) - 1; i >= 0; i-- {
		slog.DebugContext(ctx, "releasing key", "key", keys[i], "code",
			fmt.Sprintf("%#x", keyCodes[i]))

		if err := desktop.Keyboard(ctx, keyCodes[i], false); err != nil {
			return fmt.Errorf("failed to release key %q: %w", keys[i], err)
		}
	}

	return nil
}

func scroll(
	ctx context.Context,
	desktop desktoppkg.Desktop,
	x int64,
	y int64,
	scrollX int64,
	scrollY int64,
) error {
	// Scroll horizontally
	if err := scrollAxis(ctx, desktop, x, y, vnc.ButtonScrollLeft,
		vnc.ButtonScrollRight, scrollX); err != nil {
		return fmt.Errorf("failed to scroll horizontally: %w", err)
	}

	// Scroll vertically
	if err := scrollAxis(ctx, desktop, x, y, vnc.ButtonScrollUp,
		vnc.ButtonScrollDown, scrollY); err != nil {
		return fmt.Errorf("failed to scroll vertically: %w", err)
	}

	return nil
}

func scrollAxis(
	ctx context.Context,
	desktop desktoppkg.Desktop,
	x int64,
	y int64,
	buttonNegative govnc.ButtonMask,
	buttonPositive govnc.ButtonMask,
	amountPixels int64,
) error {
	var scrollButton govnc.ButtonMask

	switch {
	case amountPixels < 0:
		scrollButton = buttonNegative
	case amountPixels > 0:
		scrollButton = buttonPositive
	default:
		// Nothing to do
		return nil
	}

	// Determine the number of required mouse clicks
	// needed to scroll the given amount of pixels
	//
	// This is currently hardcoded to only support macOS
	// via Screen Sharing, which scrolls for approximately
	// 1 pixel in Safari for each mouse click
	//
	// Hopefully we'll find a more robust way to scroll
	// in the future, unfortunately the RFB protocol
	// without extensions is pretty basic in this regard
	const pixelsPerClick = 1

	amountPixelsAbs := abs(amountPixels)

	steps := amountPixelsAbs / pixelsPerClick
	if amountPixelsAbs%pixelsPerClick > 0 {
		steps++
	}

	// Use a shorter delay because we scroll only by a single pixel each mouse click
	delay := 5 * time.Millisecond

	for range steps {
		if err := mouse(ctx, desktop, scrollButton, x, y, desktoppkg.WithMouseDelay(delay)); err != nil {
			return err
		}

		if err := mouse(ctx, desktop, 0, x, y, desktoppkg.WithMouseDelay(delay)); err != nil {
			return err
		}
	}

	return nil
}

func abs[T constraints.Signed](n T) T {
	if n < 0 {
		return -n
	}

	return n
}

// screenshotPNG captures the screen and returns PNG bytes.
func screenshotPNG(ctx context.Context, desktop desktoppkg.Desktop) ([]byte, error) {
	image, err := desktop.Screen(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen: %w", err)
	}

	var pngBuffer bytes.Buffer

	if err := png.Encode(&pngBuffer, image); err != nil {
		return nil, err
	}

	return pngBuffer.Bytes(), nil
}

// screenshotDataURL captures the screen and returns a data URL string.
func screenshotDataURL(ctx context.Context, desktop desktoppkg.Desktop) (string, error) {
	pngBytes, err := screenshotPNG(ctx, desktop)
	if err != nil {
		return "", err
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes), nil
}
