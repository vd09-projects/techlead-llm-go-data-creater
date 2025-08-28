// internal/callgraph/computer.go
package callgraph

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"go/token"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	staticcg "golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Result groups outgoing (callees) and incoming (callers) edges.
type Result struct {
	Callees []model.Edge `json:"callees"`
	Callers []model.Edge `json:"callers"`
}

// --------- Public interface to enable reuse & testability ---------

// Computer builds/holds a callgraph for a repo and answers edge queries.
// Contract: Init must be called before Edges. Implementations must be safe
// for repeated Edges calls after a successful Init.
type Computer interface {
	Init(repoRoot string) error
	GetCallers(fileRel, symbol string, maxCallers int) ([]model.Edge, error)
	GetCallees(fileRel, symbol string, maxCallees int) ([]model.Edge, error)
}

// --------- Native (Static ∪ CHA) implementation ---------

type fnKey struct{ Recv, Name, File string }

type nativeComputer struct {
	once sync.Once
	err  error

	// immutable after Init
	repoRoot string
	absRepo  string
	prog     *ssa.Program
	fset     *token.FileSet

	// indexes for fast lookup (populated once)
	fnByKey        map[fnKey]*ssa.Function
	nodeStaticByFn map[*ssa.Function]*callgraph.Node
	nodeCHAByFn    map[*ssa.Function]*callgraph.Node
}

// NewNativeComputer returns a Computer that builds Static and CHA graphs once.
func NewNativeComputer() Computer { return &nativeComputer{} }

func (c *nativeComputer) Init(repoRoot string) error {
	c.once.Do(func() {
		c.repoRoot = repoRoot
		c.absRepo, _ = filepath.Abs(repoRoot)

		if !c.hasGoMod() {
			// no module → permissible empty graph
			return
		}

		pkgs := c.loadPackages()
		if len(pkgs) == 0 {
			return
		}

		prog, fset := c.buildSSA(pkgs)
		if prog == nil || fset == nil {
			return
		}
		c.prog, c.fset = prog, fset

		staticCG := staticcg.CallGraph(prog)
		chaCG := cha.CallGraph(prog)
		if chaCG != nil {
			chaCG.DeleteSyntheticNodes()
		}

		c.fnByKey = map[fnKey]*ssa.Function{}
		c.nodeStaticByFn = map[*ssa.Function]*callgraph.Node{}
		c.nodeCHAByFn = map[*ssa.Function]*callgraph.Node{}

		collect := func(cg *callgraph.Graph, store map[*ssa.Function]*callgraph.Node) {
			if cg == nil {
				return
			}
			for fn, node := range cg.Nodes {
				if fn == nil || node == nil {
					continue
				}
				store[fn] = node
				if file := fileFor(c.fset, fn); file != "" {
					c.fnByKey[fnKey{
						Recv: c.recvOf(fn),
						Name: fn.Name(),
						File: filepath.ToSlash(file),
					}] = fn
				}
			}
		}
		collect(staticCG, c.nodeStaticByFn)
		collect(chaCG, c.nodeCHAByFn)
	})
	return c.err
}

// GetCallees returns up to maxCallees unique callees for the given (fileRel, symbol).
func (c *nativeComputer) GetCallees(fileRel, symbol string, maxCallees int) ([]model.Edge, error) {
	nodeS, nodeC := c.getTargetNodes(fileRel, symbol)
	if nodeS == nil && nodeC == nil {
		return nil, nil
	}
	out := c.unionOutEdges(nodeS, nodeC, maxCallees)
	sortEdges(out)
	return dedup(out), nil
}

// GetCallers returns up to maxCallers unique callers for the given (fileRel, symbol).
func (c *nativeComputer) GetCallers(fileRel, symbol string, maxCallers int) ([]model.Edge, error) {
	nodeS, nodeC := c.getTargetNodes(fileRel, symbol)
	if nodeS == nil && nodeC == nil {
		return nil, nil
	}
	out := c.unionInEdges(nodeS, nodeC, maxCallers)
	sortEdges(out)
	return dedup(out), nil
}

