package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const dataDirName = ".iimq"
const dataFileName = "tasks.json"

var ErrCorruptData = errors.New("stored tasks are corrupted")

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, dataDirName, dataFileName)
	return &Store{path: path}, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() ([]byte, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte("[]"), nil
		}
		return nil, err
	}
	return b, nil
}

func (s *Store) Save(data []byte) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func DecodeTasks(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return ErrCorruptData
	}
	return nil
}

func EncodeTasks(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
