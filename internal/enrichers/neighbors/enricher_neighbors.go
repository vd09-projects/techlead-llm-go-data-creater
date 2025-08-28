package neighbors

import (
	"context"
	"strings"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/core"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/utils"
)

type Config struct {
	Before int
	After  int
}

type Enricher struct{ cfg Config }

func New(cfg Config) *Enricher { return &Enricher{cfg: cfg} }

func (e *Enricher) Kind() core.AspectKind { return core.AspectNeighbors }

func (e *Enricher) Enrich(_ context.Context, repo *core.RepoNode) error {
	if repo == nil {
		return nil
	}
	for _, f := range repo.Files {
		if f == nil || len(f.Functions) == 0 {
			continue
		}
		for _, fn := range f.Functions {
			nbs := e.BuildNeighborsFromLines(
				f.Lines, f.RelPath, fn.StartLine, fn.EndLine,
			)
			fn.Aspects[core.AspectNeighbors] = nbs
		}
	}
	return nil
}

func (e *Enricher) BuildNeighborsFromLines(lines []string, relPath string, startLine, endLine int) []model.Neighbor {
	before := utils.If(e.cfg.Before > 30, 30).Else(e.cfg.Before)
	after := utils.If(e.cfg.After > 30, 30).Else(e.cfg.After)

	if before == 0 && after == 0 {
		return nil
	}

	var out []model.Neighbor
	if before > 0 {
		s := utils.Max(1, startLine-before)
		e := startLine - 1
		if e >= s {
			snip := strings.Join(lines[s-1:e], "\n")
			if strings.TrimSpace(snip) != "" {
				out = append(out, model.Neighbor{Path: relPath, StartLine: s, EndLine: e, Code: snip})
			}
		}
	}
	if after > 0 {
		s := endLine + 1
		e := utils.Min(len(lines), endLine+after)
		if e >= s {
			snip := strings.Join(lines[s-1:e], "\n")
			if strings.TrimSpace(snip) != "" {
				out = append(out, model.Neighbor{Path: relPath, StartLine: s, EndLine: e, Code: snip})
			}
		}
	}
	return out
}
