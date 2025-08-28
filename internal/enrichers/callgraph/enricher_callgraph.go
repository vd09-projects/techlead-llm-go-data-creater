// internal/enrichers/callgraph/enricher.go
package callgraph

import (
	"context"
	"log"

	ncg "github.com/vd09-projects/techlead-llm-go-data-creater/internal/callgraph"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/core"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/utils"
)

type Config struct {
	RepoRoot   string
	MaxCallers int
	MaxCallees int
}

// Enricher uses a pluggable Computer; default is native (Static ∪ CHA).
type Enricher struct {
	cfg       Config
	computer  ncg.Computer
	initError error
}

// New wires a native computer by default (can inject a mock in tests).
func New(cfg Config) *Enricher {
	return &Enricher{
		cfg:      cfg,
		computer: ncg.NewNativeComputer(),
	}
}

// WithComputer allows DI for tests or alternate impls (static-only, cached file, etc.).
func (e *Enricher) WithComputer(c ncg.Computer) *Enricher {
	e.computer = c
	return e
}

func (e *Enricher) Kind() core.AspectKind { return core.AspectCallGraph }

func (e *Enricher) Enrich(_ context.Context, repo *core.RepoNode) error {
	if repo == nil {
		return nil
	}
	// Initialize once for the entire repo run.
	if err := e.computer.Init(e.cfg.RepoRoot); err != nil {
		// soft-fail: continue with empty results; record for observability if you log
		e.initError = err
	}

	for _, f := range repo.Files {
		if f == nil || len(f.Functions) == 0 {
			continue
		}
		for _, fn := range f.Functions {
			// Build symbol: "Func" or "(Recv).Func"
			sym := fn.Name
			if fn.Recv != "" {
				sym = fn.Recv + "." + fn.Name
			}

			callgraphResponse := &model.CallGraph{
				Callees:   nil,
				Callers:   nil,
				Precision: "native",
			}

			if callees, err := e.computer.GetCallees(f.RelPath, sym, e.cfg.MaxCallees); err == nil {
				callgraphResponse.Callees = utils.If(len(callees) > 0, callees).Else(nil)
			} else {
				log.Printf("callgraph: failed to get callees for %s in %s: %v", sym, f.RelPath, err)
			}

			if callers, err := e.computer.GetCallers(f.RelPath, sym, e.cfg.MaxCallers); err == nil {
				callgraphResponse.Callers = utils.If(len(callers) > 0, callers).Else(nil)
			} else {
				log.Printf("callgraph: failed to get callers for %s in %s: %v", sym, f.RelPath, err)
			}
			fn.Aspects[core.AspectCallGraph] = callgraphResponse
		}
	}
	return nil
}
