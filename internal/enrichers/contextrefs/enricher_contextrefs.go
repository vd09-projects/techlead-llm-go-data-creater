package contextrefs

import (
	"context"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/core"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/model"
	"github.com/vd09-projects/techlead-llm-go-data-creater/internal/utils"
)

// ---------- Config & construction ----------

const (
	defaultMaxRefs  = 2
	defaultMaxLines = 30
	hardCapMaxRefs  = 8
	hardCapMaxLines = 120
)

type Config struct {
	MaxRefs     int
	MaxLines    int
	Counterpart map[string][]string
}

func (c Config) withDefaults() Config {
	out := c
	if out.MaxRefs <= 0 {
		out.MaxRefs = defaultMaxRefs
	}
	if out.MaxRefs > hardCapMaxRefs {
		out.MaxRefs = hardCapMaxRefs
	}
	if out.MaxLines <= 0 {
		out.MaxLines = defaultMaxLines
	}
	if out.MaxLines > hardCapMaxLines {
		out.MaxLines = hardCapMaxLines
	}
	return out
}

type Enricher struct {
	cfg Config
	idx *Index
}

func New(cfg Config, idx *Index) *Enricher {
	return &Enricher{cfg: cfg.withDefaults(), idx: idx}
}

func (e *Enricher) Kind() core.AspectKind { return core.AspectCtxRefs }

// ---------- Public API ----------

func (e *Enricher) Enrich(_ context.Context, repo *core.RepoNode) error {
	if repo == nil || e.idx == nil {
		return nil
	}

	fileMap := make(map[string]*core.FileNode, len(repo.Files))
	for _, f := range repo.Files {
		if f != nil {
			fileMap[norm(f.RelPath)] = f
		}
	}

	for _, f := range repo.Files {
		if f == nil || len(f.Functions) == 0 {
			continue
		}
		for _, fn := range f.Functions {
			refs := e.computeForFunction(fileMap, f, fn)
			if len(refs) == 0 {
				continue
			}
			if fn.Aspects == nil {
				fn.Aspects = make(map[core.AspectKind]any, 1)
			}
			fn.Aspects[core.AspectCtxRefs] = refs
		}
	}
	return nil
}

// ---------- Main dispatcher ----------

func (e *Enricher) computeForFunction(
	files map[string]*core.FileNode,
	file *core.FileNode,
	fn *core.FunctionNode,
) []*model.ContextRef {
	recvName := utils.RecvBaseType(fn.Recv)
	recvT, pkgPath, ok := e.idx.ResolveReceiverNamed(file.RelPath, fn.Name, recvName)
	if !ok || recvT == nil {
		return nil
	}

	var refs []*model.ContextRef
	refs = append(refs, e.receiverTypeRef(files, recvT)...)
	refs = append(refs, e.interfaceMethodRef(files, recvT, pkgPath, fn.Name)...)
	// refs = append(refs, e.counterpartMethodRef(files, file, recvT, fn.Name)...)
	refs = append(refs, e.constructorRef(files, recvT)...)

	refs = dedupRefs(refs)
	refs = stableOrder(refs)

	if len(refs) > e.cfg.MaxRefs {
		refs = refs[:e.cfg.MaxRefs]
	}
	return refs
}

// ---------- Section helpers ----------

// 1) Receiver type definition
func (e *Enricher) receiverTypeRef(files map[string]*core.FileNode, recvT *types.Named) []*model.ContextRef {
	td, ok := e.idx.ReceiverDecl(recvT)
	if !ok {
		return nil
	}
	if cr, ok := e.slice(files, td.FilePath, td.StartLine, td.EndLine,
		"receiver_type", recvT.Obj().Name(),
		"Receiver shape clarifies which fields/methods this method depends on."); ok {
		return []*model.ContextRef{cr}
	}
	return nil
}

