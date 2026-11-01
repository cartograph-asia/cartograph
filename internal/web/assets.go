package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

// assets are the embedded static files, served under StaticPrefix with a
// content hash in the name (style.css -> style.1a2b3c4d.css), so they can be
// cached for a year.
type assets struct {
	byName   map[string]string // "style.css" -> "/static/style.1a2b3c4d.css"
	byHashed map[string]asset  // "style.1a2b3c4d.css" -> file
}

type asset struct {
	data  []byte
	ctype string
}

func loadAssets(fsys fs.FS) (*assets, error) {
	a := &assets{byName: map[string]string{}, byHashed: map[string]asset{}}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		ext := path.Ext(p)
		hashed := strings.TrimSuffix(p, ext) + "." + hex.EncodeToString(sum[:4]) + ext
		ctype := mime.TypeByExtension(ext)
		if ctype == "" {
			ctype = http.DetectContentType(data)
		}
		a.byName[p] = StaticPrefix + hashed
		a.byHashed[hashed] = asset{data: data, ctype: ctype}
		return nil
	})
	return a, err
}

// url returns the hashed URL of an embedded asset; it panics for an unknown
// name, which the template tests catch.
func (a *assets) url(name string) string {
	u, ok := a.byName[name]
	if !ok {
		panic(fmt.Sprintf("web: unknown asset %q", name))
	}
	return u
}

// size returns the byte size of an embedded asset by its plain name.
func (a *assets) size(name string) int {
	return len(a.byHashed[strings.TrimPrefix(a.url(name), StaticPrefix)].data)
}

// ServeHTTP serves StaticPrefix + hashed name with long cache headers.
func (a *assets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, ok := a.byHashed[strings.TrimPrefix(r.URL.Path, StaticPrefix)]
	if !ok {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	h.Set("Content-Type", f.ctype)
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(f.data))
}
