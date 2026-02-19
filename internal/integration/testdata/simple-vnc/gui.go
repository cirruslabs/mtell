package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// Create a simple GUI app and a window
	a := app.New()
	w := a.NewWindow("mtell integration test")
	w.Resize(fyne.NewSize(1024, 768))

	// Create a password input and a button
	const secret = "A very secret key!"

	passwordInput := widget.NewPasswordEntry()
	passwordInput.SetPlaceHolder(fmt.Sprintf("Enter %q and press \"Enter\" key", secret))

	btn := widget.NewButton("A very special button", func() {
		os.Exit(42)
	})

	// Hide the button for now and only reveal it upon entering the secret
	btn.Hide()

	passwordInput.OnSubmitted = func(text string) {
		if text == secret {
			btn.Show()
		}
	}

	// Set window content and show it
	w.SetContent(container.NewVBox(
		layout.NewSpacer(),
		passwordInput,
		btn,
		layout.NewSpacer(),
	))

	// Show window, focused on password input
	w.Show()
	w.RequestFocus()
	w.Canvas().Focus(passwordInput)

	// Run the GUI app
	a.Run()
}
