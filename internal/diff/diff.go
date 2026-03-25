package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	added   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	removed = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
)

// Patch produces a reverse delta patch that describes how to transform
// newContent back into oldContent. The result is a URL-encoded patch string
// suitable for storage and later application via Apply.
func Patch(newContent, oldContent string) string {
	dmp := diffmatchpatch.New()
	patches := dmp.PatchMake(newContent, oldContent)
	return dmp.PatchToText(patches)
}

// Apply reconstructs content by applying a stored patch string to base.
// Returns an error if any hunk of the patch fails to apply cleanly.
func Apply(base, patch string) (string, error) {
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch)
	if err != nil {
		return "", err
	}
	result, applied := dmp.PatchApply(patches, base)
	for _, ok := range applied {
		if !ok {
			return "", fmt.Errorf("patch failed to apply cleanly")
		}
	}
	return result, nil
}

// Compute returns a human-readable diff between old and new content.
func Compute(old, new string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(old, new, false)
	dmp.DiffCleanupSemantic(diffs)

	var sb strings.Builder
	oldLine, newLine := 1, 1
	for _, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			for _, l := range lines {
				sb.WriteString("  " + l + "\n")
			}
			oldLine += len(lines)
			newLine += len(lines)
		case diffmatchpatch.DiffDelete:
			for _, l := range lines {
				sb.WriteString(removed.Render("- "+l) + "\n")
			}
			oldLine += len(lines)
		case diffmatchpatch.DiffInsert:
			for _, l := range lines {
				sb.WriteString(added.Render("+ "+l) + "\n")
			}
			newLine += len(lines)
		}
	}
	return sb.String()
}
