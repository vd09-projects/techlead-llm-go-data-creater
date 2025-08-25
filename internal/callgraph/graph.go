// internal/callgraph/graph.go
package callgraph

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go/token"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	staticcg "golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Edge is a compact (symbol, path) pair used in JSON output.
type Edge struct {
	Symbol string `json:"symbol"`
	Path   string `json:"path"`
}

// Result groups outgoing (callees) and incoming (callers) edges.
type Result struct {
	Callees []Edge `json:"callees"`
	Callers []Edge `json:"callers"`
}

// Build constructs a native call graph for the module at repoRoot and returns
// up to maxCallers/maxCallees edges for the given target (fileRel, symbol).
// It takes the UNION of the Static and CHA call graphs to capture both direct
// calls and interface/virtual dispatch.
//
// - repoRoot: absolute or relative path to the repo/module root (must contain go.mod)
// - fileRel : function file path relative to repoRoot (posix-style is fine)
// - symbol  : "Func" or "(Type).Method" or "(*Type).Method"
func Build(repoRoot, fileRel, symbol string, maxCallers, maxCallees int) (Result, error) {
	absRepo, _ := filepath.Abs(repoRoot)
	targetFileRel := filepath.ToSlash(fileRel)

	// Use current environment (PATH, GOOS, GOARCH, GOTOOLCHAIN, etc.) and
	// only neutralize workspace/flag interference.
	env := os.Environ()
	env = append(env, "GOWORK=off", "GOFLAGS=")

	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Dir:   absRepo,
		Env:   env,
		Tests: false,
	}

	// Must be a module; if no go.mod, return empty (valid) result.
	if _, err := os.Stat(filepath.Join(absRepo, "go.mod")); err != nil {
		return Result{}, nil
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil || len(pkgs) == 0 {
		return Result{}, nil
	}
	_ = packages.PrintErrors(pkgs)

	// Build SSA program.
	prog, _ := ssautil.AllPackages(pkgs, ssa.BuilderMode(0))
	if prog == nil {
		return Result{}, nil
	}
	prog.Build()
	fset := prog.Fset
	if fset == nil {
		return Result{}, nil
	}

	// Build both graphs and take their union.
	staticCG := staticcg.CallGraph(prog)
	chaCG := cha.CallGraph(prog)
	if chaCG != nil {
		chaCG.DeleteSyntheticNodes()
	}

	// Index: (name, file) -> function; function -> node (static/CHA).
	type key struct{ Name, File string }
	fnByKey := map[key]*ssa.Function{}
	nodeStaticByFn := map[*ssa.Function]*callgraph.Node{}
	nodeCHAByFn := map[*ssa.Function]*callgraph.Node{}

	collect := func(cg *callgraph.Graph, store map[*ssa.Function]*callgraph.Node) {
		if cg == nil {
			return
		}
		for fn, node := range cg.Nodes {
			if fn == nil || node == nil {
				continue
			}
			store[fn] = node
			file := fileFor(fset, fn)
			if file == "" {
				continue
			}
			fnByKey[key{fn.Name(), filepath.ToSlash(file)}] = fn
		}
	}
	collect(staticCG, nodeStaticByFn)
	collect(chaCG, nodeCHAByFn)

	// Resolve the target function by decreasing strictness.
	wantRecv, wantName := ParseInputSymbol(symbol)
	fnRecv := func(fn *ssa.Function) string {
		if fn == nil || fn.Signature == nil || fn.Signature.Recv() == nil {
			return ""
		}
		return recvString(fn.Signature.Recv().Type())
	}

	var target *ssa.Function
	// 1) name + recv + file suffix
	if wantRecv != "" {
		for k, fn := range fnByKey {
			if k.Name != wantName || !strings.HasSuffix(k.File, targetFileRel) {
				continue
			}
			if fnRecv(fn) == wantRecv {
				target = fn
				break
			}
		}
	}
	// 2) name + recv (any file)
	if target == nil && wantRecv != "" {
		for k, fn := range fnByKey {
			if k.Name == wantName && fnRecv(fn) == wantRecv {
				target = fn
				break
			}
		}
	}
	// 3) name + file suffix
	if target == nil {
		for k, fn := range fnByKey {
			if k.Name == wantName && strings.HasSuffix(k.File, targetFileRel) {
				target = fn
				break
			}
		}
	}
	// 4) name only
	if target == nil {
		for k, fn := range fnByKey {
			if k.Name == wantName {
				target = fn
				break
			}
		}
	}
	if target == nil {
		return Result{}, nil
	}

	// Nodes for the target (may exist in only one graph).
	nodeS := nodeStaticByFn[target]
	nodeC := nodeCHAByFn[target]
	if nodeS == nil && nodeC == nil {
		return Result{}, nil
	}

	// Merge outgoing/incoming edges from both graphs.
	callees := unionOutEdges(nodeS, nodeC, fset, absRepo, maxCallees)
	callers := unionInEdges(nodeS, nodeC, fset, absRepo, maxCallers)

	// Deterministic order + dedup.
	sort.Slice(callees, func(i, j int) bool {
		if callees[i].Symbol == callees[j].Symbol {
			return callees[i].Path < callees[j].Path
		}
		return callees[i].Symbol < callees[j].Symbol
	})
	sort.Slice(callers, func(i, j int) bool {
		if callers[i].Symbol == callers[j].Symbol {
			return callers[i].Path < callers[j].Path
		}
		return callers[i].Symbol < callers[j].Symbol
	})

	return Result{
		Callees: dedup(callees),
		Callers: dedup(callers),
	}, nil
}

// unionOutEdges returns up to capN unique callees from the union of static and CHA nodes.
func unionOutEdges(ns, nc *callgraph.Node, fset *token.FileSet, repo string, capN int) []Edge {
	seen := map[string]bool{}
	var out []Edge

	add := func(edges []*callgraph.Edge) {
		for _, e := range edges {
			if e == nil || e.Callee == nil || e.Callee.Func == nil {
				continue
			}
			fn := e.Callee.Func
			file := fileFor(fset, fn)
			if file == "" {
				continue
			}
			label := displayName(fn)
			k := label + "|" + file
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, Edge{Symbol: label, Path: rel(repo, file)})
			if len(out) >= capN {
				return
			}
		}
	}

	if ns != nil {
		add(ns.Out)
	}
	if len(out) < capN && nc != nil {
		add(nc.Out)
	}
	return out
}

// unionInEdges returns up to capN unique callers from the union of static and CHA nodes.
func unionInEdges(ns, nc *callgraph.Node, fset *token.FileSet, repo string, capN int) []Edge {
	seen := map[string]bool{}
	var out []Edge

	add := func(edges []*callgraph.Edge) {
		for _, e := range edges {
			if e == nil || e.Caller == nil || e.Caller.Func == nil {
				continue
			}
			fn := e.Caller.Func
			file := fileFor(fset, fn)
			if file == "" {
				continue
			}
			label := displayName(fn)
			k := label + "|" + file
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, Edge{Symbol: label, Path: rel(repo, file)})
			if len(out) >= capN {
				return
			}
		}
	}

	if ns != nil {
		add(ns.In)
	}
	if len(out) < capN && nc != nil {
		add(nc.In)
	}
	return out
}

func dedup(in []Edge) []Edge {
	seen := map[string]bool{}
	out := make([]Edge, 0, len(in))
	for _, e := range in {
		k := e.Symbol + "|" + e.Path
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, e)
	}
	return out
}