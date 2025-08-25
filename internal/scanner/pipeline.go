package scanner

import (
	"regexp"
	"strings"

	cg "github.com/vd09-projects/techlead-llm-go-data-creater/internal/callgraph"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"golang.org/x/tools/go/packages"
)

type Options struct {
	RepoRoot       string
	ExcludeCSV     string
	IncludePrivate bool
	MaxFuncLines   int
	MinFuncLines   int
	ContextBefore  int
	ContextAfter   int
	Debug          bool
	Fields         map[string]bool
	MaxCallers     int
	MaxCallees     int
}

func ParseFields(csv string) map[string]bool {
	m := make(map[string]bool)
	for _, f := range strings.Split(csv, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			m[f] = true
		}
	}
	return m
}

func Run(opts Options) ([]model.Record, error) {
	reader := NewGoPackagesReader(opts.RepoRoot, opts.ExcludeCSV, opts.Debug)
	files, err := reader.List()
	if err != nil {
		return nil, err
	}

	// prepare fan-in (only if selection wants "selection" scoring)
	var pkgs []*packages.Package
	if opts.Fields["selection"] {
		// reuse go/packages by asking reader again; cheap enough & clear
		cfg := &packages.Config{
			Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedCompiledGoFiles | packages.NeedName,
			Dir:  opts.RepoRoot,
			Env:  reader.Env,
		}
		pkgs, _ = packages.Load(cfg, "./...")
	}

	fanin := map[string]int{}
	if opts.Fields["selection"] {
		fanin = CountFanIn(pkgs)
	}

	records := make([]model.Record, 0, 256)
	isGenerated := false // we skip generated earlier

	coreRe := regexp.MustCompile(`\bcore\b`)

	for _, fu := range files {
		funcs := ExtractFunctions(fu, opts.MaxFuncLines, opts.MinFuncLines)
		if len(funcs) == 0 {
			continue
		}

		// lines for neighbors (only if requested)
		var lines []string
		if opts.Fields["neighbors"] && (opts.ContextBefore > 0 || opts.ContextAfter > 0) {
			lines = strings.Split(fu.Src, "\n")
		}

		for _, fn := range funcs {
			exported := isExported(fn.Name)
			if !opts.IncludePrivate && !exported {
				continue
			}

			rec := model.Record{}

			// always set required minimal fields that downstream code keys on
			if opts.Fields["path"] {
				rec.Path = fn.Path
			}
			if opts.Fields["symbol"] {
				sym := fn.Name
				if fn.Recv != "" {
					sym = fn.Recv + "." + fn.Name
				}
				rec.Symbol = sym
			}
			if opts.Fields["signature"] {
				rec.Signature = strings.TrimSpace(fn.Signature)
			}
			if opts.Fields["start_line"] {
				rec.StartLine = fn.StartLine
			}
			if opts.Fields["end_line"] {
				rec.EndLine = fn.EndLine
			}
			if opts.Fields["code"] {
				rec.Code = fn.Code
			}

			// neighbors
			if opts.Fields["neighbors"] && (opts.ContextBefore > 0 || opts.ContextAfter > 0) {
				rec.Neighbors = BuildNeighborsFromLines(lines, fn.Path, fn.StartLine, fn.EndLine, opts.ContextBefore, opts.ContextAfter)
			}

			// selection (score + reason + visibility)
			if opts.Fields["selection"] {
				lineCount := fn.EndLine - fn.StartLine + 1
				faninNorm := minFloat(1.0, float64(fanin[fn.Name])/50.0)
				reason := ClassifyReason(fn.Name, fn.Path, false)
				if coreRe.MatchString(strings.ToLower(fn.Path)) && reason == "other" {
					reason = "core"
				}
				score := ComputeScore(fn.Name, exported, fn.Path, lineCount, isGenerated, faninNorm, fn.IsTestFile)
				vis := "unexported"
				if exported {
					vis = "exported"
				}
				rec.Selection = &model.Selection{
					Visibility: vis,
					Reason:     reason,
					Score:      round2(score),
				}
			}

			// call_graph (native only) — only if requested via --fields
			if opts.Fields["call_graph"] && fn.Path != "" && rec.Symbol != "" {
				// file path the callgraph wants is the relative path (same as rec.Path)
				res, _ := cg.Build(opts.RepoRoot, fn.Path, rec.Symbol, opts.MaxCallers, opts.MaxCallees)
				if len(res.Callees) > 0 || len(res.Callers) > 0 {
					rec.CallGraph = &model.CallGraph{
						Callees:   convertEdges(res.Callees),
						Callers:   convertEdges(res.Callers),
						Precision: "native",
					}
				} else {
					rec.CallGraph = &model.CallGraph{
						Callees:   nil,
						Callers:   nil,
						Precision: "native",
					}
				}
			}

			records = append(records, rec)
		}
	}

	return records, nil
}

func convertEdges(in []cg.Edge) []model.Edge {
	out := make([]model.Edge, 0, len(in))
	for _, e := range in {
		out = append(out, model.Edge{Symbol: e.Symbol, Path: e.Path})
	}
	return out
}

// neighbors using already split lines
func BuildNeighborsFromLines(lines []string, relPath string, startLine, endLine, before, after int) []model.Neighbor {
	if before == 0 && after == 0 {
		return nil
	}
	if before > 30 {
		before = 30
	}
	if after > 30 {
		after = 30
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

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := rune(name[0])
	return r >= 'A' && r <= 'Z'
}
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

//	func max(a, b int) int {
//		if a > b {
//			return a
//		}
//		return b
//	}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
