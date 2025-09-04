package site

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/**
var staticFS embed.FS

// FS returns an http.FileSystem for the embedded docs site.
func FS() http.FileSystem {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		// Should never happen if generator wrote files correctly.
		// Expose an empty FS on error.
		return http.FS(staticFS)
	}
	return http.FS(sub)
}
