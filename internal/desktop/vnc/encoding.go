package vnc

import (
	"io"

	"github.com/mitchellh/go-vnc"
)

type DesktopSizePseudoEncoding struct{}

func (*DesktopSizePseudoEncoding) Read(c *vnc.ClientConn, rect *vnc.Rectangle, r io.Reader) (vnc.Encoding, error) {
	c.FrameBufferWidth = rect.Width
	c.FrameBufferHeight = rect.Height
	return &DesktopSizePseudoEncoding{}, nil
}

func (*DesktopSizePseudoEncoding) Type() int32 {
	return -223 // RFC 6143 7.8.2
}
