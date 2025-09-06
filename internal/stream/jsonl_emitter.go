package stream

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONLEmitter writes one JSON object per line (JSONL).
type JSONLEmitter[T any] struct {
	outPath      string
	encode       EncoderFunc[T]
	addRunHeader bool
}

// NewJSONLEmitter creates a JSONLEmitter with a fixed output path.
// If encode is nil, it falls back to json.Marshal.
func NewJSONLEmitter[T any](outPath string, encode EncoderFunc[T], addRunHeader bool) *JSONLEmitter[T] {
	if encode == nil {
		encode = func(v T) ([]byte, error) { return json.Marshal(v) }
	}
	return &JSONLEmitter[T]{
		outPath:      outPath,
		encode:       encode,
		addRunHeader: addRunHeader,
	}
}

// Emit writes a slice of records.
func (je *JSONLEmitter[T]) Emit(records []T) error {
	for _, rec := range records {
		if err := je.EmitOne(rec); err != nil {
			return err
		}
	}
	return nil
}

// EmitOne writes a single record.
func (je *JSONLEmitter[T]) EmitOne(record T) error {
	var f *os.File
	var err error

	if je.outPath == "" {
		f = os.Stdout
	} else {
		if _, statErr := os.Stat(je.outPath); statErr == nil {
			// File exists → append
			f, err = os.OpenFile(je.outPath, os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
		} else {
			// Create new
			f, err = os.Create(je.outPath)
			if err != nil {
				return err
			}
		}
	}
	if f != os.Stdout {
		defer f.Close()
	}

	w := bufio.NewWriter(f)
	defer w.Flush()

	if je.addRunHeader {
		header := fmt.Sprintf("# Run at %s\n", time.Now().Format(time.RFC3339))
		if _, err := w.Write([]byte(header)); err != nil {
			return err
		}
		je.addRunHeader = false
	}

	b, err := je.encode(record)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return w.WriteByte('\n')
}
