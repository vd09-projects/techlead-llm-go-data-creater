package enrichers

import (
	"context"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/core"
)

type Enricher interface {
	Kind() core.AspectKind
	Enrich(ctx context.Context, repo *core.RepoNode) error
}
