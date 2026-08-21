package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDir creates dir and all parents when missing.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// WriteFileAtomic writes data to a temporary file and renames it over path,
// so readers never observe a half-written file.
func WriteFileAtomic(path string, data []byte) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadFile reads a file and returns its bytes.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Digest returns the SHA-256 hex digest of data.
func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SaveJSON writes any value as indented JSON using an atomic rename.
func SaveJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	return WriteFileAtomic(path, data)
}

// LoadJSON reads a JSON file into value; a missing file is not an error.
func LoadJSON(path string, value any) error {
	data, err := ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, value)
}
