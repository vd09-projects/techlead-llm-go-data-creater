package contextrefs

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type TypeDecl struct {
	PkgPath   string
	FilePath  string
	StartLine int
	EndLine   int
	Name      string // "T"
	IsStruct  bool
}

type IfaceMethod struct {
	Name      string
	StartLine int
	EndLine   int
	Obj       *types.Func
}

type InterfaceDecl struct {
	PkgPath   string
	FilePath  string
	StartLine int
	EndLine   int
	Name      string // "I"
	Methods   []IfaceMethod
}

type FuncDecl struct {
	PkgPath   string
	FilePath  string
	StartLine int
	EndLine   int
	Name      string // "NewT" or "Marshal"
	Obj       *types.Func
	RecvType  *types.Named // nil for functions
	Results   []*types.Named
}

type Index struct {
	repoRoot string
	fset     *token.FileSet
	pkgs     []*packages.Package

	// package path -> decls
	typeDeclsByPkg  map[string][]TypeDecl
	ifaceDeclsByPkg map[string][]InterfaceDecl
	funcDeclsByPkg  map[string][]FuncDecl

	// file path (rel) -> decls
	funcDeclsByFile map[string][]FuncDecl

	// fast lookup
	typeDeclByPkgAndName map[string]map[string]TypeDecl // pkg -> name -> decl
}

// ------------------------------ Public entrypoint ------------------------------

// Load builds a semantic index for all packages under repoRoot (./...).
func Load(repoRoot string) (*Index, error) {
	pkgs, err := loadPackages(repoRoot)
	if err != nil {
		return nil, err
	}
	_ = packages.PrintErrors(pkgs)

	idx := newIndex(repoRoot, pkgs)

	for _, p := range pkgs {
		if p == nil || p.TypesInfo == nil {
			continue
		}
		idx.indexPackage(p)
	}
	return idx, nil
}

// ------------------------------ Construction helpers ------------------------------

func loadPackages(repoRoot string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Dir: repoRoot,
	}
	return packages.Load(cfg, "./...")
}

func newIndex(repoRoot string, pkgs []*packages.Package) *Index {
	idx := &Index{
		repoRoot:             repoRoot,
		pkgs:                 pkgs,
		typeDeclsByPkg:       make(map[string][]TypeDecl),
		ifaceDeclsByPkg:      make(map[string][]InterfaceDecl),
		funcDeclsByPkg:       make(map[string][]FuncDecl),
		funcDeclsByFile:      make(map[string][]FuncDecl),
		typeDeclByPkgAndName: make(map[string]map[string]TypeDecl),
	}
	if len(pkgs) > 0 {
		idx.fset = pkgs[0].Fset
	}
	return idx
}

// ------------------------------ Indexing pipeline ------------------------------

func (idx *Index) indexPackage(p *packages.Package) {
	for i, file := range p.Syntax {
		if file == nil {
			continue
		}
		fileAbs := filepath.ToSlash(p.CompiledGoFiles[i])
		fileRel := rel(idx.repoRoot, fileAbs)
		idx.indexFile(p, file, fileRel)
	}
}

func (idx *Index) indexFile(p *packages.Package, file *ast.File, fileRel string) {
	pkgPath := p.PkgPath

	ast.Inspect(file, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.GenDecl:
			idx.handleGenDecl(p, pkgPath, fileRel, t)
		case *ast.FuncDecl:
			idx.handleFuncDecl(p, pkgPath, fileRel, t)
		}
		return true
	})
}

func (idx *Index) handleGenDecl(p *packages.Package, pkgPath, fileRel string, gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok || ts.Name == nil {
			continue
		}
		name := ts.Name.Name
		start, end := idx.lines(ts.Pos(), ts.End())

		// Interface
		if itNode, ok := ts.Type.(*ast.InterfaceType); ok {
			decl := idx.buildInterfaceDecl(p, pkgPath, fileRel, name, start, end, ts, itNode)
			idx.ifaceDeclsByPkg[pkgPath] = append(idx.ifaceDeclsByPkg[pkgPath], decl)
			continue
		}

		// Struct or alias
		isStruct := false
		if _, ok := ts.Type.(*ast.StructType); ok {
			isStruct = true
		}
		td := TypeDecl{
			PkgPath:   pkgPath,
			FilePath:  fileRel,
			StartLine: start,
			EndLine:   end,
			Name:      name,
			IsStruct:  isStruct,
		}
		idx.typeDeclsByPkg[pkgPath] = append(idx.typeDeclsByPkg[pkgPath], td)
		if _, ok := idx.typeDeclByPkgAndName[pkgPath]; !ok {
			idx.typeDeclByPkgAndName[pkgPath] = map[string]TypeDecl{}
		}
		idx.typeDeclByPkgAndName[pkgPath][name] = td
	}
}

