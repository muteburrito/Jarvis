package document

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"log/slog"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

type PDFLoader struct{}

func (l *PDFLoader) Load(filePath string) ([]Document, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var docs []Document
	numPages := r.NumPage()

	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			text = ""
		}
		text = strings.TrimSpace(text)
		if text != "" {
			docs = append(docs, Document{
				Content: text,
				Metadata: map[string]string{
					"source": filePath,
					"page":   fmt.Sprintf("%d", i),
				},
			})
		}
	}

	images := extractJPEGsFromPDF(filePath)
	if len(images) > 0 {
		slog.Info("extracted images from PDF", "file", filePath, "count", len(images))
		for i, imgData := range images {
			docs = append(docs, Document{
				Metadata: map[string]string{
					"source": filePath,
					"type":   "image",
					"image":  fmt.Sprintf("%d", i+1),
				},
				ImageData: imgData,
			})
		}
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no text or images extracted from %s", filePath)
	}
	return docs, nil
}

func extractJPEGsFromPDF(filePath string) [][]byte {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	jpegStart := []byte{0xFF, 0xD8, 0xFF}
	jpegEnd := []byte{0xFF, 0xD9}
	minImageSize := 2000

	var images [][]byte
	offset := 0

	for offset < len(data)-3 {
		start := bytes.Index(data[offset:], jpegStart)
		if start < 0 {
			break
		}
		start += offset

		end := bytes.Index(data[start+3:], jpegEnd)
		if end < 0 {
			break
		}
		end += start + 3 + len(jpegEnd)

		candidate := data[start:end]
		if len(candidate) >= minImageSize {
			if _, err := jpeg.DecodeConfig(bytes.NewReader(candidate)); err == nil {
				imgCopy := make([]byte, len(candidate))
				copy(imgCopy, candidate)
				images = append(images, imgCopy)
			}
		}

		offset = end
	}

	return images
}

func (l *PDFLoader) Extensions() []string {
	return []string{".pdf"}
}
