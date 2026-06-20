package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/httpx"
)

//go:embed dist
var distFS embed.FS

func Handler() http.Handler {
	assets, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		requestPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestPath == "." || requestPath == "" {
			serveIndex(w, r, assets)
			return
		}

		file, err := assets.Open(requestPath)
		if err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		serveIndex(w, r, assets)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, assets fs.FS) {
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		httpx.WriteErrorKey(w, r, http.StatusInternalServerError, "frontend_index_missing")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, strings.NewReader(string(index)))
}
