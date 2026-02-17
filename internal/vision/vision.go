package vision

import (
	"bytes"
	"errors"
	imagepkg "image"
	"image/png"
	"regexp"
	"unsafe"

	"github.com/progrium/darwinkit/macos/foundation"
	visionpkg "github.com/progrium/darwinkit/macos/vision"
	"github.com/progrium/darwinkit/objc"
)

var ErrNotFound = errors.New("no text matching the given regular expression was found")

func FindTextCoordinates(rgba imagepkg.Image, s string) (*imagepkg.Rectangle, error) {
	re, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}

	var result *imagepkg.Rectangle

	objc.WithAutoreleasePool(func() {
		result, err = findTextCoordinatesInner(rgba, re)
	})

	return result, err
}

func findTextCoordinatesInner(image imagepkg.Image, re *regexp.Regexp) (*imagepkg.Rectangle, error) {
	var imageAsPNG bytes.Buffer
	if err := png.Encode(&imageAsPNG, image); err != nil {
		return nil, err
	}

	imageRequest := visionpkg.NewImageRequestHandlerWithDataOptions(imageAsPNG.Bytes(), nil)

	textRecognizer := visionpkg.NewRecognizeTextRequest()
	textRecognizer.SetRecognitionLevel(visionpkg.RequestTextRecognitionLevelAccurate)

	var foundationErr foundation.Error
	if ok := imageRequest.PerformRequestsError([]visionpkg.IRequest{textRecognizer},
		unsafe.Pointer(&foundationErr)); !ok {
		return nil, foundation.ToGoError(foundationErr)
	}

	for _, observation := range textRecognizer.Results() {
		if !observation.IsKindOfClass(visionpkg.RecognizedTextObservationClass.Class) {
			continue
		}

		recognized := visionpkg.RecognizedTextObservationFrom(observation.Ptr())

		boundingBox := recognized.BoundingBox()

		imageBounds := image.Bounds()
		imageWidth, imageHeight := float64(imageBounds.Dx()), float64(imageBounds.Dy())

		for _, candidate := range recognized.TopCandidates(1) {
			text := candidate.String()

			if !re.MatchString(text) {
				break
			}

			minX := boundingBox.Origin.X
			maxX := boundingBox.Origin.X + boundingBox.Size.Width

			// Flip the axis, because:
			//
			// >OCR results are also reported in normalized coordinates, but with the origin at lower left.
			//
			// https://rethunk.medium.com/coordinate-transforms-in-ios-using-swift-part-1-the-l-triangle-c8204177a7e2
			minY := 1.0 - (boundingBox.Origin.Y + boundingBox.Size.Height)
			maxY := 1.0 - boundingBox.Origin.Y

			return &imagepkg.Rectangle{
				Min: imagepkg.Point{
					X: imageBounds.Min.X + int(minX*imageWidth),
					Y: imageBounds.Min.Y + int(minY*imageHeight),
				},
				Max: imagepkg.Point{
					X: imageBounds.Min.X + int(maxX*imageWidth),
					Y: imageBounds.Min.Y + int(maxY*imageHeight),
				},
			}, nil
		}
	}

	return nil, ErrNotFound
}
