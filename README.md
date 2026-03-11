# mtell

`mtell` is a CLI for driving a machine over VNC. It is useful when the task cannot be completed over SSH alone, which is a common case when automating GUI-heavy macOS workflows.

An `mtell` program can mix plain text typing with structured commands for waiting, pressing keys, clicking text found on screen, and delegating more complex flows to OpenAI's [Computer use](https://developers.openai.com/api/docs/guides/tools-computer-use).

Here is a quick demo using a local [Tart](https://github.com/cirruslabs/tart) VM:

https://github.com/user-attachments/assets/e91c6501-5347-4cf8-9b56-75b6be4a88a7

## Requirements

* A reachable VNC server, for example `example.com:5900` or `vnc://:password@example.com:5900`
* macOS on the machine running `mtell` for OCR-based commands such as `<wait '...'>` and `<click '...'>`
* `OPENAI_API_KEY` if you want to use `<prompt '...'>`

Current limitations:

* OCR-backed commands rely on Apple Vision, so they are currently macOS-only

## Installation

### Using Homebrew

```shell
brew install cirruslabs/cli/mtell
```

### Using Go

```shell
go install github.com/cirruslabs/mtell@latest
```

## Quickstart

The CLI takes a single `PROGRAM` argument:

```shell
mtell --vnc "vnc://:password@localhost:5900" PROGRAM
```

A program is just text plus angle-bracket commands:

* Plain text is typed literally
* Commands such as `<enter>` or `<wait10s>` are executed in place
* OCR-based commands use single-quoted patterns and support regular expressions

Examples:

```shell
# Type credentials and submit
mtell --vnc "vnc://:password@localhost:5900" "admin<tab>s3cret<enter>"

# Wait for a screen to appear, then click a button by visible text
mtell --vnc "vnc://:password@localhost:5900" \
  "<wait30s><click 'Select Your Country or Region'>"

# Use a regular expression to wait for text on screen
mtell --vnc "vnc://:password@localhost:5900" \
  "<wait 'FileVault( Disk)? Encryption'><click 'Continue'>"

# Let OpenAI drive the UI for a more complex task
OPENAI_API_KEY=... mtell --vnc "vnc://:password@localhost:5900" \
  "<prompt 'Accept the dialog and close the currently active window.'>"
```

Useful flags:

* `--input-delay 250ms` adjusts the delay between input actions
* `--debug` enables verbose logs
* `--version` prints the version

## Reference

### Typing

Any text outside `<...>` is typed literally:

* `hello world`
* `user@example.com<tab>hunter2<enter>`

### Waiting

These commands are useful for loading screens and synchronization:

* `<wait10>` waits 10 seconds
* `<wait5m15s>` waits 5 minutes and 15 seconds
* `<wait 'Choose Your Country'>` waits until text matching the pattern appears on screen

### Mouse

These commands use OCR to locate text on screen:

* `<click 'Accept'>` waits for the pattern to appear, then clicks the center of its bounding box

### Keyboard

Use the following commands to press keys:

* `<bs>`, `<del>`, `<enter>`, `<return>`, `<esc>`, `<tab>`, `<spacebar>` for editing
* `<insert>`, `<home>`, `<end>`, `<pageUp>`, `<pageDown>` for navigation
* `<up>`, `<down>`, `<left>`, `<right>` for arrow keys
* `<f1>`-`<f12>` for function keys
* `<menu>` for the context menu key
* `<leftAlt>`, `<rightAlt>` for <kbd>Alt</kbd>
* `<leftCtrl>`, `<rightCtrl>` for <kbd>Control</kbd>
* `<leftShift>`, `<rightShift>` for <kbd>Shift</kbd>
* `<leftSuper>`, `<rightSuper>` for <kbd>Super</kbd>
* `<leftCommand>`, `<rightCommand>` for <kbd>Command</kbd> on macOS
* `<leftOption>`, `<rightOption>` for <kbd>Option</kbd> on macOS

Any keyboard command can be modified with `On` or `Off`:

* `<leftShift>` presses and releases <kbd>Shift</kbd>
* `<leftShiftOn>` presses <kbd>Shift</kbd> without releasing it
* `<leftShiftOff>` releases <kbd>Shift</kbd>

### Computer use

These commands are powered by OpenAI's [Computer use](https://developers.openai.com/api/docs/guides/tools-computer-use):

* `<prompt 'Open Safari and dismiss any first-run dialogs.'>` operates the UI using natural language

## Background

This project is heavily inspired by [Packer's `boot_command`](https://developer.hashicorp.com/packer/integrations/cirruslabs/tart/latest/components/builder/tart#boot-configuration), but extends its command set and lets you run those commands anywhere you can start a binary.

Special thanks to [Tor Arne Vestbø](https://github.com/torarnv), who contributed the initial [`<wait 'text'>` implementation](https://github.com/cirruslabs/packer-plugin-tart/pull/178) to [Packer builder for Tart VMs](https://github.com/cirruslabs/packer-plugin-tart). That work made it clear that `boot_command` could be pushed further with screen text recognition and higher-level UI automation.
