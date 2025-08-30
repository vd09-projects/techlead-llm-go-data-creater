package backuprough

// // Add at top of file if not present:
// import (
// 	"go/types"
// 	"sort"
// 	"strings"
// 	"unicode"
// )

// // CounterpartMethodsOn finds likely counterparts for methodName on the same receiver.
// // If pairs (explicit map) is non-nil, we use it as a high-priority hint and then
// // also run generic discovery. If pairs is nil, we fully rely on generic discovery.
// func (idx *Index) CounterpartMethodsOn(named *types.Named, methodName string, pairs map[string][]string) []FuncDecl {
// 	if named == nil {
// 		return nil
// 	}
// 	pkgPath, _ := idx.fqnNamed(named)
// 	_, recvName := idx.fqnNamed(named)

// 	// 1) gather all methods on the same receiver
// 	var sameRecv []FuncDecl
// 	for _, fd := range idx.funcDeclsByPkg[pkgPath] {
// 		if fd.RecvType == nil {
// 			continue
// 		}
// 		_, rn := idx.fqnNamed(fd.RecvType)
// 		if rn == recvName {
// 			sameRecv = append(sameRecv, fd)
// 		}
// 	}

// 	methodTokens := splitCamelPreserveAcronyms(methodName)
// 	rules := defaultCounterpartRules()

// 	// 2) score candidates
// 	type scored struct {
// 		fd     FuncDecl
// 		score  float64
// 		reason string
// 	}
// 	var scoredList []scored

// 	// 2a) explicit hints (highest priority if provided)
// 	if pairs != nil {
// 		want := explicitCounterpartsFor(methodName, pairs)
// 		for _, fd := range sameRecv {
// 			if nameInSetOrHasPrefix(fd.Name, want) {
// 				scoredList = append(scoredList, scored{fd: fd, score: 2.0, reason: "explicit"})
// 			}
// 		}
// 	}

// 	// 2b) generic discovery
// 	for _, fd := range sameRecv {
// 		if fd.Name == methodName {
// 			continue
// 		}
// 		s, why := counterpartScore(methodTokens, splitCamelPreserveAcronyms(fd.Name), rules)
// 		if s > 0 {
// 			scoredList = append(scoredList, scored{fd: fd, score: s, reason: why})
// 		}
// 	}

// 	// 3) sort by score desc, then file path and start line for determinism
// 	sort.SliceStable(scoredList, func(i, j int) bool {
// 		if scoredList[i].score == scoredList[j].score {
// 			if scoredList[i].fd.FilePath == scoredList[j].fd.FilePath {
// 				return scoredList[i].fd.StartLine < scoredList[j].fd.StartLine
// 			}
// 			return scoredList[i].fd.FilePath < scoredList[j].fd.FilePath
// 		}
// 		return scoredList[i].score > scoredList[j].score
// 	})

// 	out := make([]FuncDecl, 0, len(scoredList))
// 	for _, sc := range scoredList {
// 		out = append(out, sc.fd)
// 	}
// 	return out
// }

// // ------------------------------ Helpers ------------------------------

// type counterpartRules struct {
// 	Antonyms        map[string][]string // leading verb antonyms
// 	PrefixSwaps     [][2]string         // With<->Without, Enable<->Disable
// 	InvertPrefixes  []string            // Un, De (Marshal/Unmarshal, Code/Decode)
// 	MinSharedSuffix int                 // minimal shared suffix chars after verb/prefix removal
// }

