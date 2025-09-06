package stream

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DecoderFunc converts a JSONL line (bytes) into a value of type T.
type DecoderFunc[T any] func([]byte) (T, error)

type JSONLReader[T any] struct {
	rc     io.ReadCloser
	br     *bufio.Reader
	decode DecoderFunc[T]
	lineNo int
}

// NewJSONLReader opens a JSONL (optionally gzip) reader.
// path == "" → read from os.Stdin.
// decode == nil → defaults to json.Unmarshal.
func NewJSONLReader[T any](path string, decode DecoderFunc[T]) (Reader[T], error) {
	var (
		rc io.ReadCloser
	)

	if path == "" {
		rc = nopCloser{Reader: os.Stdin}
	} else {
		f, ferr := os.Open(path)
		if ferr != nil {
			return nil, ferr
		}
		rc = f
		if strings.EqualFold(filepath.Ext(path), ".gz") {
			gzr, gerr := gzip.NewReader(f)
			if gerr != nil {
				f.Close()
				return nil, gerr
			}
			rc = compositeCloser{r: gzr, c: f}
		}
	}

	if decode == nil {
		decode = func(b []byte) (T, error) {
			var v T
			err := json.Unmarshal(b, &v)
			return v, err
		}
	}

	return &JSONLReader[T]{
		rc:     rc,
		br:     bufio.NewReader(rc),
		decode: decode,
	}, nil
}

func (r *JSONLReader[T]) Close() error {
	if r == nil || r.rc == nil {
		return nil
	}
	return r.rc.Close()
}

func (r *JSONLReader[T]) Next() (T, bool, error) {
	var zero T
	if r == nil || r.br == nil {
		return zero, false, errors.New("reader not initialized")
	}

	for {
		line, err := r.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return zero, false, nil
			}
			return zero, false, err
		}
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue // skip header/comment
		}
		v, derr := r.decode([]byte(trim))
		if derr != nil {
			return zero, false, derr
		}
		return v, true, nil
	}
}

func (r *JSONLReader[T]) ReadAll() ([]T, error) {
	var out []T
	for {
		v, ok, err := r.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *JSONLReader[T]) readLine() (string, error) {
	var b []byte
	for {
		chunk, isPrefix, err := r.br.ReadLine()
		if err != nil {
			return "", err
		}
		r.lineNo++
		b = append(b, chunk...)
		if !isPrefix {
			break
		}
	}
	return string(b), nil
}

// --- helpers ---

type nopCloser struct{ io.Reader }

func (n nopCloser) Close() error { return nil }

type compositeCloser struct {
	r io.ReadCloser
	c io.Closer
}

func (cc compositeCloser) Read(p []byte) (int, error) { return cc.r.Read(p) }
func (cc compositeCloser) Close() error {
	_ = cc.r.Close()
	return cc.c.Close()
}
