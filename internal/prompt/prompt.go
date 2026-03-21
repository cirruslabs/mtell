package prompt

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	desktoppkg "github.com/cirruslabs/mtell/internal/desktop"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

const (
	openaiModel = openai.ChatModelGPT5_4

	computerInstructions = "You are controlling a desktop over VNC. " +
		"Use the screenshot as the source of truth. Before typing, pressing Enter, " +
		"or using keyboard shortcuts, make sure the intended element or window is focused."
)

type OpenAIProvider struct{}

func (p *OpenAIProvider) Prompt(ctx context.Context, desktop desktoppkg.Desktop, text string) error {
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
		slog.DebugContext(ctx, "calling LLM", "model", openaiModel)

		params := responses.ResponseNewParams{
			Instructions: openai.String(computerInstructions),
			Model:        openaiModel,
			Reasoning: responses.ReasoningParam{
				Effort: shared.ReasoningEffortXhigh,
			},
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
				if err := handleOpenAIAction(ctx, desktop, action); err != nil {
					return err
				}
			}

			screenshotURL, err := screenshotDataURL(ctx, desktop)
			if err != nil {
				return err
			}

			input = append(input, responses.ResponseInputItemParamOfComputerCallOutput(
				computerCall.CallID,
				responses.ResponseComputerToolCallOutputScreenshotParam{
					ImageURL: openai.String(screenshotURL),
				},
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

func handleOpenAIAction(
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

		points := make([]point, len(typedAction.Path))
		for i, p := range typedAction.Path {
			points[i] = point{X: p.X, Y: p.Y}
		}

		return drag(ctx, desktop, points)
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
