package api

import (
	"embed"
	"io/fs"
)

//go:embed static/* static/**
var apiStaticFS embed.FS

// dashboardFS exposes a sub-filesystem rooted at static/.
var dashboardFS fs.FS = func() fs.FS {
	sub, err := fs.Sub(apiStaticFS, "static")
	if err != nil {
		return apiStaticFS
	}
	return sub
}()
