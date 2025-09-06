package pipeline

import (
	"context"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/core"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/enrichers"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/extractor"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/scanner"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/stream"
)

type Options struct {
	RepoRoot   string
	OutPath    string
	RepoName   string
	CommitHash string
	Lang       string
}

type Pipeline struct {
	Reader    *scanner.GoPackagesReader
	Extractor extractor.Extractor
	Enrichers []enrichers.Enricher
	Emitter   stream.Emitter[model.Record]
}

func New(reader *scanner.GoPackagesReader, ex extractor.Extractor, ens []enrichers.Enricher, em stream.Emitter[model.Record]) *Pipeline {
	return &Pipeline{Reader: reader, Extractor: ex, Enrichers: ens, Emitter: em}
}

func (p *Pipeline) Run(ctx context.Context, opts Options) error {
	// list & build in-memory tree
	units, err := p.Reader.List()
	if err != nil {
		return err
	}
	repo := &core.RepoNode{
		Root:  opts.RepoRoot,
		Files: p.Extractor.Extract(units),
	}

	// enrichment passes
	for _, enr := range p.Enrichers {
		if err := enr.Enrich(ctx, repo); err != nil {
			return err
		}
	}

	// flatten -> records
	recs := core.ToRecords(repo, opts.RepoName, opts.CommitHash, opts.Lang)
	return p.Emitter.Emit(recs)
}