// func defaultCounterpartRules() counterpartRules {
// 	return counterpartRules{
// 		Antonyms: map[string][]string{
// 			"open":      {"close"},
// 			"start":     {"stop", "end"},
// 			"begin":     {"end", "finish"},
// 			"enable":    {"disable"},
// 			"lock":      {"unlock"},
// 			"add":       {"remove", "delete", "del"},
// 			"append":    {"remove", "truncate"},
// 			"push":      {"pop"},
// 			"inc":       {"dec"},
// 			"incr":      {"decr"},
// 			"increase":  {"decrease"},
// 			"show":      {"hide"},
// 			"attach":    {"detach"},
// 			"connect":   {"disconnect"},
// 			"mount":     {"unmount"},
// 			"enter":     {"exit", "leave"},
// 			"up":        {"down"},
// 			"next":      {"prev", "previous"},
// 			"marshal":   {"unmarshal"},
// 			"encode":    {"decode"},
// 			"serialize": {"deserialize"},
// 			"sugar":     {"desugar"},
// 			"with":      {"without"},
// 			"get":       {"set"}, // weaker weight inside scoring
// 		},
// 		PrefixSwaps: [][2]string{
// 			{"with", "without"},
// 			{"enable", "disable"},
// 		},
// 		InvertPrefixes:  []string{"un", "de"},
// 		MinSharedSuffix: 3,
// 	}
// }

// func splitCamelPreserveAcronyms(s string) []string {
// 	if s == "" {
// 		return nil
// 	}
// 	var toks []string
// 	start := 0
// 	runes := []rune(s)
// 	push := func(i int) {
// 		if i > start {
// 			toks = append(toks, strings.ToLower(string(runes[start:i])))
// 			start = i
// 		}
// 	}
// 	for i := 1; i < len(runes); i++ {
// 		prev, cur := runes[i-1], runes[i]
// 		if unicode.IsLower(prev) && unicode.IsUpper(cur) {
// 			push(i)
// 			continue
// 		}
// 		if unicode.IsLetter(prev) && unicode.IsDigit(cur) {
// 			push(i)
// 			continue
// 		}
// 		if unicode.IsDigit(prev) && unicode.IsLetter(cur) {
// 			push(i)
// 			continue
// 		}
// 		// acronym boundary: "HTTPServer" -> "HTTP" | "Server"
// 		if unicode.IsUpper(prev) && unicode.IsUpper(cur) {
// 			if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
// 				push(i)
// 			}
// 		}
// 	}
// 	push(len(runes))
// 	return toks
// }

// func explicitCounterpartsFor(name string, pairs map[string][]string) []string {
// 	var want []string
// 	for a, bs := range pairs {
// 		if strings.HasPrefix(name, a) {
// 			want = append(want, bs...)
// 		}
// 		for _, b := range bs {
// 			if strings.HasPrefix(name, b) {
// 				want = append(want, a)
// 			}
// 		}
// 	}
// 	return dedupStrings(want)
// }

// func nameInSetOrHasPrefix(name string, wants []string) bool {
// 	for _, w := range wants {
// 		if name == w || strings.HasPrefix(name, w) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func dedupStrings(in []string) []string {
// 	seen := map[string]struct{}{}
// 	var out []string
// 	for _, s := range in {
// 		if _, ok := seen[s]; ok {
// 			continue
// 		}
// 		seen[s] = struct{}{}
// 		out = append(out, s)
// 	}
// 	return out
// }

// // counterpartScore returns (score, reason). Score > 0 => likely counterpart.
// func counterpartScore(a, b []string, rules counterpartRules) (float64, string) {
// 	if len(a) == 0 || len(b) == 0 {
// 		return 0, ""
// 	}
// 	if s, why := antonymScore(a, b, rules); s > 0 {
// 		return s, why
// 	}
// 	if s, why := invertPrefixScore(a, b, rules); s > 0 {
// 		return s, why
// 	}
// 	if s, why := prefixSwapScore(a, b, rules); s > 0 {
// 		return s, why
// 	}
// 	return 0, ""
// }

// func antonymScore(a, b []string, rules counterpartRules) (float64, string) {
// 	va, sa := leadingVerb(a)
// 	vb, sb := leadingVerb(b)
// 	if va == "" || vb == "" {
// 		return 0, ""
// 	}
// 	if areAntonyms(va, vb, rules.Antonyms) && sharedSuffixLen(sa, sb) >= rules.MinSharedSuffix {
// 		if (va == "get" && vb == "set") || (va == "set" && vb == "get") {
// 			return 0.6, "get/set+suffix"
// 		}
// 		return 1.0, "antonym+suffix"
// 	}
// 	return 0, ""
// }

