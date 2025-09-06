package main

import (
	"context"
	"flag"
	"log"
	"strings"

	baseenrichers "github.com/vd09-projects/techlead-llm-go-data-creater/internal/enrichers"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/enrichers/callgraph"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/enrichers/contextrefs"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/enrichers/neighbors"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/enrichers/selection"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/extractor"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/gitutil"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/pipeline"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/scanner"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/stream"
)

func main() {
	var (
		repoRoot       = flag.String("repo", ".", "Path to repo root")
		commitRef      = flag.String("commit", "", "Commit hash/ref (metadata only)")
		includePrivate = flag.Bool("include-private", false, "Include unexported/private functions (honored by extractor)")
		maxFuncLines   = flag.Int("max-func-lines", 120, "Hard cap on function lines (after trimming)")
		minFuncLines   = flag.Int("min-func-lines", 3, "Skip functions shorter than this many lines")

		ctxBefore = flag.Int("context-before", 0, "Neighbor lines before function start (<=30)")
		ctxAfter  = flag.Int("context-after", 0, "Neighbor lines after function end (<=30)")

		excludeCSV = flag.String("exclude", "(^|/)(vendor|third_party|\\.git|build|dist)/", "Comma-separated regex to exclude paths")

		fieldsCSV = flag.String("fields", "repo,commit,lang,path,symbol,signature,start_line,end_line,code,neighbors,selection,call_graph,context_refs", "Comma-separated output fields")

		debug   = flag.Bool("debug", false, "Verbose logging")
		outPath = flag.String("out", "", "Path to JSONL output file (optional, defaults to stdout)")

		maxCallers = flag.Int("max-callers", 10, "Max callers included")
		maxCallees = flag.Int("max-callees", 10, "Max callees included")

		// NEW: context_refs specific
		ctxMaxRefs  = flag.Int("context-refs-max", 2, "Max context refs per record (<=2)")
		ctxMaxLines = flag.Int("context-refs-max-lines", 30, "Max lines per snippet (<=30)")
	)
	flag.Parse()
	_ = includePrivate

	log.SetFlags(0)
	if *debug {
		log.SetPrefix("[DEBUG] ")
	}

	fields := ParseFields(*fieldsCSV)

	ens := make([]baseenrichers.Enricher, 0, 4)
	if fields["neighbors"] && (*ctxBefore > 0 || *ctxAfter > 0) {
		ens = append(ens, neighbors.New(neighbors.Config{
			Before: *ctxBefore, After: *ctxAfter,
		}))
	}
	if fields["selection"] {
		ens = append(ens, selection.New(*repoRoot, nil))
	}
	if fields["call_graph"] {
		ens = append(ens, callgraph.New(callgraph.Config{
			RepoRoot: *repoRoot, MaxCallers: *maxCallers, MaxCallees: *maxCallees,
		}))
	}
	if fields["context_refs"] {
		// Build semantic index ONCE if context_refs requested
		idx, err := contextrefs.Load(*repoRoot)
		if err != nil && *debug {
			log.Printf("semindex load error: %v", err)
		} else {
			ens = append(ens, contextrefs.New(
				contextrefs.Config{MaxRefs: *ctxMaxRefs, MaxLines: *ctxMaxLines},
				idx,
			))
		}
	}

	reader := scanner.NewGoPackagesReader(*repoRoot, *excludeCSV, *debug)

	je := stream.NewJSONLEmitter[model.Record](*outPath, nil, true)
	pl := pipeline.New(
		reader,
		extractor.NewASTExtractor(*minFuncLines, *maxFuncLines),
		ens,
		je,
	)

	opts := pipeline.Options{
		RepoRoot:   *repoRoot,
		OutPath:    *outPath,
		RepoName:   gitutil.InferRepoName(*repoRoot),
		CommitHash: gitutil.ResolveCommit(*repoRoot, *commitRef),
		Lang:       "go",
	}

	if err := pl.Run(context.Background(), opts); err != nil {
		log.Fatalf("scan error: %v", err)
	}
}

func ParseFields(csv string) map[string]bool {
	m := make(map[string]bool)
	for _, f := range strings.Split(csv, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			m[f] = true
		}
	}
	return m
}
