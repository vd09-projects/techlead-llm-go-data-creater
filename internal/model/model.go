package model

import "encoding/json"

type Neighbor struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Code      string `json:"code"`
}

type Selection struct {
	Visibility string  `json:"visibility"` // exported|unexported|unknown
	Reason     string  `json:"reason"`     // constructor|public_api|encoder|core|sampling|fallback_parser|other
	Score      float64 `json:"score"`      // 0..1 rounded to 2 decimals
}

type Record struct {
	Repo      string     `json:"repo"`
	Commit    string     `json:"commit"`
	Lang      string     `json:"lang"`
	Path      string     `json:"path"`
	Symbol    string     `json:"symbol"`
	Signature string     `json:"signature"`
	StartLine int        `json:"start_line"`
	EndLine   int        `json:"end_line"`
	Code      string     `json:"code"`
	Neighbors []Neighbor `json:"neighbors,omitempty"`
	Selection *Selection `json:"selection,omitempty"`
}

func (r Record) ToJSON() ([]byte, error) {
	// compact JSON, omit empty via `omitempty`
	return json.Marshal(r)
}
