package document

import (
	"strings"
)

type Chunk struct {
	Content    string
	DocumentID string
	Metadata   map[string]string
	Index      int
}

type Chunker struct {
	ChunkSize    int
	ChunkOverlap int
	Separators   []string
}

func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	return &Chunker{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Separators:   []string{"\n\n", "\n", " ", ""},
	}
}

func (c *Chunker) SplitDocuments(docs []Document, docID string) []Chunk {
	var chunks []Chunk
	idx := 0
	for _, doc := range docs {
		text := strings.ReplaceAll(doc.Content, "\r\n", "\n")
		parts := c.splitText(text, c.Separators)
		for _, part := range parts {
			meta := make(map[string]string)
			for k, v := range doc.Metadata {
				meta[k] = v
			}
			chunks = append(chunks, Chunk{
				Content:    part,
				DocumentID: docID,
				Metadata:   meta,
				Index:      idx,
			})
			idx++
		}
	}
	return chunks
}

func (c *Chunker) splitText(text string, separators []string) []string {
	if len(text) <= c.ChunkSize {
		if strings.TrimSpace(text) != "" {
			return []string{text}
		}
		return nil
	}

	sep := ""
	for _, s := range separators {
		if s == "" || strings.Contains(text, s) {
			sep = s
			break
		}
	}

	var splits []string
	if sep == "" {
		for i := 0; i < len(text); i += c.ChunkSize {
			end := i + c.ChunkSize
			if end > len(text) {
				end = len(text)
			}
			splits = append(splits, text[i:end])
		}
	} else {
		splits = strings.Split(text, sep)
	}

	return c.mergeSplits(splits, sep)
}

func (c *Chunker) mergeSplits(splits []string, sep string) []string {
	var results []string
	var current []string
	currentLen := 0

	for _, s := range splits {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		sepLen := 0
		if len(current) > 0 {
			sepLen = len(sep)
		}

		if currentLen+len(s)+sepLen > c.ChunkSize && len(current) > 0 {
			merged := strings.Join(current, sep)
			if strings.TrimSpace(merged) != "" {
				results = append(results, merged)
			}

			for currentLen > c.ChunkOverlap || (currentLen+len(s)+sepLen > c.ChunkSize && currentLen > 0) {
				if len(current) == 0 {
					break
				}
				removed := current[0]
				current = current[1:]
				currentLen -= len(removed)
				if len(current) > 0 {
					currentLen -= len(sep)
				}
			}
		}

		current = append(current, s)
		if len(current) == 1 {
			currentLen = len(s)
		} else {
			currentLen += len(s) + len(sep)
		}
	}

	if len(current) > 0 {
		merged := strings.Join(current, sep)
		if strings.TrimSpace(merged) != "" {
			results = append(results, merged)
		}
	}

	return results
}
