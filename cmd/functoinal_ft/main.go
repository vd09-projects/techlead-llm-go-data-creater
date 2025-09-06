package main

import (
	"flag"

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
)

func main() {
	flag.Parse()
	if *inPath == "" || *outPath == "" {
		panic("usage: -in scan.jsonl -out finetune.jsonl [flags]")
	}

	jr, err := stream.NewJSONLReader[model.Record](*inPath, nil)
	utils.MustNotErr(err)
	je := stream.NewJSONLEmitter[*ft.FineTuneRecord](*outPath, nil, true)

	reg := ft.NewQuestionRegistry().Register(ft_strategy.NewSignatureStrategy())
	if *useCallgraph {
		reg.Register(ft_strategy.NewCallgraphStrategy())
	}
	if *useContextref {
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
