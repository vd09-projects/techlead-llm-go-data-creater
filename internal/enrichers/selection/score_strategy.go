package selection

import (
	"go/ast"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/core"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/utils"
	"golang.org/x/tools/go/packages"
)

var specialAPINames = []string{"New", "With", "Sugar", "Desugar", "Named", "WithOptions"}

type Strategy interface {
	ClassifyReason(path string, fn *core.FunctionNode) string
	Score(path string, fn *core.FunctionNode) float64
	Visibility(path string, fn *core.FunctionNode) string
}

type DefaultStrategy struct {
	repoRoot    string
	nameToFanin map[string]int
}

func (ds *DefaultStrategy) precompute() {
	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedCompiledGoFiles | packages.NeedName,
		Dir:  ds.repoRoot,
	}
	pkgs, _ := packages.Load(cfg, "./...")
	nameToFanin := ds.countFanIn(pkgs)
	ds.nameToFanin = nameToFanin
}

// CountFanIn returns a map of simple function name -> number of call sites across pkgs.
func (ds *DefaultStrategy) countFanIn(pkgs []*packages.Package) map[string]int {
	out := make(map[string]int)
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fun := call.Fun.(type) {
				case *ast.Ident:
					if fun.Name != "" {
						out[fun.Name]++
					}
				case *ast.SelectorExpr:
					if fun.Sel != nil && fun.Sel.Name != "" {
						out[fun.Sel.Name]++
					}
				}
				return true
			})
		}
	}
	// light normalization: ignore obviously unhelpful names
	for k := range out {
		if ds.isKeywordish(k) {
			delete(out, k)
		}
	}
	return out
}

func (ds *DefaultStrategy) isKeywordish(s string) bool {
	s = strings.ToLower(s)
	switch s {
	case "new", "make", "len", "cap", "append", "copy", "delete", "close", "panic", "recover":
		return true
	default:
		return false
	}
}

func (ds *DefaultStrategy) Visibility(path string, fn *core.FunctionNode) string {
	exported := len(fn.Name) > 0 && fn.Name[0] >= 'A' && fn.Name[0] <= 'Z'
	return utils.If(exported, "exported").Else("unexported")
}

func (ds *DefaultStrategy) ClassifyReason(path string, fn *core.FunctionNode) string {
	base := strings.ToLower(filepath.Base(path))
	_ = base
	if strings.HasPrefix(fn.Name, "New") {
		return "constructor"
	}
	for _, p := range specialAPINames {
		if strings.HasPrefix(fn.Name, p) {
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
	if strings.Contains(strings.ToLower(path), "core") {
		return "core"
	}
	return "other"
}

func (ds *DefaultStrategy) Score(path string, fn *core.FunctionNode) float64 {
	name := fn.Name
	exported := len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
	lineCount := fn.EndLine - fn.StartLine + 1
	faninNorm := utils.Min(1.0, float64(ds.nameToFanin[name])/50.0)
	isTest := strings.HasSuffix(strings.ToLower(path), "_test.go")

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
	return utils.RoundN(utils.Clamp01(score), 2)
}

func NewDefaultStrategy(repoRoot string) Strategy {
	ds := &DefaultStrategy{
		repoRoot: repoRoot,
	}
	ds.precompute()
	return ds
}
