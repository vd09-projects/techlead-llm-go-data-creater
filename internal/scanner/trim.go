package scanner

import "strings"

// TrimFunctionCode preserves signature + opening brace,
// then keeps as many top-level body lines (depth==1) as possible,
// adds "// ... trimmed ..." and closes with "}".
func TrimFunctionCode(full string, maxLines int) (trimmed string, outLines int) {
	full = normalizeNewlines(full)
	lines := strings.Split(full, "\n")
	if len(lines) <= maxLines {
		return full, len(lines)
	}

	text := strings.Join(lines, "\n")
	braceIdx := strings.Index(text, "{")
	if braceIdx < 0 {
		// weird; just hard cut
		out := strings.Join(lines[:maxLines], "\n")
		return out, maxLines
	}

	sig := text[:braceIdx+1]
	body := text[braceIdx+1:]

	sigLines := strings.Split(sig, "\n")
	maxBodyLines := maxLines - len(sigLines) - 1 // reserve 1 for closing brace
	if maxBodyLines <= 0 {
		out := strings.Join(append(sigLines, "  // ... trimmed ...", "}"), "\n")
		return out, len(sigLines) + 2
	}

	// scan body with basic state machine
	state := "code" // code | sl | ml | dq | sq | raw
	depth := 1
	i := 0
	curLines := 0
	cutIdx := 0

	for i < len(body) && curLines < maxBodyLines-1 /*reserve for comment*/ {
		ch := body[i]
		next := byte(0)
		if i+1 < len(body) {
			next = body[i+1]
		}
		switch state {
		case "code":
			if ch == '"' && !isEscaped(body, i) {
				state = "dq"
			} else if ch == '\'' && !isEscaped(body, i) {
				state = "sq"
			} else if ch == '`' {
				state = "raw"
			} else if ch == '/' && next == '/' {
				state = "sl"
				i++
			} else if ch == '/' && next == '*' {
				state = "ml"
				i++
			} else if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
			}
		case "sl":
			if ch == '\n' {
				state = "code"
			}
		case "ml":
			if ch == '*' && next == '/' {
				state = "code"
				i++
			}
		case "dq":
			if ch == '"' && !isEscaped(body, i) {
				state = "code"
			}
		case "sq":
			if ch == '\'' && !isEscaped(body, i) {
				state = "code"
			}
		case "raw":
			if ch == '`' {
				state = "code"
			}
		}
		if ch == '\n' {
			curLines++
			if depth == 1 {
				cutIdx = i + 1
			}
		}
		i++
	}

	kept := strings.TrimRight(body[:cutIdx], "\n")
	outLines = len(sigLines)
	var result []string
	result = append(result, sigLines...)
	if kept != "" {
		result = append(result, kept)
	}
	result = append(result, "  // ... trimmed ...", "}")
	return strings.Join(result, "\n"), outLines + lineCount(kept) + 2
}

func isEscaped(s string, i int) bool {
	back := 0
	j := i - 1
	for j >= 0 && s[j] == '\\' {
		back++
		j--
	}
	return back%2 == 1
}
