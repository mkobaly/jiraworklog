package jiraworklog

import "embed"

// StaticFS holds the static web assets (css/js) embedded into the binary
// so the deployed executable has no filesystem dependencies.
//
//go:embed static
var StaticFS embed.FS
