#!/usr/bin/env bash
# Builds Windows + Linux binaries. Runs on Linux (WSL) only.
set -euo pipefail
cd "$(dirname "$0")"

# Tailwind standalone CLI (single binary, no Node). Auto-downloaded on first
# run and cached in tools/ (gitignored), so a fresh clone just works.
TAILWIND_VERSION="v4.3.0"
TAILWIND_BIN="./tools/tailwindcss-${TAILWIND_VERSION}"
if [ ! -x "$TAILWIND_BIN" ]; then
    echo "Downloading tailwindcss ${TAILWIND_VERSION}..."
    mkdir -p tools
    curl -fsSLo "$TAILWIND_BIN" \
        "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-x64"
    chmod +x "$TAILWIND_BIN"
fi

templ generate

# Compile CSS into static/, which embed.go bakes into the binary
"$TAILWIND_BIN" -i tailwind.input.css -o static/css/tailwind.css --minify

env GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(cat VERSION)" -o ./bin/jiraworklog.exe ./cmd/jiraworklog
env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(cat VERSION)" -o ./bin/jiraworklog ./cmd/jiraworklog
