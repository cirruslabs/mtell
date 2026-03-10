package desktop

import "time"

type MouseInput struct {
	Delay time.Duration
}

type MouseOption func(input *MouseInput)

func WithMouseDelay(delay time.Duration) MouseOption {
	return func(input *MouseInput) {
		input.Delay = delay
	}
}
