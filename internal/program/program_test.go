package program_test

import (
	"testing"
	"time"

	"github.com/cirruslabs/mtell/internal/keymap"
	programpkg "github.com/cirruslabs/mtell/internal/program"
	"github.com/stretchr/testify/require"
)

func TestWaitDurationImplicit(t *testing.T) {
	program, err := programpkg.ParseString("<wait10>")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.WaitDuration{}, statement)

	statementTyped := statement.(*programpkg.WaitDuration)
	require.Equal(t, 10*time.Second, statementTyped.Duration)
}

func TestWaitDurationExplicit(t *testing.T) {
	program, err := programpkg.ParseString("<wait5m15s>")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.WaitDuration{}, statement)

	statementTyped := statement.(*programpkg.WaitDuration)
	require.Equal(t, 5*time.Minute+15*time.Second, statementTyped.Duration)
}

func TestWaitText(t *testing.T) {
	program, err := programpkg.ParseString("<wait 'Hello, World!'>")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.WaitText{}, statement)

	statementTyped := statement.(*programpkg.WaitText)
	require.Equal(t, "Hello, World!", statementTyped.Text)
}

func TestClick(t *testing.T) {
	program, err := programpkg.ParseString("<click 'Hello, World!'>")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.ClickText{}, statement)

	statementTyped := statement.(*programpkg.ClickText)
	require.Equal(t, "Hello, World!", statementTyped.Text)
}

func TestKeyPress(t *testing.T) {
	program, err := programpkg.ParseString("<leftAlt>")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.PressKey{}, statement)

	statementTyped := statement.(*programpkg.PressKey)
	require.Equal(t, uint32(keymap.KeyMap["leftAlt"]), statementTyped.Code)
	require.Equal(t, programpkg.KeyOn|programpkg.KeyOff, statementTyped.Mask)
}

func TestKeyOn(t *testing.T) {
	program, err := programpkg.ParseString("<leftAltOn>")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.PressKey{}, statement)

	statementTyped := statement.(*programpkg.PressKey)
	require.Equal(t, uint32(keymap.KeyMap["leftAlt"]), statementTyped.Code)
	require.Equal(t, programpkg.KeyOn, statementTyped.Mask)
}

func TestKeyOff(t *testing.T) {
	program, err := programpkg.ParseString("<leftAltOff>")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.PressKey{}, statement)

	statementTyped := statement.(*programpkg.PressKey)
	require.Equal(t, uint32(keymap.KeyMap["leftAlt"]), statementTyped.Code)
	require.Equal(t, programpkg.KeyOff, statementTyped.Mask)
}

func TestType(t *testing.T) {
	program, err := programpkg.ParseString("Just random text.")
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	statement := program.Statements[0]
	require.IsType(t, &programpkg.TypeText{}, statement)

	statementTyped := statement.(*programpkg.TypeText)
	require.Equal(t, "Just random text.", statementTyped.Text)
}
