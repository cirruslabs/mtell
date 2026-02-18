#!/bin/bash

# Fail fast
set -euo pipefail

# Start X session
export DISPLAY=:99

Xvfb "${DISPLAY}" -screen 0 1024x768x24 &

# Wait for the X session to be active
for i in $(seq 1 100); do
    if xdpyinfo -display "${DISPLAY}" >/dev/null 2>&1; then
        break
    fi

    sleep 1
done

# Start VNC server
exec x11vnc \
  -display "${DISPLAY}" \
  -rfbport 5900 \
  -forever \
  -shared \
  -nopw \
  -xkb \
  -cursor none \
  &

# Start GUI program
./gui
