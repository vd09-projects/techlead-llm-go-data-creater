package callgraph

import (
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ssa"
)

func ParseInputSymbol(s string) (recv string, name string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") {
		if i := strings.Index(s, ")."); i > 1 {
			inside := strings.TrimSpace(s[1:i])
			rest := strings.TrimSpace(s[i+2:])
			if rest != "" {
				return inside, rest
			}
		}
	}
	if dot := strings.LastIndexByte(s, '.'); dot >= 0 && !strings.HasPrefix(s, "(") {
		return "", strings.TrimSpace(s[dot+1:])
	}
	return "", s
}

func displayName(fn *ssa.Function) string {
	if fn == nil {
		return ""
	}
	if sig := fn.Signature; sig != nil && sig.Recv() != nil {
		return "(" + recvString(sig.Recv().Type()) + ")." + fn.Name()
	}
	return fn.Name()
}

func recvString(t types.Type) string {
	switch tt := t.(type) {
	case *types.Pointer:
		return "*" + recvString(tt.Elem())
	case *types.Named:
		if obj := tt.Obj(); obj != nil {
			return obj.Name()
		}
		return tt.String()
	default:
		return types.TypeString(t, func(p *types.Package) string { return p.Name() })
	}
}

func fileFor(fset *token.FileSet, fn *ssa.Function) string {
	if fn == nil {
		return ""
	}
	pos := fn.Pos()
	if pos == token.NoPos {
		return ""
	}
	return filepath.ToSlash(fset.PositionFor(pos, true).Filename)
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(r)
}