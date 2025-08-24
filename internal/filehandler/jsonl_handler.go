package filehandler

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
)

func WriteOutput(records []model.Record, outPath string) error {
	// Decide where to write
	var f *os.File
	if outPath == "" {
		f = os.Stdout
	} else {
		// Check if file exists
		if _, err := os.Stat(outPath); err == nil {
			// Exists → append mode
			f, err = os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			// Add a run header with timestamp
			header := fmt.Sprintf("# Run at %s\n", time.Now().Format(time.RFC3339))
			if _, err := f.WriteString(header); err != nil {
				f.Close()
				return err
			}
		} else {
			// Does not exist → create
			f, err = os.Create(outPath)
			if err != nil {
				return err
			}
		}
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	for _, rec := range records {
		b, err := rec.ToJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
	}
	return nil
}
