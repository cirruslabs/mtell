# mtell

`mtell` CLI allows you to tell a machine to do something over VNC. It's indispensable when the actions to be performed cannot be done over a traditional SSH connection, which is a frequent necessity when automating macOS machines.

Besides allowing you to script basic keyboard and mouse interactions, `mtell` can also utilize computer vision to wait for and click certain elements on the screen.

This project is heavily inspired by [Packer's `boot_command`](https://developer.hashicorp.com/packer/integrations/cirruslabs/tart/latest/components/builder/tart#boot-configuration), but extends its command set and allows you to run these commands in any environment where you can start a binary.

Special thanks to [Tor Arne Vestbø](https://github.com/torarnv), who contributed the initial [`<wait 'text'>` implementation](https://github.com/cirruslabs/packer-plugin-tart/pull/178) to [Packer builder for Tart VMs](https://github.com/cirruslabs/packer-plugin-tart), which made us realize that we can further extend the `boot_command` and do pretty cool things with it, for example, locating and clicking buttons using computer vision.

## Installation

### Using Homebrew

```shell
brew install cirruslabs/cli/mtell
```

### Using Golang

```shell
go install github.com/cirruslabs/mtell@latest
```

## Usage

```shell
mtell --vnc "vnc://:password@localhost:5900" "<wait10s><click 'Select Your Country or Region'>"
```

## Reference

### Waiting

There commands are useful to work around loading screens.

* `<wait10>` — wait 10 seconds
* `<wait5m15s>` — wait 5 minutes and 15 seconds
* `<wait 'Choose Your Country'>` — using computer vision, wait for the pattern (can be a regular expression) to appear on screen

### Keyboard

Introduce the following commands into your program to press a corresponding key:

* `<bs>`, `<del>`, `<enter>`, `<return>`, `<esc>`, `<tab>`, `<spacebar>` — editing keys
* `<insert>`, `<home>`, `<end>`, `<pageUp>`, `<pageDown>` — navigation keys
* `<up>`, `<down>`, `<left>`, `<right>` — arrow keys
* `<f1>`–`<f12>` — <kbd>F1</kbd>–<kbd>F12</kbd> keys
* `<menu>` — context menu key
* `<leftAlt>`, `<rightAlt>` — <kbd>Alt</kbd> key
* `<leftCtrl>`, `<rightCtrl>` — <kbd>Control</kbd> key
* `<leftShift>`, `<rightShift>` — <kbd>Shift</kbd> key
* `<leftSuper>`, `<rightSuper>` — <kbd>Super</kbd> key
* `<leftCommand>`, `<rightCommand>` — <kbd>⌘</kbd> key on macOS
* `<leftOption>`, `<rightOption>` — <kbd>⌥</kbd> key on macOS

Any keyboard command can be modified with `On` or `Off` modifier, for example:

* `<leftShift>` —  presses <kbd>Shift</kbd> key and releases it
* `<leftShiftOn>` — presses <kbd>Shift</kbd> key without releasing it
* `<leftShiftOff>` — releases <kbd>Shift</kbd> key

### Mouse

These commands allow one to utilize a mouse, currently only through computer vision:

* `<click 'Accept'>` — using computer vision, wait for the pattern (can be a regular expression) to appear on screen and click in the center of its bounding box
