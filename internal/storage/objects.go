package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// rewindDir returns the path to the global ~/.rewind directory.
func rewindDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".rewind")
}

// path returns the filesystem path for an object identified by hash,
// using the first two characters as a subdirectory prefix.
func path(hash string) string {
	if len(hash) < 2 {
		return filepath.Join(rewindDir(), "objects", hash)
	}
	return filepath.Join(rewindDir(), "objects", hash[:2], hash[2:])
}

// hashBytes returns the SHA256 hex digest of data.
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Delete removes the object file for the given hash. It is a no-op if
// the object does not exist.
func Delete(hash string) error {
	p := path(hash)
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat failed: %w", err)
	}
	return os.Remove(p)
}

// Store writes data into the content-addressable object store under
// ~/.rewind/objects/<prefix>/<hash>. It is a no-op if the object already
// exists, providing automatic deduplication.
func Store(hash string, data []byte) error {
	if Exists(hash) {
		return nil
	}
	p := path(hash)
	dir := filepath.Dir(p)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("error creating object directory: %w", err)
	}
	return os.WriteFile(p, data, 0644)
}

// Load retrieves the raw bytes for the object identified by hash.
func Load(hash string) ([]byte, error) {
	if !Exists(hash) {
		return nil, fmt.Errorf("object not found: %s", hash)
	}
	return os.ReadFile(path(hash))
}

// Exists reports whether an object with the given hash is present in the store.
func Exists(hash string) bool {
	_, err := os.Stat(path(hash))
	return err == nil
}