func (idx *Index) handleFuncDecl(p *packages.Package, pkgPath, fileRel string, fdNode *ast.FuncDecl) {
	if fdNode.Name == nil {
		return
	}
	fnObj, _ := p.TypesInfo.Defs[fdNode.Name].(*types.Func)
	if fnObj == nil {
		return
	}

	start, end := idx.lines(fdNode.Pos(), fdNode.End())
	recvNamed := receiverNamed(fnObj)
	results := resultNamedTypes(fnObj)

	fd := FuncDecl{
		PkgPath:   pkgPath,
		FilePath:  fileRel,
		StartLine: start,
		EndLine:   end,
		Name:      fdNode.Name.Name,
		Obj:       fnObj,
		RecvType:  recvNamed,
		Results:   results,
	}

	idx.funcDeclsByPkg[pkgPath] = append(idx.funcDeclsByPkg[pkgPath], fd)
	idx.funcDeclsByFile[fileRel] = append(idx.funcDeclsByFile[fileRel], fd)
}

func (idx *Index) buildInterfaceDecl(
	p *packages.Package,
	pkgPath, fileRel, name string,
	start, end int,
	ts *ast.TypeSpec,
	itNode *ast.InterfaceType,
) InterfaceDecl {
	decl := InterfaceDecl{
		PkgPath:   pkgPath,
		FilePath:  fileRel,
		StartLine: start,
		EndLine:   end,
		Name:      name,
	}

	if itNode.Methods == nil {
		return decl
	}

	// Resolve the concrete *types.Interface once for accurate method lookups.
	var iface *types.Interface
	if obj := p.TypesInfo.Defs[ts.Name]; obj != nil && obj.Type() != nil {
		if it, ok := obj.Type().Underlying().(*types.Interface); ok {
			iface = it
		}
	}

	for _, f := range itNode.Methods.List {
		// Skip embedded interfaces for a minimal slice (as original code).
		if len(f.Names) == 0 {
			continue
		}
		mName := f.Names[0].Name
		ms, me := idx.lines(f.Pos(), f.End())

		var fnObj *types.Func
		if iface != nil {
			for i := 0; i < iface.NumMethods(); i++ {
				m := iface.Method(i)
				if m.Name() == mName {
					fnObj = m
					break
				}
			}
		}

		decl.Methods = append(decl.Methods, IfaceMethod{
			Name: mName, StartLine: ms, EndLine: me, Obj: fnObj,
		})
	}
	return decl
}

// ------------------------------ Small utilities ------------------------------

func (idx *Index) lines(start, end token.Pos) (int, int) {
	return idx.fset.PositionFor(start, true).Line, idx.fset.PositionFor(end, true).Line
}

func receiverNamed(fnObj *types.Func) *types.Named {
	sig, ok := fnObj.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return nil
	}
	if rn, ok := deref(sig.Recv().Type()).(*types.Named); ok {
		return rn
	}
	return nil
}

func resultNamedTypes(fnObj *types.Func) []*types.Named {
	var out []*types.Named
	sig, ok := fnObj.Type().(*types.Signature)
	if !ok || sig.Results() == nil {
		return out
	}
	for i := 0; i < sig.Results().Len(); i++ {
		if rn, ok := deref(sig.Results().At(i).Type()).(*types.Named); ok {
			out = append(out, rn)
		}
	}
	return out
}

