package desktop

import (
	"context"
	"image"
	"io"
	"strings"
	"unicode"

	"github.com/cirruslabs/mtell/internal/keymap"
	"github.com/mitchellh/go-vnc"
)

type Desktop interface {
	Run(ctx context.Context) error
	Mouse(ctx context.Context, mask vnc.ButtonMask, x uint16, y uint16, opts ...MouseOption) error
	Keyboard(ctx context.Context, keysum uint32, down bool) error
	Screen(ctx context.Context) (image.Image, error)

	io.Closer
}

func TypeText(ctx context.Context, desktop Desktop, text string) error {
	for _, c := range text {
		if err := TypeRune(ctx, desktop, c); err != nil {
			return err
		}
	}

	return nil
}

func TypeRune(ctx context.Context, desktop Desktop, r rune) error {
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
