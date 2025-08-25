package scanner

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/utils"
)

var genCodeRe = regexp.MustCompile(`(?i)^\s*//\s*Code generated`)
var testFileRe = regexp.MustCompile(`_test\.go$`)

type FuncInfo struct {
	Path       string // rel posix
	Name       string
	Recv       string // (T) or (*T) or (pkg.T) as text
	Signature  string // "func ..." up to '{'
	StartLine  int
	EndLine    int
	Code       string // trimmed
	IsTestFile bool
}

func ExtractFunctions(u FileUnit, maxFuncLines, minFuncLines int) (out []FuncInfo) {
	// Skip generated (first 5 lines)
	headLines := strings.Split(u.Src, "\n")
	head := strings.Join(headLines[:utils.Min(5, len(headLines))], "\n")
	if genCodeRe.MatchString(head) {
		return nil
	}

	ast.Inspect(u.File, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name == nil {
			return true
		}
		name := fd.Name.Name

		recv := ""
		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			recvType := strings.TrimSpace(sliceByPos(u.Src, u.Fset, fd.Recv.List[0].Type.Pos(), fd.Recv.List[0].Type.End()))
			recv = "(" + recvType + ")"
		}

		signEnd := fd.End()
		if fd.Body != nil {
			signEnd = fd.Body.Lbrace
		}
		signature := strings.TrimSpace(sliceByPos(u.Src, u.Fset, fd.Pos(), signEnd))

		start := u.Fset.PositionFor(fd.Pos(), true).Line
		// endPos := fd.End()
		// end := u.Fset.PositionFor(endPos, true).Line
		code := sliceByPos(u.Src, u.Fset, fd.Pos(), fd.End())

		trimmed, lines := TrimFunctionCode(code, maxFuncLines)
		if lineCount(trimmed) < minFuncLines {
			return true
		}

		out = append(out, FuncInfo{
			Path:       u.RelPath,
			Name:       name,
			Recv:       recv,
			Signature:  signature,
			StartLine:  start,
			EndLine:    start + lines - 1,
			Code:       ensureTrailingNL(trimmed),
			IsTestFile: testFileRe.MatchString(u.RelPath),
		})
		return true
	})
	return out
}

func sliceByPos(src string, fset *token.FileSet, start, end token.Pos) string {
	p0 := fset.PositionFor(start, true).Offset
	p1 := fset.PositionFor(end, true).Offset
	if p0 < 0 || p1 > len(src) || p0 > p1 {
		return ""
	}
	return src[p0:p1]
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}
func ensureTrailingNL(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
