package program

import (
	"strings"
	"time"
)

type KeyMask int

const (
	KeyOn KeyMask = 1 << iota
	KeyOff
)

func (mask KeyMask) String() string {
	var states []string

	if mask&KeyOn != 0 {
		states = append(states, "KeyOn")
	}

	if mask&KeyOff != 0 {
		states = append(states, "KeyOff")
	}

	return strings.Join(states, "|")
}

type Program struct {
	Statements []Statement
}

type Statement interface {
	isStatement()
}

type WaitDuration struct {
	Duration time.Duration
}

func (*WaitDuration) isStatement() {}

type WaitText struct {
	Text string
}

func (*WaitText) isStatement() {}

type ClickText struct {
	Text string
}

func (*ClickText) isStatement() {}

type PressKey struct {
	Name string
	Code uint32
	Mask KeyMask
}

func (*PressKey) isStatement() {}

type TypeText struct {
	Text string
}

func (*TypeText) isStatement() {}

type Prompt struct {
	Text string
}

func (*Prompt) isStatement() {}

func ParseString(s string) (*Program, error) {
	programAny, err := Parse("", []byte(s))
	if err != nil {
		return nil, err
	}

	return programAny.(*Program), nil
}
