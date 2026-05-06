package document

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

type XLSXLoader struct{}

func (l *XLSXLoader) Load(filePath string) ([]Document, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	var docs []Document
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		var lines []string
		for _, row := range rows {
			lines = append(lines, strings.Join(row, "\t"))
		}

		content := strings.Join(lines, "\n")
		if strings.TrimSpace(content) == "" {
			continue
		}

		docs = append(docs, Document{
			Content: content,
			Metadata: map[string]string{
				"source": filePath,
				"sheet":  sheet,
			},
		})
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no data extracted from %s", filePath)
	}
	return docs, nil
}

func (l *XLSXLoader) Extensions() []string {
	return []string{".xlsx"}
}