// getTargetNodes centralizes target resolution and node lookup.
// Returns the static and CHA nodes for the target (either may be nil).
func (c *nativeComputer) getTargetNodes(fileRel, symbol string) (nodeS, nodeC *callgraph.Node) {
	// If Init never ran or repo had no go.mod, we simply return nils.
	if c.fset == nil {
		return nil, nil
	}
	target := c.resolveTarget(filepath.ToSlash(fileRel), symbol)
	if target == nil {
		return nil, nil
	}
	return c.nodeStaticByFn[target], c.nodeCHAByFn[target]
}

// --------- internal helpers (nativeComputer methods) ---------

func (c *nativeComputer) hasGoMod() bool {
	_, err := os.Stat(filepath.Join(c.absRepo, "go.mod"))
	return err == nil
}

func (c *nativeComputer) neutralEnv() []string {
	env := os.Environ()
	return append(env, "GOWORK=off", "GOFLAGS=")
}

func (c *nativeComputer) loadPackages() []*packages.Package {
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Dir:   c.absRepo,
		Env:   c.neutralEnv(),
		Tests: false,
	}
	pkgs, _ := packages.Load(cfg, "./...")
	_ = packages.PrintErrors(pkgs)
	return pkgs
}

func (c *nativeComputer) buildSSA(pkgs []*packages.Package) (*ssa.Program, *token.FileSet) {
	prog, _ := ssautil.AllPackages(pkgs, ssa.BuilderMode(0))
	if prog == nil {
		return nil, nil
	}
	prog.Build()
	return prog, prog.Fset
}

func (c *nativeComputer) recvOf(fn *ssa.Function) string {
	if fn == nil || fn.Signature == nil || fn.Signature.Recv() == nil {
		return ""
	}
	return recvString(fn.Signature.Recv().Type())
}

func (c *nativeComputer) resolveTarget(targetFileRel, symbol string) *ssa.Function {
	wantRecv, wantName := ParseInputSymbol(symbol)

	// 1) name + recv + file suffix
	if wantRecv != "" {
		for k, fn := range c.fnByKey {
			if k.Name != wantName || !strings.HasSuffix(k.File, targetFileRel) {
				continue
			}
			if k.Recv == wantRecv {
				return fn
			}
		}
	}

	// 2) name + file suffix (functions only)
	if wantRecv == "" {
		for k, fn := range c.fnByKey {
			if k.Name == wantName && strings.HasSuffix(k.File, targetFileRel) && k.Recv == "" {
				return fn
			}
		}
	}
	return nil
}

func (c *nativeComputer) unionOutEdges(ns, nc *callgraph.Node, capN int) []model.Edge {
	seen := map[string]bool{}
	var out []model.Edge

	add := func(edges []*callgraph.Edge) {
		for _, e := range edges {
			if e == nil || e.Callee == nil || e.Callee.Func == nil {
				continue
			}
			fn := e.Callee.Func
			file := fileFor(c.fset, fn)
			if file == "" {
				continue
			}
			label := displayName(fn)
			k := label + "|" + file
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, model.Edge{Symbol: label, Path: rel(c.absRepo, file)})
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

func (c *nativeComputer) unionInEdges(ns, nc *callgraph.Node, capN int) []model.Edge {
	seen := map[string]bool{}
	var out []model.Edge

	add := func(edges []*callgraph.Edge) {
		for _, e := range edges {
			if e == nil || e.Caller == nil || e.Caller.Func == nil {
				continue
			}
			fn := e.Caller.Func
			file := fileFor(c.fset, fn)
			if file == "" {
				continue
			}
			label := displayName(fn)
			k := label + "|" + file
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, model.Edge{Symbol: label, Path: rel(c.absRepo, file)})
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

// shared helpers (not tied to receiver)

func sortEdges(es []model.Edge) {
	sort.Slice(es, func(i, j int) bool {
		if es[i].Symbol == es[j].Symbol {
			return es[i].Path < es[j].Path
		}
		return es[i].Symbol < es[j].Symbol
	})
}

func dedup(in []model.Edge) []model.Edge {
	seen := map[string]bool{}
	out := make([]model.Edge, 0, len(in))
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
