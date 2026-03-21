package prompt

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	desktoppkg "github.com/cirruslabs/mtell/internal/desktop"
	"github.com/cenkalti/backoff/v4"
	"google.golang.org/genai"
)

const (
	geminiModel = "gemini-2.5-computer-use-preview-10-2025"

	geminiInstructions = "You are controlling a desktop over VNC. " +
		"Use the screenshot as the source of truth. Before typing, pressing Enter, " +
		"or using keyboard shortcuts, make sure the intended element or window is focused."

	// Gemini uses a normalized 1000x1000 coordinate grid (0-999).
	geminiCoordMax = 999
)

type GeminiProvider struct{}

func (p *GeminiProvider) Prompt(ctx context.Context, desktop desktoppkg.Desktop, text string) error {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{
			{
				ComputerUse: &genai.ComputerUse{
					Environment: genai.EnvironmentBrowser,
				},
			},
		},
		SystemInstruction: genai.NewContentFromText(geminiInstructions, genai.RoleUser),
	}

	// Take initial screenshot to include with the prompt
	pngBytes, err := screenshotPNG(ctx, desktop)
	if err != nil {
		return fmt.Errorf("failed to capture initial screenshot: %w", err)
	}

	contents := []*genai.Content{
		{
			Role: genai.RoleUser,
			Parts: []*genai.Part{
				genai.NewPartFromText(text),
				genai.NewPartFromBytes(pngBytes, "image/png"),
			},
		},
	}

	for {
		slog.DebugContext(ctx, "calling LLM", "model", geminiModel)

		response, err := geminiGenerateWithRetry(ctx, client, contents, config)
		if err != nil {
			return fmt.Errorf("failed to call Gemini: %w", err)
		}

		functionCalls := response.FunctionCalls()
		if len(functionCalls) == 0 {
			slog.DebugContext(ctx, "no function calls returned, looks like we're done")

			if text := response.Text(); text != "" {
				slog.InfoContext(ctx, "returned an answer to a prompt", "text", text)
			}

			return nil
		}

		// Get screen dimensions for coordinate denormalization
		screenImage, err := desktop.Screen(ctx)
		if err != nil {
			return fmt.Errorf("failed to capture screen for dimensions: %w", err)
		}
		screenWidth := int64(screenImage.Bounds().Dx())
		screenHeight := int64(screenImage.Bounds().Dy())

		// Add the model's response to the conversation
		if len(response.Candidates) > 0 && response.Candidates[0].Content != nil {
			contents = append(contents, response.Candidates[0].Content)
		}

		for _, fc := range functionCalls {
			if err := handleGeminiAction(ctx, desktop, fc, screenWidth, screenHeight); err != nil {
				return err
			}

			// Take screenshot after each action
			pngBytes, err := screenshotPNG(ctx, desktop)
			if err != nil {
				return err
			}

			// Send function response with screenshot.
			// Gemini computer use requires a "current_url" field in the response.
			contents = append(contents, &genai.Content{
				Role: genai.RoleUser,
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							Name: fc.Name,
							Response: map[string]any{
								"status":      "completed",
								"current_url": "about:blank",
							},
						},
					},
					genai.NewPartFromBytes(pngBytes, "image/png"),
				},
			})
		}
	}
}

// geminiGenerateWithRetry calls GenerateContent with exponential backoff retry
// on transient errors (429, 503, 500).
func geminiGenerateWithRetry(
	ctx context.Context,
	client *genai.Client,
	contents []*genai.Content,
	config *genai.GenerateContentConfig,
) (*genai.GenerateContentResponse, error) {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 2 * time.Second
	b.MaxInterval = 30 * time.Second
	b.MaxElapsedTime = 2 * time.Minute

	var response *genai.GenerateContentResponse

	err := backoff.Retry(func() error {
		var err error

		response, err = client.Models.GenerateContent(ctx, geminiModel, contents, config)
		if err != nil {
			errMsg := err.Error()

			// Retry on transient server errors
			if strings.Contains(errMsg, "503") ||
				strings.Contains(errMsg, "500") ||
				strings.Contains(errMsg, "429") ||
				strings.Contains(errMsg, "UNAVAILABLE") ||
				strings.Contains(errMsg, "RESOURCE_EXHAUSTED") ||
				strings.Contains(errMsg, "INTERNAL") {
				slog.WarnContext(ctx, "transient Gemini error, retrying", "error", err)

				return err
			}

			// Non-retryable error
			return backoff.Permanent(err)
		}

		return nil
	}, backoff.WithContext(b, ctx))

	return response, err
}

// denormalize converts Gemini's normalized 0-999 coordinate to actual pixel coordinate.
func denormalize(normalized float64, screenDimension int64) int64 {
	return int64(normalized / geminiCoordMax * float64(screenDimension-1))
}

