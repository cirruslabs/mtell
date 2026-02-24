package integration

import (
	"log"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/cirruslabs/mtell/internal/desktop/vnc"
	"github.com/cirruslabs/mtell/internal/executor"
	programpkg "github.com/cirruslabs/mtell/internal/program"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestSimpleVNC(t *testing.T) {
	// Receive extra logs from desktop, executor, etc.
	slog.SetLogLoggerLevel(slog.LevelDebug)

	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context: filepath.Join("testdata", "simple-vnc"),
			},
			ExposedPorts: []string{"5900/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5900/tcp"),
			).WithStartupTimeoutDefault(2 * time.Minute),
		},
		Started: true,
		Logger:  log.Default(),
	})
	require.NoError(t, err)

	endpoint, err := container.Endpoint(t.Context(), "vnc")
	require.NoError(t, err)

	desktop, err := vnc.New(t.Context(), endpoint, 100*time.Millisecond)
	require.NoError(t, err)
	defer desktop.Close()

	go func() {
		_ = desktop.Run(t.Context())
	}()

	program, err := programpkg.ParseString("<wait1><wait1s>A very secret key!<enter>" +
		"<wait 'A very special button'><click 'A very special button'>")
	require.NoError(t, err)

	err = executor.Execute(t.Context(), desktop, program)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		state, err := container.State(t.Context())
		require.NoError(t, err)

		return state.ExitCode == 42
	}, 5*time.Second, 250*time.Millisecond)
}
