package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("mtell integration test")
	w.Resize(fyne.NewSize(1024, 768))

	btn := widget.NewButton("A very special button", func() {
		os.Exit(42)
	})

	// Center the button in the window
	w.SetContent(container.NewCenter(btn))

	w.ShowAndRun()
}
