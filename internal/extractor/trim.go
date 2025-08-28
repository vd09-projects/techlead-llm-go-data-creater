package extractor

import (
	"regexp"
	"strings"
)

// very simple comment removers (regex-based; will NOT respect strings)
var slCommentRe = regexp.MustCompile(`(?m)//[^\n]*`)
var mlCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)

// trimFunctionCodeSimple removes comments, then line-caps.
// If a "{" exists, it preserves the signature (up to and including "{")
// and applies the line-cap to the body portion.
func trimFunctionCode(full string, maxLines int) (string, int) {
	full = normalizeNewlines(full)

	// find opening brace to split signature/body (optional nicety)
	openIdx := strings.Index(full, "{")
	if openIdx == -1 {
		// no body; just remove comments & cap
		clean := removeComments(full)
		return capByLines(clean, maxLines)
	}

	sig := full[:openIdx+1]
	body := full[openIdx+1:]

	// remove comments only from the body (keep signature intact)
	cleanBody := removeComments(body)

	// split into lines
	sigLines := splitKeep(sig)
	bodyLines := splitKeep(cleanBody)

	// how many body lines can we keep?
	remaining := maxLines - len(sigLines)
	if remaining <= 0 {
		out := append([]string{}, sigLines...)
		out = append(out, "// ... trimmed ...")
		outStr := strings.Join(out, "\n")
		return ensureTrailingNL(outStr), lineCount(outStr)
	}

	if len(bodyLines) <= remaining {
		out := append([]string{}, sigLines...)
		out = append(out, bodyLines...)
		outStr := strings.Join(out, "\n")
		return ensureTrailingNL(outStr), lineCount(outStr)
	}

	// keep top remaining lines from body
	out := append([]string{}, sigLines...)
	out = append(out, bodyLines[:remaining]...)
	out = append(out, "// ... trimmed ...")
	outStr := strings.Join(out, "\n")
	return ensureTrailingNL(outStr), lineCount(outStr)
}

func removeComments(s string) string {
	// order matters: strip block comments first, then line comments
	s = mlCommentRe.ReplaceAllString(s, "")
	s = slCommentRe.ReplaceAllString(s, "")
	// trim trailing spaces/newlines from comment removal
	return strings.TrimRight(s, "\n")
}

// splitKeep splits by \n without dropping a trailing final line if empty
func splitKeep(s string) []string {
	// normalize already ensured \n, but we won’t trim right
	return strings.Split(s, "\n")
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
}

func capByLines(s string, maxLines int) (string, int) {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		out := strings.Join(lines, "\n")
		return ensureTrailingNL(out), lineCount(out)
	}
	out := strings.Join(lines[:maxLines], "\n") + "\n// ... trimmed ...\n"
	return out, lineCount(out)
}
