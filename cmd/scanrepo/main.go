package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/filehandler"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/gitutil"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/scanner"
)

func main() {
	var (
		repoRoot       = flag.String("repo", ".", "Path to repo root")
		commitRef      = flag.String("commit", "", "Commit hash/ref (metadata only)")
		includePrivate = flag.Bool("include-private", false, "Include unexported/private functions")
		maxFuncLines   = flag.Int("max-func-lines", 120, "Hard cap on function lines (after trimming)")
		minFuncLines   = flag.Int("min-func-lines", 3, "Skip functions shorter than this many lines")
		ctxBefore      = flag.Int("context-before", 0, "Neighbor lines before function start (<=30)")
		ctxAfter       = flag.Int("context-after", 0, "Neighbor lines after function end (<=30)")
		excludeCSV     = flag.String("exclude", "(^|/)(vendor|third_party|\\.git|build|dist)/", "Comma-separated regex to exclude paths")
		fieldsCSV      = flag.String("fields", "repo,commit,lang,path,symbol,signature,start_line,end_line,code,neighbors,selection,call_graph", "Comma-separated output fields")
		debug          = flag.Bool("debug", false, "Verbose logging")
		outPath        = flag.String("out", "", "Path to JSONL output file (optional, defaults to stdout)")
		maxCallers     = flag.Int("max-callers", 10, "Max callers in call_graph (only if --fields includes call_graph)")
		maxCallees     = flag.Int("max-callees", 10, "Max callees in call_graph (only if --fields includes call_graph)")
	)
	flag.Parse()

	log.SetFlags(0)
	if *debug {
		log.SetPrefix("[DEBUG] ")
	}

	repoName := gitutil.InferRepoName(*repoRoot)
	commitHash := gitutil.ResolveCommit(*repoRoot, *commitRef)

	opts := scanner.Options{
		RepoRoot:       *repoRoot,
		ExcludeCSV:     *excludeCSV,
		IncludePrivate: *includePrivate,
		MaxFuncLines:   *maxFuncLines,
		MinFuncLines:   *minFuncLines,
		ContextBefore:  *ctxBefore,
		ContextAfter:   *ctxAfter,
		Debug:          *debug,
		Fields:         scanner.ParseFields(*fieldsCSV),
		MaxCallers:     *maxCallers,
		MaxCallees:     *maxCallees,
	}

	records, err := scanner.Run(opts)
	if err != nil {
		log.Fatalf("scan error: %v", err)
	}

	// fill static meta only if the field is requested
	for i := range records {
		if opts.Fields["repo"] {
			records[i].Repo = repoName
		}
		if opts.Fields["commit"] {
			records[i].Commit = commitHash
		}
		if opts.Fields["lang"] {
			records[i].Lang = "go"
		}
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Path == records[j].Path {
			return records[i].StartLine < records[j].StartLine
		}
		return records[i].Path < records[j].Path
	})

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := filehandler.WriteOutput(records, *outPath); err != nil {
		log.Fatalf("write error: %v", err)
	}
}
