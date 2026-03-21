package prompt

import (
	"context"

	desktoppkg "github.com/cirruslabs/mtell/internal/desktop"
)

type Provider interface {
	Prompt(ctx context.Context, desktop desktoppkg.Desktop, text string) error
}
