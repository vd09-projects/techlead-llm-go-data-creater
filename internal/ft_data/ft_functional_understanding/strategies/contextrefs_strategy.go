package strategies

import (
	ft "github.com/vd09-projects/techlead-llm-go-data-creater/internal/ft_data/ft_functional_understanding"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

type ContextRefsStrategy struct{}

func (cs *ContextRefsStrategy) Name() string { return "context_refs" }

func (cs *ContextRefsStrategy) Apply(rec model.Record) []*ft.FineTuneRecord {
	ftRecords := []*ft.FineTuneRecord{}
	for _, ref := range rec.ContextRefs {
		ftRecord := ft.NewFineTuneRecord()
		qtctx := cs.GetUserContext(rec, ref)
		qtctx.CodeReference = nil
		ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
			Role:     "user",
			Context:  qtctx,
			Messages: ref.Why,
		})

		ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
			Role:    "assistant",
			Context: cs.GetUserContext(rec, ref),
		})
		ftRecords = append(ftRecords, ftRecord)
	}
	return ftRecords
}

func (*ContextRefsStrategy) GetUserContext(rec model.Record, ref *model.ContextRef) *ft.BaseContext {
	context :=
		&ft.BaseContext{
			Repo:      rec.Repo,
			Path:      rec.Path,
			Symbol:    rec.Symbol,
			Signature: rec.Signature,
			Lines:     [2]int{rec.StartLine, rec.EndLine},
			Code:      rec.Code,
			CodeReference: &model.ContextRef{
				Path:      ref.Path,
				StartLine: ref.StartLine,
				EndLine:   ref.EndLine,
				Code:      ref.Code,
				Kind:      ref.Kind,
				Why:       "",
			},
		}
	return context
}

func NewContextRefsStrategy() *ContextRefsStrategy {
	return &ContextRefsStrategy{}
}
