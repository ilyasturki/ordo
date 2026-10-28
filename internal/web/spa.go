package web

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
)

// bundle is the app's browser build, copied in by `just web` and the nix build; a plain `go build`
// finds only placeholder.html.
//
//go:embed all:spa
var bundle embed.FS

type app struct {
	files fs.FS
	index []byte
	// scripts names the inline scripts of index.html for the CSP, each as " 'sha256-…'".
	scripts string
	serve   http.Handler
}

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

func loadApp(bundle fs.FS) (*app, error) {
	files, err := fs.Sub(bundle, "spa")
	if err != nil {
		return nil, err
	}
	a := &app{files: files, serve: http.FileServerFS(files)}
	if a.index, err = fs.ReadFile(files, "index.html"); err != nil {
		if a.index, err = fs.ReadFile(files, "placeholder.html"); err != nil {
			return nil, err
		}
	}
	for _, m := range inlineScript.FindAllSubmatch(a.index, -1) {
		sum := sha256.Sum256(m[1])
		a.scripts += " 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	}
	return a, nil
}

// ServeHTTP serves the bundle's files, and index.html for every other path: the app routes itself.
func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name != "" && name != "index.html" && name != "placeholder.html" {
		if info, err := fs.Stat(a.files, name); err == nil && !info.IsDir() {
			if strings.HasPrefix(name, "_app/immutable/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			a.serve.ServeHTTP(w, r)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(a.index))
}
