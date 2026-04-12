package docs

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed index.html openapi.json
var docsFS embed.FS

func Handler() http.Handler {
	staticFS, err := fs.Sub(docsFS, ".")
	if err != nil {
		panic("docs.Handler failed to create sub filesystem")
	}

	return http.FileServer(http.FS(staticFS))
}