// 2) Interface methods declaring this function
func (e *Enricher) interfaceMethodRef(
	files map[string]*core.FileNode,
	recvT *types.Named,
	pkgPath, fnName string,
) []*model.ContextRef {
	ifaces := e.idx.ImplementedInterfacesDeclaring(recvT, fnName)
	sort.SliceStable(ifaces, func(i, j int) bool {
		if (ifaces[i].PkgPath == pkgPath) != (ifaces[j].PkgPath == pkgPath) {
			return ifaces[i].PkgPath == pkgPath
		}
		if ifaces[i].FilePath != ifaces[j].FilePath {
			return ifaces[i].FilePath < ifaces[j].FilePath
		}
		return ifaces[i].Name < ifaces[j].Name
	})
	for _, idecl := range ifaces {
		for _, m := range idecl.Methods {
			if m.Name != fnName {
				continue
			}
			if cr, ok := e.slice(files, idecl.FilePath, m.StartLine, m.EndLine,
				"interface_method", idecl.Name+"."+fnName,
				"Shows the interface contract this method satisfies."); ok {
				return []*model.ContextRef{cr}
			}
		}
	}
	return nil
}

// 3) Counterpart methods on the same receiver
// TODO pending setup Counterpart
func (e *Enricher) counterpartMethodRef(
	files map[string]*core.FileNode,
	file *core.FileNode,
	recvT *types.Named,
	fnName string,
) []*model.ContextRef {
	cps := e.idx.CounterpartMethodsOn(recvT, fnName, e.cfg.Counterpart)
	if len(cps) == 0 {
		return nil
	}
	sort.SliceStable(cps, func(i, j int) bool {
		if (cps[i].FilePath == file.RelPath) != (cps[j].FilePath == file.RelPath) {
			return cps[i].FilePath == file.RelPath
		}
		if cps[i].FilePath != cps[j].FilePath {
			return cps[i].FilePath < cps[j].FilePath
		}
		return cps[i].StartLine < cps[j].StartLine
	})
	d := cps[0]
	label := recvT.Obj().Name() + "." + d.Name
	if cr, ok := e.slice(files, d.FilePath, d.StartLine, d.EndLine,
		"counterpart_method", label,
		"A symmetric API clarifies paired usage and trade-offs."); ok {
		return []*model.ContextRef{cr}
	}
	return nil
}

// 4) Constructors for this type
func (e *Enricher) constructorRef(files map[string]*core.FileNode, recvT *types.Named) []*model.ContextRef {
	cons := e.idx.ConstructorsFor(recvT)
	if len(cons) == 0 {
		return nil
	}
	sort.SliceStable(cons, func(i, j int) bool {
		if cons[i].FilePath != cons[j].FilePath {
			return cons[i].FilePath < cons[j].FilePath
		}
		return cons[i].StartLine < cons[j].StartLine
	})
	d := cons[0]
	if cr, ok := e.slice(files, d.FilePath, d.StartLine, d.EndLine,
		"factory_constructor", d.Name,
		"Constructor shows how the central type is created/configured."); ok {
		return []*model.ContextRef{cr}
	}
	return nil
}

// ---------- Helpers ----------

func (e *Enricher) slice(
	files map[string]*core.FileNode,
	rel string,
	start, end int,
	kind, symbol, why string,
) (*model.ContextRef, bool) {
	if start < 1 || end < start {
		return nil, false
	}
	f := files[norm(rel)]
	if f == nil || len(f.Lines) == 0 {
		return nil, false
	}
	if start > len(f.Lines) {
		return nil, false
	}
	maxEnd := utils.Min(len(f.Lines), start+e.cfg.MaxLines-1)
	if end > maxEnd {
		end = maxEnd
	}
	code := strings.Join(f.Lines[start-1:end], "\n")
	code = utils.NormalizeCode(code)

	return &model.ContextRef{
		Path:      rel,
		StartLine: start,
		EndLine:   end,
		Code:      code,
		Kind:      kind,
		Symbol:    symbol,
		Why:       why,
	}, true
}

func dedupRefs(in []*model.ContextRef) []*model.ContextRef {
	seen := make(map[string]struct{}, len(in))
	out := make([]*model.ContextRef, 0, len(in))
	for _, r := range in {
		if r == nil {
			continue
		}
		key := norm(r.Path) + "|" + strconv.Itoa(r.StartLine) + "|" + strconv.Itoa(r.EndLine)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

func stableOrder(in []*model.ContextRef) []*model.ContextRef {
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].Path != in[j].Path {
			return in[i].Path < in[j].Path
		}
		return in[i].StartLine < in[j].StartLine
	})
	return in
}

func norm(p string) string { return filepath.ToSlash(p) }
