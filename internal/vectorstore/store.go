package vectorstore

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Entry struct {
	ID         string            `json:"id"`
	DocumentID string            `json:"document_id"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	Embedding  []float32         `json:"embedding"`
}

type SearchResult struct {
	Entry
	Score float32 `json:"score"`
}

type DocumentInfo struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	FilePath   string    `json:"file_path"`
	Size       int64     `json:"size"`
	ChunkCount int       `json:"chunk_count"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Store struct {
	mu        sync.RWMutex
	entries   []Entry
	documents map[string]DocumentInfo
	dimension int
}

func New(dimension int) *Store {
	return &Store{
		entries:   make([]Entry, 0),
		documents: make(map[string]DocumentInfo),
		dimension: dimension,
	}
}

func (s *Store) Add(entries []Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entries...)
}

func (s *Store) Search(query []float32, topK int) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 {
		return nil
	}

	return searchEntries(query, topK, s.entries)
}

func (s *Store) HybridSearch(query []float32, queryText string, topK int) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 {
		return nil
	}

	return hybridSearchEntries(query, queryText, topK, s.entries)
}

func (s *Store) HybridSearchByDocumentIDs(query []float32, queryText string, topK int, documentIDs []string) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 || len(documentIDs) == 0 {
		return nil
	}

	allowed := make(map[string]bool, len(documentIDs))
	for _, id := range documentIDs {
		if id != "" {
			allowed[id] = true
		}
	}
	if len(allowed) == 0 {
		return nil
	}

	entries := make([]Entry, 0)
	for _, entry := range s.entries {
		if allowed[entry.DocumentID] {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil
	}

	return hybridSearchEntries(query, queryText, topK, entries)
}

func hybridSearchEntries(query []float32, queryText string, topK int, entries []Entry) []SearchResult {
	if len(entries) == 0 {
		return nil
	}

	queryTerms := tokenize(queryText)
	if len(queryTerms) == 0 {
		return searchEntries(query, topK, entries)
	}

	normalizedQuery := NormalizeVector(query)
	docTokens := make([][]string, len(entries))
	docFreq := make(map[string]int)
	totalDocLength := 0

	for i, entry := range entries {
		tokens := tokenize(entry.Content)
		docTokens[i] = tokens
		totalDocLength += len(tokens)

		seen := make(map[string]bool)
		for _, token := range tokens {
			if !seen[token] {
				seen[token] = true
				docFreq[token]++
			}
		}
	}

	avgDocLength := float64(totalDocLength) / float64(len(entries))
	if avgDocLength <= 0 {
		avgDocLength = 1
	}

	results := make([]SearchResult, len(entries))
	vectorScores := make([]float64, len(entries))
	bm25Scores := make([]float64, len(entries))
	var maxBM25 float64

	for i, entry := range entries {
		vectorScore := float64(DotProduct(normalizedQuery, NormalizeVector(entry.Embedding)))
		vectorScores[i] = vectorScore

		bm25Score := bm25(queryTerms, docTokens[i], docFreq, len(entries), avgDocLength)
		bm25Scores[i] = bm25Score
		if bm25Score > maxBM25 {
			maxBM25 = bm25Score
		}
	}

	for i, entry := range entries {
		vectorPart := (vectorScores[i] + 1) / 2
		if vectorPart < 0 {
			vectorPart = 0
		}

		bm25Part := 0.0
		if maxBM25 > 0 {
			bm25Part = bm25Scores[i] / maxBM25
		}

		combined := 0.65*vectorPart + 0.35*bm25Part
		results[i] = SearchResult{
			Entry: entry,
			Score: float32(combined),
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

func searchEntries(query []float32, topK int, entries []Entry) []SearchResult {
	if len(entries) == 0 {
		return nil
	}

	normalized := NormalizeVector(query)
	results := make([]SearchResult, len(entries))
	for i, e := range entries {
		results[i] = SearchResult{
			Entry: e,
			Score: DotProduct(normalized, NormalizeVector(e.Embedding)),
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

func bm25(queryTerms []string, docTokens []string, docFreq map[string]int, docCount int, avgDocLength float64) float64 {
	const k1 = 1.5
	const b = 0.75

	if len(docTokens) == 0 || docCount == 0 {
		return 0
	}

	termFreq := make(map[string]int)
	for _, token := range docTokens {
		termFreq[token]++
	}

	docLength := float64(len(docTokens))
	score := 0.0
	for _, term := range uniqueTerms(queryTerms) {
		tf := float64(termFreq[term])
		if tf == 0 {
			continue
		}

		df := float64(docFreq[term])
		idf := math.Log(1 + (float64(docCount)-df+0.5)/(df+0.5))
		denominator := tf + k1*(1-b+b*(docLength/avgDocLength))
		score += idf * ((tf * (k1 + 1)) / denominator)
	}
	return score
}

func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
	})
}

func uniqueTerms(terms []string) []string {
	seen := make(map[string]bool)
	unique := make([]string, 0, len(terms))
	for _, term := range terms {
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		unique = append(unique, term)
	}
	return unique
}

func (s *Store) RemoveByDocumentID(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if e.DocumentID != docID {
			filtered = append(filtered, e)
		}
	}
	s.entries = filtered
}

func (s *Store) AddDocument(info DocumentInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[info.ID] = info
}

func (s *Store) RemoveDocument(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.documents, docID)
}

func (s *Store) ListDocuments() []DocumentInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	docs := make([]DocumentInfo, 0, len(s.documents))
	for _, d := range s.documents {
		docs = append(docs, d)
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UploadedAt.After(docs[j].UploadedAt)
	})
	return docs
}

func (s *Store) EntryCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

func (s *Store) DocumentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make([]Entry, 0)
	s.documents = make(map[string]DocumentInfo)
}

func (s *Store) Dimension() int {
	return s.dimension
}
