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
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"golang.org/x/exp/constraints"
)

const (
	model = openai.ChatModelGPT5_4

	computerInstructions = "You are controlling a desktop over VNC. " +
		"Use the screenshot as the source of truth. Before typing, pressing Enter, " +
		"or using keyboard shortcuts, make sure the intended element or window is focused."

	waitDuration = 5 * time.Second
)

func Prompt(ctx context.Context, desktop desktoppkg.Desktop, text string) error {
	client := openai.NewClient()

	input := responses.ResponseInputParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Role: responses.EasyInputMessageRoleUser,
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: openai.String(text),
				},
			},
		},
	}

	var previousResponseID param.Opt[string]

	for {
		slog.DebugContext(ctx, "calling LLM", "model", model)

		params := responses.ResponseNewParams{
			Instructions: openai.String(computerInstructions),
			Model:        model,
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: input,
			},
			Tools: []responses.ToolUnionParam{
				{
					OfComputer: new(responses.NewComputerToolParam()),
				},
			},
			PreviousResponseID: previousResponseID,
		}

		response, err := client.Responses.New(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to call LLM: %w", err)
		}

		input = responses.ResponseInputParam{}

		for _, item := range response.Output {
			computerCall, ok := item.AsAny().(responses.ResponseComputerToolCall)
			if !ok {
				slog.DebugContext(ctx, "skipping non-computer tool call item",
					"type", item.Type,
				)

				continue
			}

			if len(computerCall.PendingSafetyChecks) != 0 {
				return fmt.Errorf("computer call %q has %d pending safety checks, but "+
					"pending safety checks are not supported yet", computerCall.CallID,
					len(computerCall.PendingSafetyChecks))
			}

			if len(computerCall.Actions) == 0 {
				return fmt.Errorf("no actions in computer call")
			}

			for _, action := range computerCall.Actions {
				if err := handleAction(ctx, desktop, action); err != nil {
					return err
				}
			}

			screenshotOutput, err := screenshot(ctx, desktop)
			if err != nil {
				return err
			}

			input = append(input, responses.ResponseInputItemParamOfComputerCallOutput(
				computerCall.CallID,
				screenshotOutput,
			))
		}

		if len(input) == 0 {
			slog.DebugContext(ctx, "no longer requesting computer tool calls, looks like we're done")

			if outputText := response.OutputText(); outputText != "" {
				slog.InfoContext(ctx, "returned an answer to a prompt", "text", outputText)
			}

			return nil
		}

		previousResponseID = openai.String(response.ID)
	}
}

func handleAction(
	ctx context.Context,
	desktop desktoppkg.Desktop,
	action responses.ComputerActionUnion,
) error {
	switch typedAction := action.AsAny().(type) {
	case responses.ComputerActionClick:
		slog.InfoContext(ctx, "clicking", "button", typedAction.Button,
			"x", typedAction.X, "y", typedAction.Y)

		return click(ctx, desktop, typedAction.Button, typedAction.X, typedAction.Y)
	case responses.ComputerActionDoubleClick:
		slog.InfoContext(ctx, "double clicking",
			"x", typedAction.X, "y", typedAction.Y)

		if err := click(ctx, desktop, "left", typedAction.X, typedAction.Y); err != nil {
			return err
		}

		return click(ctx, desktop, "left", typedAction.X, typedAction.Y)
	case responses.ComputerActionDrag:
		slog.InfoContext(ctx, "dragging", "num_points", len(typedAction.Path))

		return drag(ctx, desktop, typedAction.Path)
	case responses.ComputerActionKeypress:
		slog.InfoContext(ctx, "pressing keys", "keys", typedAction.Keys)

		return keypress(ctx, desktop, typedAction.Keys)
	case responses.ComputerActionMove:
		slog.InfoContext(ctx, "moving mouse", "x", typedAction.X, "y", typedAction.Y)

		return mouse(ctx, desktop, 0, typedAction.X, typedAction.Y)
	case responses.ComputerActionScreenshot:
		// We unconditionally attach the screenshot after processing each computer call
		// so let's just acknowledge the screenshot request
		slog.InfoContext(ctx, "taking screenshot")

		return nil
	case responses.ComputerActionScroll:
		slog.InfoContext(ctx, "scrolling", "x", typedAction.X, "y", typedAction.Y,
			"scroll_x", typedAction.ScrollX, "scroll_y", typedAction.ScrollY)

		return scroll(ctx, desktop, typedAction.X, typedAction.Y, typedAction.ScrollX, typedAction.ScrollY)
	case responses.ComputerActionType:
		slog.InfoContext(ctx, "typing", "text", typedAction.Text)

		return desktoppkg.TypeText(ctx, desktop, typedAction.Text)
	case responses.ComputerActionWait:
		slog.DebugContext(ctx, "waiting", "duration", waitDuration)

		select {
		case <-time.After(waitDuration):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	default:
		slog.WarnContext(ctx, "ignoring unsupported action", "type", action.Type)

		return nil
	}
}

func screenshot(
	ctx context.Context,
	desktop desktoppkg.Desktop,
) (responses.ResponseComputerToolCallOutputScreenshotParam, error) {
	image, err := desktop.Screen(ctx)
	if err != nil {
		return responses.ResponseComputerToolCallOutputScreenshotParam{},
			fmt.Errorf("failed to capture screen: %w", err)
	}

	var pngBuffer bytes.Buffer

	if err := png.Encode(&pngBuffer, image); err != nil {
		return responses.ResponseComputerToolCallOutputScreenshotParam{}, err
	}

	return responses.ResponseComputerToolCallOutputScreenshotParam{
		ImageURL: openai.String("data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBuffer.Bytes())),
	}, nil
}

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

func drag(ctx context.Context, desktop desktoppkg.Desktop, path []responses.ComputerActionDragPath) error {
	slog.DebugContext(ctx, "dragging")

	if len(path) < 2 {
		return fmt.Errorf("drag path is too short, expected at least 2 elements, got %d", len(path))
	}

	// Start dragging
	start := path[0]

	if err := mouse(ctx, desktop, vnc.ButtonLeft, start.X, start.Y); err != nil {
		return err
	}

	// Continue dragging
	for _, point := range path[1:] {
		if err := mouse(ctx, desktop, vnc.ButtonLeft, point.X, point.Y); err != nil {
			return err
		}
	}

	// Stop dragging
	end := path[len(path)-1]

	return mouse(ctx, desktop, 0, end.X, end.Y)
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
