package document

import (
	"fmt"
	"os"
)

type ImageLoader struct{}

func (l *ImageLoader) Load(filePath string) ([]Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image file: %s", filePath)
	}

	return []Document{{
		Metadata:  map[string]string{"source": filePath, "type": "image"},
		ImageData: data,
	}}, nil
}

func (l *ImageLoader) Extensions() []string {
	return []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".tiff", ".tif"}
}
