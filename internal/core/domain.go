package core

type AspectKind string

const (
	AspectNeighbors AspectKind = "neighbors"
	AspectSelection AspectKind = "selection"
	AspectCallGraph AspectKind = "call_graph"
	AspectCtxRefs   AspectKind = "context_refs"
)

type RepoNode struct {
	Root  string
	Files []*FileNode
}

type FileNode struct {
	RelPath   string
	Lines     []string // for neighbors; kept optional but handy
	Functions []*FunctionNode
}

type FunctionNode struct {
	Name          string
	Recv          string
	Signature     string
	StartLine     int
	EndLine       int
	TrimmedLength int
	Code          string
	IsTestFile    bool
	Aspects       map[AspectKind]any
}
