package document

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

type PPTXLoader struct{}

type pptxSlide struct {
	CommonData pptxCSld `xml:"cSld"`
}

type pptxCSld struct {
	SpTree pptxSpTree `xml:"spTree"`
}

type pptxSpTree struct {
	Shapes []pptxSp `xml:"sp"`
}

type pptxSp struct {
	TxBody pptxTxBody `xml:"txBody"`
}

type pptxTxBody struct {
	Paragraphs []pptxParagraph `xml:"p"`
}

type pptxParagraph struct {
	Runs []pptxRun `xml:"r"`
}

type pptxRun struct {
	Text string `xml:"t"`
}

func (l *PPTXLoader) Load(filePath string) ([]Document, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	defer r.Close()

	var slideFiles []*zip.File
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideFiles = append(slideFiles, f)
		}
	}

	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].Name < slideFiles[j].Name
	})

	if len(slideFiles) == 0 {
		return nil, fmt.Errorf("no slides found in %s", filePath)
	}

	var allText []string
	for i, sf := range slideFiles {
		text, err := extractSlideText(sf)
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) != "" {
			allText = append(allText, fmt.Sprintf("[Slide %d]\n%s", i+1, text))
		}
	}

	content := strings.Join(allText, "\n\n")
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("no text extracted from %s", filePath)
	}

	return []Document{{
		Content:  content,
		Metadata: map[string]string{"source": filePath},
	}}, nil
}

func extractSlideText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	var slide pptxSlide
	if err := xml.Unmarshal(data, &slide); err != nil {
		return "", err
	}

	var lines []string
	for _, sp := range slide.CommonData.SpTree.Shapes {
		for _, p := range sp.TxBody.Paragraphs {
			var parts []string
			for _, run := range p.Runs {
				if run.Text != "" {
					parts = append(parts, run.Text)
				}
			}
			if text := strings.Join(parts, ""); text != "" {
				lines = append(lines, text)
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

func (l *PPTXLoader) Extensions() []string {
	return []string{".pptx"}
}
