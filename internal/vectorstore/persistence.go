package vectorstore

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
)

type persistData struct {
	Entries   []Entry
	Documents map[string]DocumentInfo
	Dimension int
}

func (s *Store) Save(dir string) error {
	s.mu.RLock()
	data := persistData{
		Entries:   s.entries,
		Documents: s.documents,
		Dimension: s.dimension,
	}
	s.mu.RUnlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	target := filepath.Join(dir, "vectorstore.gob")
	tmp := target + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer f.Close()

	if err := gob.NewEncoder(f).Encode(data); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("encode: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func LoadFromDisk(dir string) (*Store, error) {
	path := filepath.Join(dir, "vectorstore.gob")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	var data persistData
	if err := gob.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return &Store{
		entries:   data.Entries,
		documents: data.Documents,
		dimension: data.Dimension,
	}, nil
}
