package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

type desktopAssets struct {
	assets  fs.FS
	apiBase string
}

func (d desktopAssets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "templates/index.html"
	}
	if path == "templates/index.html" || path == "index.html" {
		d.serveIndex(w)
		return
	}

	data, err := fs.ReadFile(d.assets, path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType(path))
	w.Write(data)
}

func (d desktopAssets) serveIndex(w http.ResponseWriter) {
	data, err := fs.ReadFile(d.assets, "templates/index.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	injection := fmt.Sprintf(
		"<script>window.jarvisApiBase = %q;</script>\n",
		d.apiBase,
	)
	html := strings.Replace(string(data), "</head>", injection+"</head>", 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "text/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	default:
		return http.DetectContentType([]byte(path))
	}
}
