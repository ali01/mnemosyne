package api

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var staticFiles embed.FS

// StaticFS returns the embedded frontend files, or nil if empty.
func EmbeddedFS() fs.FS {
	entries, err := fs.ReadDir(staticFiles, "static")
	if err != nil || len(entries) == 0 {
		return nil
	}
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil
	}
	return sub
}
