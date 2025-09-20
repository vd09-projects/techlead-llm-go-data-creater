package main

import (
	"encoding/json"
	"flag"
	"strings"

	ffu "github.com/vd09-projects/techlead-llm-go-data-creater/internal/ft_data/ft_functional_understanding"
	ft "github.com/vd09-projects/techlead-llm-go-data-creater/internal/ft_data/ft_functional_understanding"
	ft_strategy "github.com/vd09-projects/techlead-llm-go-data-creater/internal/ft_data/ft_functional_understanding/strategies"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/stream"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/utils"
)

var (
	inPath        = flag.String("in", "", "Input JSONL from scanrepo")
	outPath       = flag.String("out", "", "Output JSONL for fine-tuning")
	useCallgraph  = flag.Bool("use-callgraph", false, "Generate questions for callgraph functions instead of all functions")
	useContextref = flag.Bool("use-contextref", false, "Generate questions for context-referenced functions instead of all functions")
	strategiesCSV = flag.String("strategies", mergeStrategies(ffu.StrategyCallgraph, ffu.StrategyCodePredictor, ffu.StrategySignature, ffu.StrategyContextRefs), "Comma-separated strategies to use. Options: signature, example_callgraph, context_refs")
)

func main() {
	flag.Parse()
	if *inPath == "" || *outPath == "" {
		panic("usage: -in scan.jsonl -out finetune.jsonl [flags]")
	}

	strategies := ParseFields(*strategiesCSV)

	jr, err := stream.NewJSONLReader[model.Record](*inPath, nil)
	utils.MustNotErr(err)
	je := stream.NewJSONLEmitter(*outPath, func(ftr *ft.FineTuneRecord) ([]byte, error) {
		// return json.Marshal(ftr.Conversations)
		return json.Marshal(ftr)
	}, true)

	reg := ft.NewQuestionRegistry()

	if strategies[ffu.StrategySignature] {
		reg.Register(ft_strategy.NewSignatureStrategy())
	}
	if strategies[ffu.StrategyCodePredictor] {
		reg.Register(ft_strategy.NewCodePredictorStrategy())
	}
	if strategies[ffu.StrategyCallgraph] {
		reg.Register(ft_strategy.NewCallgraphStrategy())
	}
	if strategies[ffu.StrategyContextRefs] {
		reg.Register(ft_strategy.NewContextRefsStrategy())
	}

	gen := ft.NewGenerator(reg)

	for {
		rec, ok, err := jr.Next()
		utils.MustNotErr(err)
		if !ok {
			break
		}

		ftRecords := gen.Generate(rec)
		je.Emit(ftRecords)
	}
	jr.Close()
}

func ParseFields(csv string) map[ffu.Strategies]bool {
	m := make(map[ffu.Strategies]bool)
	for _, f := range strings.Split(csv, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			m[ffu.Strategies(f)] = true
		}
	}
	return m
}

func mergeStrategies(strats ...ffu.Strategies) string {
	var parts []string
	for _, s := range strats {
		parts = append(parts, string(s))
	}
	return strings.Join(parts, ",")
}