func handleGeminiAction(
	ctx context.Context,
	desktop desktoppkg.Desktop,
	fc *genai.FunctionCall,
	screenWidth int64,
	screenHeight int64,
) error {
	args := fc.Args

	switch fc.Name {
	case "click_at":
		x := denormalize(getFloat(args, "x"), screenWidth)
		y := denormalize(getFloat(args, "y"), screenHeight)

		slog.InfoContext(ctx, "clicking", "x", x, "y", y)

		return click(ctx, desktop, "left", x, y)

	case "hover_at":
		x := denormalize(getFloat(args, "x"), screenWidth)
		y := denormalize(getFloat(args, "y"), screenHeight)

		slog.InfoContext(ctx, "hovering", "x", x, "y", y)

		return mouse(ctx, desktop, 0, x, y)

	case "type_text_at":
		x := denormalize(getFloat(args, "x"), screenWidth)
		y := denormalize(getFloat(args, "y"), screenHeight)
		text := getString(args, "text")

		slog.InfoContext(ctx, "typing at", "x", x, "y", y, "text", text)

		// Click at position first to focus
		if err := click(ctx, desktop, "left", x, y); err != nil {
			return err
		}

		// Clear existing text if requested
		if getBool(args, "clear_before_typing") {
			if err := keypress(ctx, desktop, []string{"control", "a"}); err != nil {
				return err
			}
		}

		// Type the text
		if err := desktoppkg.TypeText(ctx, desktop, text); err != nil {
			return err
		}

		// Press enter if requested
		if getBool(args, "press_enter") {
			return keypress(ctx, desktop, []string{"enter"})
		}

		return nil

	case "key_combination":
		keysStr := getString(args, "keys")
		keys := strings.Split(keysStr, "+")

		slog.InfoContext(ctx, "key combination", "keys", keys)

		return keypress(ctx, desktop, keys)

	case "scroll_at":
		x := denormalize(getFloat(args, "x"), screenWidth)
		y := denormalize(getFloat(args, "y"), screenHeight)
		direction := getString(args, "direction")
		magnitude := getFloat(args, "magnitude")

		if magnitude == 0 {
			magnitude = 100 // default scroll amount in pixels
		}

		slog.InfoContext(ctx, "scrolling", "x", x, "y", y, "direction", direction, "magnitude", magnitude)

		var scrollX, scrollY int64

		switch strings.ToLower(direction) {
		case "up":
			scrollY = -int64(magnitude)
		case "down":
			scrollY = int64(magnitude)
		case "left":
			scrollX = -int64(magnitude)
		case "right":
			scrollX = int64(magnitude)
		}

		return scroll(ctx, desktop, x, y, scrollX, scrollY)

	case "scroll_document":
		direction := getString(args, "direction")
		magnitude := getFloat(args, "magnitude")

		if magnitude == 0 {
			magnitude = 300
		}

		slog.InfoContext(ctx, "scrolling document", "direction", direction, "magnitude", magnitude)

		// Scroll at center of screen
		centerX := screenWidth / 2
		centerY := screenHeight / 2

		var scrollX, scrollY int64

		switch strings.ToLower(direction) {
		case "up":
			scrollY = -int64(magnitude)
		case "down":
			scrollY = int64(magnitude)
		case "left":
			scrollX = -int64(magnitude)
		case "right":
			scrollX = int64(magnitude)
		}

		return scroll(ctx, desktop, centerX, centerY, scrollX, scrollY)

	case "drag_and_drop":
		x := denormalize(getFloat(args, "x"), screenWidth)
		y := denormalize(getFloat(args, "y"), screenHeight)
		destX := denormalize(getFloat(args, "destination_x"), screenWidth)
		destY := denormalize(getFloat(args, "destination_y"), screenHeight)

		slog.InfoContext(ctx, "dragging", "from_x", x, "from_y", y, "to_x", destX, "to_y", destY)

		return drag(ctx, desktop, []point{
			{X: x, Y: y},
			{X: destX, Y: destY},
		})

	case "wait_5_seconds":
		slog.InfoContext(ctx, "waiting 5 seconds")

		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}

	case "open_web_browser", "navigate", "go_back", "go_forward", "search":
		// These are browser-specific actions that don't map directly to VNC.
		// Log and skip them.
		slog.WarnContext(ctx, "ignoring browser-specific action", "name", fc.Name, "args", args)

		return nil

	default:
		slog.WarnContext(ctx, "ignoring unsupported Gemini action", "name", fc.Name)

		return nil
	}
}

// Helper functions to extract typed values from the args map.

func getFloat(args map[string]any, key string) float64 {
	v, ok := args[key]
	if !ok {
		return 0
	}

	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}

func getString(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}

	s, ok := v.(string)
	if !ok {
		return ""
	}

	return s
}

func getBool(args map[string]any, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}

	b, ok := v.(bool)
	if !ok {
		return false
	}

	return b
}
