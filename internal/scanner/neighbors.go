package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

func BuildNeighbors(fileLines map[string][]string, repoRoot, relPath string, startLine, endLine, before, after int) []model.Neighbor {
	if before == 0 && after == 0 {
		return nil
	}
	if before > 30 {
		before = 30
	}
	if after > 30 {
		after = 30
	}

	full := filepath.Join(repoRoot, relPath)
	lines, ok := fileLines[full]
	if !ok {
		// lazy read if not present
		b, err := os.ReadFile(full)
		if err != nil {
			return nil
		}
		txt := normalizeNewlines(string(b))
		lines = strings.Split(txt, "\n")
	}

	var out []model.Neighbor
	if before > 0 {
		s := max(1, startLine-before)
		e := startLine - 1
		if e >= s && (e-s+1) <= 30 {
			snip := strings.Join(lines[s-1:e], "\n")
			if strings.TrimSpace(snip) != "" {
				out = append(out, model.Neighbor{Path: relPath, StartLine: s, EndLine: e, Code: snip})
			}
		}
	}
	if after > 0 {
		s := endLine + 1
		e := min(len(lines), endLine+after)
		if e >= s && (e-s+1) <= 30 {
			snip := strings.Join(lines[s-1:e], "\n")
			if strings.TrimSpace(snip) != "" {
				out = append(out, model.Neighbor{Path: relPath, StartLine: s, EndLine: e, Code: snip})
			}
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
