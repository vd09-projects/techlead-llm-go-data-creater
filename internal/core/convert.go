package core

import (
	"sort"
	"strings"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

func ToRecords(repo *RepoNode, repoName, commitHash, lang string) []model.Record {
	var out []model.Record
	for _, f := range repo.Files {
		for _, fn := range f.Functions {
			rec := model.Record{
				Repo:      repoName,
				Commit:    commitHash,
				Lang:      lang,
				Path:      f.RelPath,
				Symbol:    symbolOf(fn),
				Signature: strings.TrimSpace(fn.Signature),
				StartLine: fn.StartLine,
				EndLine:   fn.EndLine,
				Code:      fn.Code,
			}

			if v, ok := fn.Aspects[AspectNeighbors].([]model.Neighbor); ok {
				rec.Neighbors = v
			}
			if v, ok := fn.Aspects[AspectSelection].(*model.Selection); ok {
				rec.Selection = v
			}
			if v, ok := fn.Aspects[AspectCallGraph].(*model.CallGraph); ok {
				rec.CallGraph = v
			}
			if v, ok := fn.Aspects[AspectCtxRefs].([]*model.ContextRef); ok && len(v) > 0 {
				rec.ContextRefs = v
			}
			out = append(out, rec)
		}
	}

	// Stable order: path asc, start_line asc
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].StartLine < out[j].StartLine
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func symbolOf(fn *FunctionNode) string {
	if fn.Recv == "" {
		return fn.Name
	}
	return fn.Recv + "." + fn.Name
}
