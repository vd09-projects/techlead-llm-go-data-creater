package selection

import (
	"context"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/core"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

type Enricher struct {
	RepoRoot string
	Strat    Strategy
}

func New(repoRoot string, strat Strategy) *Enricher {
	if strat == nil {
		strat = NewDefaultStrategy(repoRoot)
	}
	return &Enricher{RepoRoot: repoRoot, Strat: strat}
}

func (e *Enricher) Kind() core.AspectKind { return core.AspectSelection }

func (e *Enricher) Enrich(ctx context.Context, repo *core.RepoNode) error {
	if repo == nil {
		return nil
	}

	for _, f := range repo.Files {
		if f == nil || len(f.Functions) == 0 {
			continue
		}
		for _, fn := range f.Functions {
			fn.Aspects[core.AspectSelection] = &model.Selection{
				Visibility: e.Strat.Visibility(f.RelPath, fn),
				Reason:     e.Strat.ClassifyReason(f.RelPath, fn),
				Score:      e.Strat.Score(f.RelPath, fn),
			}
		}
	}
	return nil
}
