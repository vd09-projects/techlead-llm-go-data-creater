package scanner

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

type FileUnit struct {
	Filename string    // absolute path
	RelPath  string    // posix rel path from RepoRoot
	File     *ast.File // parsed AST
	Fset     *token.FileSet
	Src      string // full file text, normalized newlines
}

type SourceReader interface {
	List() ([]FileUnit, error)
}

type GoPackagesReader struct {
	RepoRoot   string
	ExcludeREs []*regexp.Regexp
	Env        []string
	Debug      bool
}

func NewGoPackagesReader(repoRoot string, excludeCSV string, debug bool) *GoPackagesReader {
	return &GoPackagesReader{
		RepoRoot:   repoRoot,
		ExcludeREs: compileExcludeRegexes(excludeCSV),
		Env:        os.Environ(),
		Debug:      debug,
	}
}

func (r *GoPackagesReader) List() ([]FileUnit, error) {
	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedCompiledGoFiles | packages.NeedName,
		Dir:  r.RepoRoot,
		Env:  r.Env,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	var out []FileUnit
	for _, p := range pkgs {
		for i, f := range p.Syntax {
			if f == nil {
				continue
			}
			fn := p.CompiledGoFiles[i]
			rel := relPosix(r.RepoRoot, fn)
			if shouldExclude(rel, r.ExcludeREs) {
				continue
			}
			b, err := os.ReadFile(fn)
			if err != nil {
				continue
			}
			src := normalizeNewlines(string(b))
			out = append(out, FileUnit{
				Filename: fn,
				RelPath:  rel,
				File:     f,
				Fset:     p.Fset,
				Src:      src,
			})
		}
	}
	return out, nil
}

// --- helpers (shared) ---

func compileExcludeRegexes(csv string) []*regexp.Regexp {
	var res []*regexp.Regexp
	for _, p := range splitCSV(csv) {
		if re, err := regexp.Compile(p); err == nil {
			res = append(res, re)
		}
	}
	return res
}

func shouldExclude(rel string, res []*regexp.Regexp) bool {
	for _, r := range res {
		if r.MatchString(rel) {
			return true
		}
	}
	return false
}

func relPosix(root, filename string) string {
	rel, err := filepath.Rel(root, filename)
	if err != nil {
		return toPosix(filename)
	}
	return toPosix(rel)
}

func toPosix(p string) string { return strings.ReplaceAll(p, string(filepath.Separator), "/") }

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
