// Package web embeds the built frontend static files so the Go binary
// serves them without an external file system dependency.
package web

import "embed"

// FS provides access to the compiled frontend assets embedded from the
// static/ directory. It is consumed by the HTTP router to serve SPA files.
//go:embed static
var FS embed.FS
