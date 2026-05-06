package document

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Document struct {
	Content   string
	Metadata  map[string]string
	ImageData []byte
}

type Loader interface {
	Load(filePath string) ([]Document, error)
	Extensions() []string
}

type Registry struct {
	loaders map[string]Loader
}

func NewRegistry() *Registry {
	return &Registry{loaders: make(map[string]Loader)}
}

func (r *Registry) Register(loader Loader) {
	for _, ext := range loader.Extensions() {
		r.loaders[strings.ToLower(ext)] = loader
	}
}

func (r *Registry) Load(filePath string) ([]Document, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	loader, ok := r.loaders[ext]
	if !ok {
		if isBinaryFile(filePath) {
			return nil, fmt.Errorf("binary file not supported: %s", filePath)
		}
		loader, ok = r.loaders[".txt"]
		if !ok {
			return nil, fmt.Errorf("text loader not registered")
		}
	}
	return loader.Load(filePath)
}

func (r *Registry) CanLoad(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	if _, ok := r.loaders[ext]; ok {
		return true
	}
	return !isBinaryFile(filePath)
}

func (r *Registry) SupportedExtensions() []string {
	seen := make(map[string]bool)
	var exts []string
	for ext := range r.loaders {
		if !seen[ext] {
			seen[ext] = true
			exts = append(exts, ext)
		}
	}
	return exts
}

func isBinaryFile(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return true
	}
	buf = buf[:n]
	if len(buf) == 0 {
		return false
	}
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}
	return !utf8.Valid(buf)
}
