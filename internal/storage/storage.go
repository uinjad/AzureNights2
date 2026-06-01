// Package storage is the persistence adapter: a JSON-file implementation of the
// app.Repository port. It is the outermost layer on the save/load side — the app
// depends on the Repository interface, never on this package. Swapping to a
// database later means writing a sibling adapter, not touching the app.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/uinjad/AzureNights2/internal/app"
)

// FileRepo persists a save as a single JSON file on disk.
type FileRepo struct {
	path string
}

// Compile-time proof that FileRepo satisfies the port.
var _ app.Repository = (*FileRepo)(nil)

// NewFileRepo returns a repository backed by the given file path.
func NewFileRepo(path string) *FileRepo {
	return &FileRepo{path: path}
}

// Exists reports whether a save file is present.
func (r *FileRepo) Exists() bool {
	_, err := os.Stat(r.path)
	return err == nil
}

// Load reads and decodes the save file.
func (r *FileRepo) Load() (*app.Snapshot, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("storage: reading save: %w", err)
	}
	var snap app.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("storage: decoding save: %w", err)
	}
	return &snap, nil
}

// Save encodes the snapshot and writes it atomically: the data goes to a temp
// file in the same directory, which is then renamed over the real file. A crash
// mid-write can never leave a corrupt half-save behind.
func (r *FileRepo) Save(snap *app.Snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: encoding save: %w", err)
	}
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("storage: creating save dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".save-*.tmp")
	if err != nil {
		return fmt.Errorf("storage: creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // harmless no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("storage: writing save: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("storage: syncing save: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("storage: closing save: %w", err)
	}
	if err := os.Rename(tmpName, r.path); err != nil {
		return fmt.Errorf("storage: replacing save: %w", err)
	}
	return nil
}
