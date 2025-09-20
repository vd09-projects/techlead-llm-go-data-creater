package model

import "encoding/json"

type Neighbor struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Code      string `json:"code"`
}

type Selection struct {
	Visibility string  `json:"visibility"`
	Reason     string  `json:"reason"`
	Score      float64 `json:"score"`
}

type Edge struct {
	Symbol string `json:"symbol"`
	Path   string `json:"path"`
}

type CallGraph struct {
	Callees   []Edge `json:"callees"`
	Callers   []Edge `json:"callers"`
	Precision string `json:"precision,omitempty"` // always "native" in our flow
}

type ContextRef struct {
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	Code      string `json:"code,omitempty"`
	Kind      string `json:"kind,omitempty"`   // receiver_type | interface_method | counterpart_method | factory_constructor
	Symbol    string `json:"symbol,omitempty"` // optional
	Why       string `json:"why,omitempty"`    // <=140 chars
}

type Record struct {
	Repo        string        `json:"repo"`
	Commit      string        `json:"commit"`
	Lang        string        `json:"lang"`
	Path        string        `json:"path"`
	Symbol      string        `json:"symbol"`
	Signature   string        `json:"signature"`
	StartLine   int           `json:"start_line"`
	EndLine     int           `json:"end_line"`
	Code        string        `json:"code"`
	Neighbors   []Neighbor    `json:"neighbors,omitempty"`
	Selection   *Selection    `json:"selection,omitempty"`
	CallGraph   *CallGraph    `json:"call_graph,omitempty"`
	ContextRefs []*ContextRef `json:"context_refs,omitempty"`
}

func (r Record) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
