package strategies

import (
	ft "github.com/vd09-projects/techlead-llm-go-data-creater/internal/ft_data/ft_functional_understanding"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

type CodePredictorStrategy struct{}

func (*CodePredictorStrategy) Name() string { return "signature" }

func (cps *CodePredictorStrategy) Apply(rec model.Record) []*ft.FineTuneRecord {
	ftRecord := ft.NewFineTuneRecord()
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:     "user",
		Context:  cps.GetUserContext(rec),
		Messages: "Using the provided function symbol and signature, locate and output the full implementation (source code) of that function.",
	})
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:    "assistant",
		Context: cps.GetAssistantContext(rec),
	})
	return []*ft.FineTuneRecord{ftRecord}
}

func (*CodePredictorStrategy) GetUserContext(rec model.Record) *ft.BaseContext {
	context :=
		&ft.BaseContext{
			Repo:      rec.Repo,
			Path:      rec.Path,
			Symbol:    rec.Symbol,
			Signature: rec.Signature,
			Lines:     []int{rec.StartLine, rec.EndLine},
		}
	return context
}

func (*CodePredictorStrategy) GetAssistantContext(rec model.Record) *ft.BaseContext {
	context :=
		&ft.BaseContext{
			Code: rec.Code,
		}
	return context
}

func NewCodePredictorStrategy() *CodePredictorStrategy {
	return &CodePredictorStrategy{}
}
