package document

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type DOCXLoader struct{}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"body>p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}

type docxRun struct {
	Text string `xml:"t"`
}

func (l *DOCXLoader) Load(filePath string) ([]Document, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	var docXML io.ReadCloser
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docXML, err = f.Open()
			if err != nil {
				return nil, fmt.Errorf("open document.xml: %w", err)
			}
			break
		}
	}
	if docXML == nil {
		return nil, fmt.Errorf("no word/document.xml in %s", filePath)
	}
	defer docXML.Close()

	var body docxBody
	if err := xml.NewDecoder(docXML).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode docx xml: %w", err)
	}

	var paragraphs []string
	for _, p := range body.Paragraphs {
		var parts []string
		for _, run := range p.Runs {
			if run.Text != "" {
				parts = append(parts, run.Text)
			}
		}
		if text := strings.Join(parts, ""); text != "" {
			paragraphs = append(paragraphs, text)
		}
	}

	content := strings.Join(paragraphs, "\n")
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("no text extracted from %s", filePath)
	}

	return []Document{{
		Content:  content,
		Metadata: map[string]string{"source": filePath},
	}}, nil
}

func (l *DOCXLoader) Extensions() []string {
	return []string{".docx"}
}