func deref(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

func (idx *Index) fqnNamed(t *types.Named) (pkgPath, name string) {
	if t == nil || t.Obj() == nil || t.Obj().Pkg() == nil {
		return "", ""
	}
	return t.Obj().Pkg().Path(), t.Obj().Name()
}

// ResolveReceiverNamed finds the receiver *types.Named for a method at (fileRel, methodName, recvHintFromRecord).
// recvHint may be "T" (from "(T)" or "(*T)") — used to disambiguate when multiple methods share a name.
func (idx *Index) ResolveReceiverNamed(fileRel, methodName, recvHint string) (recv *types.Named, pkgPath string, ok bool) {
	fns := idx.funcDeclsByFile[fileRel]
	for _, fd := range fns {
		if fd.Name != methodName || fd.RecvType == nil {
			continue
		}
		_, rn := idx.fqnNamed(fd.RecvType)
		if recvHint == "" || rn == recvHint {
			p, _ := idx.fqnNamed(fd.RecvType)
			return fd.RecvType, p, true
		}
	}
	// fallback: scan all (slower but safe)
	for pkg, fds := range idx.funcDeclsByPkg {
		for _, fd := range fds {
			if fd.Name == methodName && fd.RecvType != nil {
				_, rn := idx.fqnNamed(fd.RecvType)
				if recvHint == "" || rn == recvHint {
					return fd.RecvType, pkg, true
				}
			}
		}
	}
	return nil, "", false
}

// ReceiverDecl returns the struct/alias declaration for a named type.
func (idx *Index) ReceiverDecl(named *types.Named) (TypeDecl, bool) {
	pkg, name := idx.fqnNamed(named)
	if pkg == "" || name == "" {
		return TypeDecl{}, false
	}
	if m := idx.typeDeclByPkgAndName[pkg]; m != nil {
		td, ok := m[name]
		return td, ok
	}
	return TypeDecl{}, false
}

// ImplementedInterfacesDeclaring lists interfaces that `named` implements
// AND that explicitly DECLARE method `methodName`.
// Order: same package first, then others (caller may sort further).
func (idx *Index) ImplementedInterfacesDeclaring(named *types.Named, methodName string) []InterfaceDecl {
	if named == nil {
		return nil
	}

	var out []InterfaceDecl
	namedType := named // keep as *types.Named for pointer checks

	for pkgPath, ifaces := range idx.ifaceDeclsByPkg {
		_ = pkgPath

		for _, idecl := range ifaces {
			// 1) Ensure this interface actually declares the method name
			hasMethod := false
			for _, m := range idecl.Methods {
				if m.Name == methodName {
					hasMethod = true
					break
				}
			}
			if !hasMethod {
				continue
			}

			// 2) Resolve the real *types.Interface via package scope (no m.Obj assumptions)
			it, ok := idx.lookupInterface(idecl.PkgPath, idecl.Name)
			if !ok || it == nil {
				continue
			}

			// 3) Check implementation against both T and *T
			if types.Implements(namedType, it) || types.Implements(types.NewPointer(namedType), it) {
				out = append(out, idecl)
			}
		}
	}

	// Prefer same package first (stable order otherwise)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PkgPath == out[j].PkgPath {
			return out[i].FilePath < out[j].FilePath
		}
		return out[i].PkgPath == named.Obj().Pkg().Path()
	})

	return out
}

// lookupInterface returns the *types.Interface for an interface declared as `name` in `pkgPath`.
func (idx *Index) lookupInterface(pkgPath, name string) (*types.Interface, bool) {
	if pkgPath == "" || name == "" {
		return nil, false
	}
	for _, p := range idx.pkgs {
		if p == nil || p.Types == nil || p.Types.Scope() == nil {
			continue
		}
		if p.PkgPath != pkgPath {
			continue
		}
		obj := p.Types.Scope().Lookup(name)
		if obj == nil {
			return nil, false
		}
		t := obj.Type()
		if t == nil {
			return nil, false
		}
		if it, ok := t.Underlying().(*types.Interface); ok {
			it.Complete()
			return it, true
		}
		return nil, false
	}
	return nil, false
}

// CounterpartMethodsOn returns methods on the same receiver named as counterparts (pairs map).
func (idx *Index) CounterpartMethodsOn(named *types.Named, methodName string, pairs map[string][]string) []FuncDecl {
	var want []string
	for a, bs := range pairs {
		if strings.HasPrefix(methodName, a) {
			want = append(want, bs...)
		}
		for _, b := range bs {
			if strings.HasPrefix(methodName, b) {
				want = append(want, a)
			}
		}
	}
	if len(want) == 0 {
		return nil
	}
	pkgPath, _ := idx.fqnNamed(named)
	var out []FuncDecl
	for _, fd := range idx.funcDeclsByPkg[pkgPath] {
		if fd.RecvType == nil {
			continue
		}
		_, rn := idx.fqnNamed(fd.RecvType)
		_, wantRN := idx.fqnNamed(named)
		if rn != wantRN {
			continue
		}
		for _, w := range want {
			if fd.Name == w || strings.HasPrefix(fd.Name, w) {
				out = append(out, fd)
				break
			}
		}
	}
	return out
}

// ConstructorsFor returns functions in same package whose name starts with New and return T or *T.
func (idx *Index) ConstructorsFor(named *types.Named) []FuncDecl {
	pkgPath, _ := idx.fqnNamed(named)
	if pkgPath == "" {
		return nil
	}
	_, tname := idx.fqnNamed(named)
	var out []FuncDecl
	for _, fd := range idx.funcDeclsByPkg[pkgPath] {
		if fd.RecvType != nil {
			continue // methods not constructors
		}
		if !strings.HasPrefix(fd.Name, "New") {
			continue
		}
		for _, r := range fd.Results {
			_, rn := idx.fqnNamed(r)
			if rn == tname {
				out = append(out, fd)
				break
			}
		}
	}
	return out
}

func rel(root, abs string) string {
	r, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(r)
}
