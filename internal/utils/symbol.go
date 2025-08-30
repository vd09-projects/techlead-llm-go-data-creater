package utils

import (
	"strings"
	"unicode"
)

func RecvBaseType(recv string) string {
	// "(T)" or "(*T)" -> "T"
	recv = strings.TrimSpace(recv)
	if strings.HasPrefix(recv, "(") && strings.HasSuffix(recv, ")") {
		inner := strings.TrimSpace(recv[1 : len(recv)-1])
		inner = strings.TrimPrefix(inner, "*")
		return inner
	}
	return ""
}

func NormalizeCode(s string) string {
	// normalize newlines, strip trailing spaces, redact obvious URLs
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
		// crude URL redaction
		if strings.Contains(lines[i], "http://") || strings.Contains(lines[i], "https://") {
			lines[i] = "/* redacted */"
		}
	}
	return strings.Join(lines, "\n")
}
