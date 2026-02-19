package desktop

import (
	"context"
	"image"
	"io"

	"github.com/mitchellh/go-vnc"
)

type Desktop interface {
	Run(ctx context.Context) error
	Mouse(ctx context.Context, mask vnc.ButtonMask, x uint16, y uint16) error
	Keyboard(ctx context.Context, keysum uint32, down bool) error
	Screen(ctx context.Context) (image.Image, error)

	io.Closer
}
