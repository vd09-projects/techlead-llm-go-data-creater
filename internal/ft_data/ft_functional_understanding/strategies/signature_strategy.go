package strategies

import (
	"fmt"

	ft "github.com/vd09-projects/techlead-llm-go-data-creater/internal/ft_data/ft_functional_understanding"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

type SignatureStrategy struct{}

func (*SignatureStrategy) Name() string { return "signature" }

func (ss *SignatureStrategy) Apply(rec model.Record) []*ft.FineTuneRecord {
	ftRecord := ft.NewFineTuneRecord()
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:     "user",
		Context:  ss.GetUserContext(rec),
		Messages: fmt.Sprintf("What is the signature of the function or method named %q?", rec.Symbol),
	})
	ftRecord.Conversations = append(ftRecord.Conversations, &ft.Conversation{
		Role:     "assistant",
		Messages: fmt.Sprintf("The signature of %q is:\n\n%s", rec.Symbol, rec.Signature),
	})
	return []*ft.FineTuneRecord{ftRecord}
}

func (*SignatureStrategy) GetUserContext(rec model.Record) *ft.BaseContext {
	context :=
		&ft.BaseContext{
			Repo:   rec.Repo,
			Path:   rec.Path,
			Symbol: rec.Symbol,
			Lines:  [2]int{rec.StartLine, rec.EndLine},
			Code:   rec.Code,
		}
	return context
}

func NewSignatureStrategy() *SignatureStrategy {
	return &SignatureStrategy{}
}
