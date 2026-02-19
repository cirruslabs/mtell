package vision_test

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/cirruslabs/mtell/internal/vision"
	"github.com/stretchr/testify/require"
)

func TestFindTextCoordinates(t *testing.T) {
	imageFile, err := os.Open(filepath.Join("testdata", "select-your-country-or-region.png"))
	require.Nil(t, err)
	defer imageFile.Close()

	image, err := png.Decode(imageFile)
	require.Nil(t, err)

	rectangle, err := vision.FindTextCoordinates(t.Context(), image, "Select")
	require.NoError(t, err)
	require.NotNil(t, rectangle)

	// Ensure that resulting coordinates differ no more than 5 pixels
	// from the coordinates observed during writing this test
	const maxAllowedDelta = 5

	require.InDelta(t, 683, rectangle.Min.X, maxAllowedDelta)
	require.InDelta(t, 659, rectangle.Min.Y, maxAllowedDelta)
	require.InDelta(t, 1175, rectangle.Max.X, maxAllowedDelta)
	require.InDelta(t, 699, rectangle.Max.Y, maxAllowedDelta)
}
