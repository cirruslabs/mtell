package vnc

import (
	"context"
	"image"
	colorpkg "image/color"
	"log/slog"
	"net"
	"net/url"
	"reflect"
	"time"

	"github.com/mitchellh/go-vnc"
	"golang.org/x/sync/singleflight"
)

type Desktop struct {
	client          *vnc.ClientConn
	serverMessageCh chan vnc.ServerMessage
	image           *image.RGBA
	imageCh         chan image.Image
	singleflight    singleflight.Group
	inputDelay      time.Duration
}

func New(ctx context.Context, target string, inputDelay time.Duration) (*Desktop, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	dialer := net.Dialer{}

	netConn, err := dialer.DialContext(ctx, "tcp", targetURL.Host)
	if err != nil {
		return nil, err
	}

	serverMessageCh := make(chan vnc.ServerMessage)

	clientConfig := &vnc.ClientConfig{
		ServerMessageCh: serverMessageCh,
	}

	if password, ok := targetURL.User.Password(); ok {
		clientConfig.Auth = []vnc.ClientAuth{
			&DHAuth{
				Username: targetURL.User.Username(),
				Password: password,
			},
			&vnc.PasswordAuth{
				Password: password,
			},
		}
	}

	client, err := vnc.Client(netConn, clientConfig)
	if err != nil {
		return nil, err
	}

	err = client.SetEncodings([]vnc.Encoding{
		&vnc.RawEncoding{},
		&DesktopSizePseudoEncoding{},
	})
	if err != nil {
		return nil, err
	}

	desktop := &Desktop{
		client:          client,
		serverMessageCh: serverMessageCh,
		imageCh:         make(chan image.Image),
		inputDelay:      inputDelay,
	}

	// Now that we've connected, initialize the image with the server-provided screen dimensions
	desktop.imageDimensionsChanged()

	return desktop, nil
}

func (desktop *Desktop) Run(ctx context.Context) error {
	for {
		select {
		case serverMessage := <-desktop.serverMessageCh:
			switch message := serverMessage.(type) {
			case *vnc.FramebufferUpdateMessage:
				slog.DebugContext(ctx, "received framebuffer update message",
					"num_rectangles", len(message.Rectangles))

				if err := desktop.handleFramebufferUpdateMessage(ctx, message); err != nil {
					return err
				}
			default:
				slog.DebugContext(ctx, "ignoring unsupported server message",
					"type", reflect.TypeOf(message))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (desktop *Desktop) handleFramebufferUpdateMessage(
	ctx context.Context,
	message *vnc.FramebufferUpdateMessage,
) error {
	for _, rectangle := range message.Rectangles {
		switch enc := rectangle.Enc.(type) {
		case *vnc.RawEncoding:
			for i, color := range enc.Colors {
				x, y := i%int(rectangle.Width), i/int(rectangle.Width)
				r, g, b := uint8(color.R), uint8(color.G), uint8(color.B)

				desktop.image.Set(int(rectangle.X)+x, int(rectangle.Y)+y, colorpkg.RGBA{
					R: r,
					G: g,
					B: b,
					A: 255,
				})
			}

			// Propagate message to reader(s)
			select {
			case desktop.imageCh <- desktop.image:
				// Continue
			default:
				// No one wants an image
			}
		case *DesktopSizePseudoEncoding:
			desktop.imageDimensionsChanged()
		default:
			slog.DebugContext(ctx, "ignoring unsupported framebuffer update rectangle encoding",
				"type", reflect.TypeOf(enc))
		}
	}

	return nil
}

func (desktop *Desktop) Mouse(ctx context.Context, mask vnc.ButtonMask, x uint16, y uint16) error {
	if err := desktop.client.PointerEvent(mask, x, y); err != nil {
		return err
	}

	return desktop.introduceInputDelay(ctx)
}

func (desktop *Desktop) Keyboard(ctx context.Context, keysum uint32, down bool) error {
	if err := desktop.client.KeyEvent(keysum, down); err != nil {
		return err
	}

	return desktop.introduceInputDelay(ctx)
}

func (desktop *Desktop) Screen(ctx context.Context) (image.Image, error) {
	// Avoid multiple outgoing framebuffer update requests
	resultCh := desktop.singleflight.DoChan("screen", func() (interface{}, error) {
		// Tell the VNC server that we need fresh framebuffer contents
		if err := desktop.client.FramebufferUpdateRequest(false, 0, 0,
			desktop.client.FrameBufferWidth, desktop.client.FrameBufferHeight); err != nil {
			return nil, err
		}

		// Wait for the response
		select {
		case image := <-desktop.imageCh:
			return image, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}

		return result.Val.(image.Image), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (desktop *Desktop) Close() error {
	return desktop.client.Close()
}

func (desktop *Desktop) imageDimensionsChanged() {
	w, h := int(desktop.client.FrameBufferWidth), int(desktop.client.FrameBufferHeight)
	desktop.image = image.NewRGBA(image.Rect(0, 0, w, h))
}

func (desktop *Desktop) introduceInputDelay(ctx context.Context) error {
	select {
	case <-time.After(desktop.inputDelay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
