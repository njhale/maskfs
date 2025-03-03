package mask

import (
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/njhale/maskfs/pkg/index"
)

// GlobMask is responsible for determining which files and directories are included
type GlobMask struct {
	matcher gitignore.Matcher
}

func (m *GlobMask) Masked(entry *index.Entry) bool {
	if entry == nil {
		// The entry is not valid, mask it
		return true
	}

	// Normalize the path
	parts := strings.Split(entry.FSPath, "/")

	// Check if the path matches the rules
	return !m.matcher.Match(parts, entry.IsDir)
}

// NewGlobMask creates a new GlobMask from a new-line delimited list of rules.
// The rules are processed in the order they are given and the last rule takes precedence.
// Note: GlobMask rules use the same syntax as .gitignore, but instead of selecting files to ignore -- like Git does -- GlobMask uses them to select files to include in the index.
func NewGlobMask(rules string) (*GlobMask, error) {
	var patterns []gitignore.Pattern
	for _, line := range strings.Split(rules, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, gitignore.ParsePattern(line, nil))
	}

	return &GlobMask{
		matcher: gitignore.NewMatcher(patterns),
	}, nil
}
