package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"rewind/internal/diff"
	"rewind/internal/metadata"
	"rewind/internal/storage"
	"rewind/pkg/hash"
)

// hashBytes returns the SHA256 hex digest of the given byte slice.
func hashBytes(data []byte) string {
	str, _ := hash.Compute(data)
	return str
}

// Reconstruct rebuilds the file content at versionID by loading the latest
// full snapshot and applying reverse deltas backwards through the chain.
func Reconstruct(meta *metadata.FileMeta, versionID string) ([]byte, error) {
	if len(meta.Versions) == 0 {
		return nil, fmt.Errorf("no versions found")
	}
	latest := meta.Versions[len(meta.Versions)-1]
	content, err := storage.Load(latest.Hash)
	if err != nil {
		return nil, err
	}
	for i := len(meta.Versions) - 1; i >= 0; i-- {
		v := meta.Versions[i]
		if v.IsDelta {
			patch, err := storage.Load(v.Hash)
			if err != nil {
				return nil, err
			}
			result, err := diff.Apply(string(content), string(patch))
			if err != nil {
				return nil, err
			}
			content = []byte(result)
		}
		if v.ID == versionID {
			return content, nil
		}
	}
	return nil, fmt.Errorf("version not found")
}

// Track registers a file with Rewind by creating its initial metadata entry.
// No snapshot is saved at this stage.
func Track(filePath string) (*metadata.FileMeta, error) {
	if err := metadata.ValidateFile(filePath); err != nil {
		return nil, err
	}
	return metadata.Init(filePath)
}

// Save snapshots the current state of filePath with the given message.
// The first save stores the full file content. Each subsequent save demotes
// the previous latest version to a reverse delta and stores the new content
// as the full base, keeping the latest version always directly readable.
func Save(filePath string, message string) error {
	if err := metadata.ValidateFile(filePath); err != nil {
		return err
	}
	newBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	newContent := string(newBytes)

	meta, err := metadata.Load(filePath)
	if err != nil {
		meta = &metadata.FileMeta{
			FilePath: filePath,
		}
	}

	if meta == nil || len(meta.Versions) == 0 {
		hash := hashBytes(newBytes)
		if err := storage.Store(hash, newBytes); err != nil {
			return err
		}
		meta = &metadata.FileMeta{
			FilePath: filePath,
			Versions: []metadata.Version{
				{
					ID:        "v1",
					Hash:      hash,
					Message:   message,
					Timestamp: time.Now(),
					IsDelta:   false,
				},
			},
		}
		return metadata.Save(filePath, meta)
	}

	prev := meta.Versions[len(meta.Versions)-1]
	oldHash := prev.Hash

	prevBytes, err := storage.Load(oldHash)
	if err != nil {
		return err
	}
	prevContent := string(prevBytes)

	patch := diff.Patch(newContent, prevContent)
	patchBytes := []byte(patch)
	patchHash := hashBytes(patchBytes)

	if err := storage.Store(patchHash, patchBytes); err != nil {
		return err
	}

	prev.IsDelta = true
	prev.Hash = patchHash
	meta.Versions[len(meta.Versions)-1] = prev

	if !isHashReferenced(meta, oldHash) {
		_ = storage.Delete(oldHash)
	}

	newHash := hashBytes(newBytes)
	if err := storage.Store(newHash, newBytes); err != nil {
		return err
	}

	id := fmt.Sprintf("v%d", len(meta.Versions)+1)
	meta.Versions = append(meta.Versions, metadata.Version{
		ID:        id,
		Hash:      newHash,
		Message:   message,
		Timestamp: time.Now(),
		IsDelta:   false,
	})
	return metadata.Save(filePath, meta)
}

// History returns a formatted list of all saved versions for filePath,
// each showing the version ID, timestamp, and commit message.
func History(filePath string) ([]string, error) {
	meta, err := metadata.Load(filePath)
	if err != nil {
		return nil, err
	}
	var hist []string
	for _, v := range meta.Versions {
		str := fmt.Sprintf("%s   %s   %s", v.ID, v.Timestamp.Format("2006-01-02 15:04:05"), v.Message)
		hist = append(hist, str)
	}
	return hist, nil
}

// Diff reads the current file on disk and compares it against the content of
// versionID, returning a human-readable diff string.
func Diff(filePath, versionID string) (string, error) {
	newBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	meta, err := metadata.Load(filePath)
	if err != nil {
		return "", err
	}
	if len(meta.Versions) == 0 {
		return "", fmt.Errorf("no snapshots found for %s", filePath)
	}
	oldBytes, err := Reconstruct(meta, versionID)
	if err != nil {
		return "", err
	}
	header := fmt.Sprintf("--- %s  (%s)\n+++ %s  (current)\n", filepath.Base(filePath), versionID, filepath.Base(filePath))
	final := header + diff.Compute(string(oldBytes), string(newBytes))
	return final, nil
}

// Revert restores filePath to the content of the given versionID.
// If versionID is the latest it is loaded directly; otherwise the content
// is reconstructed by replaying the reverse delta chain.
func Revert(filePath string, versionID string) error {
	meta, err := metadata.Load(filePath)
	if err != nil {
		return err
	}
	if len(meta.Versions) == 0 {
		return fmt.Errorf("no versions found")
	}
	latest := meta.Versions[len(meta.Versions)-1]

	var content []byte

	if latest.ID == versionID {
		content, err = storage.Load(latest.Hash)
		if err != nil {
			return err
		}
	} else {
		content, err = Reconstruct(meta, versionID)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(filePath, content, 0644)
}
