package scanner

import (
	"math"
	"path/filepath"
	"regexp"
	"strings"
)

var specialAPINames = []string{"New", "With", "Sugar", "Desugar", "Named", "WithOptions"}

func ClassifyReason(name, path string, fallback bool) string {
	if fallback {
		return "fallback_parser"
	}
	base := strings.ToLower(filepath.Base(path))
	_ = base
	if strings.HasPrefix(name, "New") {
		return "constructor"
	}
	for _, p := range []string{"With", "Sugar", "Desugar", "Named", "WithOptions"} {
		if strings.HasPrefix(name, p) {
			return "public_api"
		}
	}
	pl := strings.ToLower(path)
	if strings.Contains(pl, "encoder") {
		return "encoder"
	}
	if regexp.MustCompile(`\bcore\b`).MatchString(pl) {
		return "core"
	}
	if strings.Contains(pl, "sampling") {
		return "sampling"
	}
	return "other"
}

func ComputeScore(name string, exported bool, path string, lineCount int, isGenerated bool, faninNorm float64, isTest bool) float64 {
	score := 0.40
	if exported {
		score += 0.20
	}
	for _, p := range specialAPINames {
		if strings.HasPrefix(name, p) {
			score += 0.15
			break
		}
	}
	if lineCount >= 5 && lineCount <= 80 {
		score += 0.10
	}
	pl := strings.ToLower(path)
	if strings.Contains(pl, "encoder") || regexp.MustCompile(`\bcore\b`).MatchString(pl) || strings.Contains(pl, "sampling") {
		score += 0.05
	}
	if faninNorm > 0 {
		if faninNorm > 1 {
			faninNorm = 1
		}
		score += 0.20 * faninNorm
	}
	if isTest {
		score -= 0.25
	}
	if isGenerated {
		score -= 0.50
	}
	return clamp01(score)
}

// func isExported(name string) bool {
// 	if name == "" {
// 		return false
// 	}
// 	r := rune(name[0])
// 	return r >= 'A' && r <= 'Z'
// }

func normalizeVisibilityPtr(exported bool) *string {
	v := "unexported"
	if exported {
		v = "exported"
	}
	return &v
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
