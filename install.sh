#!/bin/bash
set -e

# Build the binary
GOCACHE="$(pwd)/tmp/gocache" go build -o shortcut-git .

# Copy to /usr/bin
sudo cp shortcut-git /usr/bin/

echo "shortcut-git installed to /usr/bin/"