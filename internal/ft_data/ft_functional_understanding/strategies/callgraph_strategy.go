package strategies

import (
	"fmt"

	ft "github.com/vd09-projects/techlead-llm-go-data-creater/internal/ft_data/ft_functional_understanding"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/utils"
)

type CallgraphStrategy struct{}

func (cs *CallgraphStrategy) Name() string { return "example_callgraph" }

func (cs *CallgraphStrategy) Apply(rec model.Record) []*ft.FineTuneRecord {
	ftRecords := []*ft.FineTuneRecord{}
	if len(rec.CallGraph.Callers) != 0 {
		ftRecords = append(ftRecords, cs.getCallersFineTuneRecord(rec))
	}
	if len(rec.CallGraph.Callees) != 0 {
		ftRecords = append(ftRecords, cs.getCalleesFineTuneRecord(rec))
	}
	return ftRecords
}

func (cs *CallgraphStrategy) getCalleesFineTuneRecord(rec model.Record) *ft.FineTuneRecord {
	ftRecord := ft.NewFineTuneRecord()
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:     "user",
		Context:  cs.getUserCalleesContext(rec),
		Messages: fmt.Sprintf("Can you list upto five example callers of %q?", rec.Symbol),
	})

	context := cs.getUserCalleesContext(rec)
	context.Callees = rec.CallGraph.Callees[:utils.Min(5, len(rec.CallGraph.Callees))]
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:    "assistant",
		Context: context,
	})
	return ftRecord
}

func (cs *CallgraphStrategy) getCallersFineTuneRecord(rec model.Record) *ft.FineTuneRecord {
	ftRecord := ft.NewFineTuneRecord()
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:     "user",
		Context:  cs.getUserCallersContext(rec),
		Messages: fmt.Sprintf("Can you list upto five example callers of %q?", rec.Symbol),
	})

	context := cs.getUserCallersContext(rec)
	context.Callers = rec.CallGraph.Callers[:utils.Min(5, len(rec.CallGraph.Callers))]
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:    "assistant",
		Context: context,
	})
	return ftRecord
}

func (*CallgraphStrategy) getUserCallersContext(rec model.Record) *ft.BaseContext {
	context :=
		&ft.BaseContext{
			Repo:      rec.Repo,
			Path:      rec.Path,
			Symbol:    rec.Symbol,
			Signature: rec.Signature,
			Lines:     [2]int{rec.StartLine, rec.EndLine},
		}
	return context
}

func (*CallgraphStrategy) getUserCalleesContext(rec model.Record) *ft.BaseContext {
	context :=
		&ft.BaseContext{
			Repo:      rec.Repo,
			Path:      rec.Path,
			Symbol:    rec.Symbol,
			Signature: rec.Signature,
			Lines:     [2]int{rec.StartLine, rec.EndLine},
			Code:      rec.Code,
		}
	return context
}

func NewCallgraphStrategy() *CallgraphStrategy {
	return &CallgraphStrategy{}
}
