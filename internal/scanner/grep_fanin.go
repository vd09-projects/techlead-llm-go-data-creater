package scanner

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

// CountFanIn returns a map of simple function name -> number of call sites across pkgs.
func CountFanIn(pkgs []*packages.Package) map[string]int {
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
		if isKeywordish(k) {
			delete(out, k)
		}
	}
	return out
}

func isKeywordish(s string) bool {
	s = strings.ToLower(s)
	switch s {
	case "new", "make", "len", "cap", "append", "copy", "delete", "close", "panic", "recover":
		return true
	default:
		return false
	}
}
