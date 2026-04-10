package web

import (
	"embed"
	"io/fs"
)

//go:embed dist dist/*
var distFS embed.FS

func Dist() fs.FS {
	assets, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return assets
}