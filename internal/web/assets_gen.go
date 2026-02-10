// Package web provides embedded frontend assets.
package web

import (
	"embed"
	"io/fs"
)

// Embedded assets are generated at build time.
//
//go:embed dist/* dist/assets/*
var embeddedAssets embed.FS

// FS returns the embedded frontend file system.
func FS() (fs.FS, error) {
	return fs.Sub(embeddedAssets, "dist")
}
