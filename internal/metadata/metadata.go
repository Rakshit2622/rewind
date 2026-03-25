package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Version represents a single snapshot entry for a tracked file.
type Version struct {
	ID        string    `json:"id"`
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	IsDelta   bool      `json:"is_delta"`
}

// FileMeta holds the complete version history for a tracked file.
type FileMeta struct {
	FilePath string    `json:"file_path"`
	Versions []Version `json:"versions"`
}

// rewindDir returns the path to the global ~/.rewind directory.
func rewindDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".rewind")
}

// path returns the metadata file path for a given tracked file.
// The absolute file path is hashed to produce a unique directory name,
// preventing collisions between files with the same basename.
func path(file string) string {
	abs, err := filepath.Abs(file)
	if err != nil {
		abs = file
	}
	name := sha256.Sum256([]byte(abs))
	return filepath.Join(rewindDir(), "files", hex.EncodeToString(name[:]), "metadata.json")
}

// ValidateFile checks that the given path exists and is a regular file,
// returning an error if it does not exist or is a directory.
func ValidateFile(file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("%s: file not found", file)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", file)
	}
	return nil
}

// Load reads and parses the metadata for the given file path.
func Load(filePath string) (*FileMeta, error) {
	data, err := os.ReadFile(path(filePath))
	if err != nil {
		return &FileMeta{}, fmt.Errorf("%s is not tracked", filePath)
	}
	var meta FileMeta
	if err = json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}
	return &meta, nil
}

// Save serialises meta and writes it to the metadata file for filePath.
func Save(filePath string, meta *FileMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling metadata: %w", err)
	}
	return os.WriteFile(path(filePath), data, 0644)
}

// Init creates a fresh metadata entry for a file that is not yet tracked.
// If the file is already tracked it returns the existing metadata instead.
func Init(filePath string) (*FileMeta, error) {
	if Exists(filePath) {
		return Load(filePath)
	}
	p := path(filePath)
	err := os.MkdirAll(filepath.Dir(p), 0755)
	if err != nil {
		return &FileMeta{}, fmt.Errorf("error creating metadata directory: %w", err)
	}
	meta := FileMeta{
		FilePath: filePath,
		Versions: []Version{},
	}
	if err = Save(filePath, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// Exists reports whether the given file is already tracked by Rewind.
func Exists(filePath string) bool {
	_, err := os.Stat(path(filePath))
	return err == nil
}
