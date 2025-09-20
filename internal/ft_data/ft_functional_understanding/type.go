package ftfunctionalunderstanding

import (
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

type BaseContext struct {
	Repo          string            `json:"repo,omitempty"`
	Path          string            `json:"path,omitempty"`
	Symbol        string            `json:"symbol,omitempty"`
	Lines         []int             `json:"lines,omitempty"`
	Signature     string            `json:"signature,omitempty"`
	RecvType      string            `json:"receiver_type,omitempty"`
	Interfaces    []string          `json:"interfaces,omitempty"`
	Neighbors     []model.Neighbor  `json:"neighbors,omitempty"`
	Callers       []model.Edge      `json:"callers,omitempty"`
	Callees       []model.Edge      `json:"callees,omitempty"`
	Notes         []string          `json:"notes,omitempty"`
	Code          string            `json:"code,omitempty"`
	CodeReference *model.ContextRef `json:"code_reference,omitempty"`
}

type Conversation struct {
	Role     string       `json:"role"`
	Context  *BaseContext `json:"content,omitempty"`
	Messages string       `json:"question,omitempty"`
	Response string       `json:"answer,omitempty"`
	// MustContain string       `json:"must_contain_for_validation,omitempty"`
}

type FineTuneRecord struct {
	Conversations []*Conversation `json:"messages"`
}

func NewFineTuneRecord() *FineTuneRecord {
	ftRecord := &FineTuneRecord{}
	ftRecord.Conversations = append(ftRecord.Conversations, &Conversation{
		Role: "system",
		Messages: `You are a senior engineer who has worked on this repository for years.
You have complete knowledge of every function, type, interface, and relationship in the codebase.
Your task is to answer questions about the repository with accurate, detailed, and contextual explanations.
Explain concepts in a clear, professional, and mentorship-like manner.`,
	})
	return ftRecord
}

// QuestionStrategy is the Strategy interface.
// Each strategy can decide whether to emit a Q/A for a record.
type QuestionStrategy interface {
	Name() string
	Apply(rec model.Record) []*FineTuneRecord
}

type Strategies string

const (
	StrategySignature     Strategies = "signature"
	StrategyCodePredictor Strategies = "code"
	StrategyCallgraph     Strategies = "example_callgraph"
	StrategyContextRefs   Strategies = "context_refs"
)