// func invertPrefixScore(a, b []string, rules counterpartRules) (float64, string) {
// 	// Marshal vs Unmarshal; Code vs Decode
// 	if sharedSuffixLen(stripInvertPrefix(a, rules), stripInvertPrefix(b, rules)) < rules.MinSharedSuffix {
// 		return 0, ""
// 	}
// 	// If one has invert prefix and the other doesn't (or opposing invert prefixes), it’s a match
// 	if hasInvertPrefix(a, rules) != hasInvertPrefix(b, rules) || opposingInvertPrefixes(a, b, rules) {
// 		return 0.9, "invertprefix+suffix"
// 	}
// 	return 0, ""
// }

// func prefixSwapScore(a, b []string, rules counterpartRules) (float64, string) {
// 	for _, p := range rules.PrefixSwaps {
// 		if (hasLeading(a, p[0]) && hasLeading(b, p[1])) || (hasLeading(a, p[1]) && hasLeading(b, p[0])) {
// 			sa := trimLeading(a, p[0])
// 			sb := trimLeading(b, p[1])
// 			if hasLeading(a, p[1]) {
// 				sa = trimLeading(a, p[1])
// 				sb = trimLeading(b, p[0])
// 			}
// 			if sharedSuffixLen(sa, sb) >= rules.MinSharedSuffix {
// 				return 0.95, "prefixswap+suffix"
// 			}
// 		}
// 	}
// 	return 0, ""
// }

// func leadingVerb(tokens []string) (verb string, rest []string) {
// 	if len(tokens) == 0 {
// 		return "", nil
// 	}
// 	return tokens[0], tokens[1:]
// }

// func areAntonyms(a, b string, m map[string][]string) bool {
// 	a = strings.ToLower(a)
// 	b = strings.ToLower(b)
// 	for _, x := range m[a] {
// 		if b == x {
// 			return true
// 		}
// 	}
// 	for _, x := range m[b] {
// 		if a == x {
// 			return true
// 		}
// 	}
// 	return false
// }

// func hasLeading(tokens []string, lead string) bool {
// 	return len(tokens) > 0 && strings.EqualFold(tokens[0], lead)
// }

// func trimLeading(tokens []string, lead string) []string {
// 	if hasLeading(tokens, lead) {
// 		return tokens[1:]
// 	}
// 	return tokens
// }

// func hasInvertPrefix(tokens []string, rules counterpartRules) bool {
// 	if len(tokens) == 0 {
// 		return false
// 	}
// 	for _, p := range rules.InvertPrefixes {
// 		if strings.HasPrefix(tokens[0], p) && len(tokens[0]) > len(p) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func opposingInvertPrefixes(a, b []string, rules counterpartRules) bool {
// 	if len(a) == 0 || len(b) == 0 {
// 		return false
// 	}
// 	pa := invertPrefix(a[0], rules)
// 	pb := invertPrefix(b[0], rules)
// 	return pa != "" && pb != "" && !strings.EqualFold(pa, pb)
// }

// func invertPrefix(tok string, rules counterpartRules) string {
// 	for _, p := range rules.InvertPrefixes {
// 		if strings.HasPrefix(tok, p) && len(tok) > len(p) {
// 			return p
// 		}
// 	}
// 	return ""
// }

// func stripInvertPrefix(tokens []string, rules counterpartRules) []string {
// 	if len(tokens) == 0 {
// 		return tokens
// 	}
// 	for _, p := range rules.InvertPrefixes {
// 		if strings.HasPrefix(tokens[0], p) && len(tokens[0]) > len(p) {
// 			return append([]string{strings.ToLower(tokens[0][len(p):])}, tokens[1:]...)
// 		}
// 	}
// 	return tokens
// }

// func sharedSuffixLen(a, b []string) int {
// 	sa := strings.Join(a, "")
// 	sb := strings.Join(b, "")
// 	i := len(sa) - 1
// 	j := len(sb) - 1
// 	n := 0
// 	for i >= 0 && j >= 0 && sa[i] == sb[j] {
// 		n++
// 		i--
// 		j--
// 	}
// 	return n
// }
