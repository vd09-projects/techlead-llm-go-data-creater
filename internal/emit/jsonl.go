package emit

import (
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/filehandler"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

type Emitter interface {
	Emit(records []model.Record, outPath string) error
}

type JSONLEmitter struct{}

func (JSONLEmitter) Emit(records []model.Record, outPath string) error {
	return filehandler.WriteOutput(records, outPath)
}
