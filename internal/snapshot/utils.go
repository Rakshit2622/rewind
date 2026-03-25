package snapshot

import "rewind/internal/metadata"

// isHashReferenced reports whether the given hash is still referenced by
// any version in meta. Used before deleting an object to avoid orphaning
// versions that share the same content.
func isHashReferenced(meta *metadata.FileMeta, hash string) bool {
	for _, v := range meta.Versions {
		if v.Hash == hash {
			return true
		}
	}
	return false
}
