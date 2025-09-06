package ftfunctionalunderstanding

import "github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"

// QuestionRegistry holds registered strategies in order.
type QuestionRegistry struct {
	strategies []QuestionStrategy
}

// NewQuestionRegistry creates an empty registry.
func NewQuestionRegistry() *QuestionRegistry {
	return &QuestionRegistry{strategies: []QuestionStrategy{}}
}

func (r *QuestionRegistry) Register(strats ...QuestionStrategy) *QuestionRegistry {
	r.strategies = append(r.strategies, strats...)
	return r
}

func (r *QuestionRegistry) Strategies() []QuestionStrategy {
	// return copy to keep immutability
	out := make([]QuestionStrategy, len(r.strategies))
	copy(out, r.strategies)
	return out
}

// Generator wires BaseContext (Builder) + Strategy list.
type Generator struct {
	registry *QuestionRegistry
}

// NewGenerator with injected pieces (DI-friendly).
func NewGenerator(reg *QuestionRegistry) *Generator {
	return &Generator{registry: reg}
}

// Generate runs all strategies and collects non-nil Q/A pairs.
func (g *Generator) Generate(rec model.Record) []*FineTuneRecord {
	var out []*FineTuneRecord
	for _, s := range g.registry.Strategies() {
		if qa := s.Apply(rec); qa != nil {
			out = append(out, qa...)
		}
	}
	return out
}
