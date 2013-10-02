package main

import (
	"code.google.com/p/go.exp/go/types"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func savePackageName(p *build.Package) {
	for _, name := range append(append(append(p.GoFiles, p.IgnoredGoFiles...), p.CgoFiles...), p.TestGoFiles...) {
		path := filepath.Join(p.Dir, name)
		b, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		src := string(b)
		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, "", src, parser.PackageClauseOnly)
		if err != nil {
			panic(err)
		}
		oldName := astFile.Name
		i := fset.Position(oldName.Pos()).Offset
		src = src[:i] + p.Name + src[i+len(oldName.Name):]
		if err := ioutil.WriteFile(path, []byte(src), 0666); err != nil {
			panic(err)
		}
	}

	if pkg, ok := pkgs[p.ImportPath]; ok {
		pkg.Name = p.Name
	}

	// TODO: update all uses?  could get messy with name conflicts.  not that everything has work perfectly.
}

func saveType(t *types.NamedType) {
	w := newWriter(t.Obj)
	defer w.close()

	u := t.Underlying
	walkType(u, func(tt *types.NamedType) {
		if p := tt.Obj.Pkg; p != t.Obj.Pkg {
			if _, ok := w.pkgNames[p]; !ok {
				w.pkgNames[p] = w.name(p.Name)
			}
		}
	})
	w.imports()

	w.write("type %s %s", t.Obj.Name, w.typ(u))
}

func saveFunc(f *funcNode) {
	w := newWriter(f.obj)
	defer w.close()

	for p := range f.pkgRefs {
		w.pkgNames[p] = w.name(p.Name)
	}
	w.imports()
	w.fun(f, map[*port]string{})
}

type writer struct {
	src      *os.File
	pkgNames map[*types.Package]string
	names    map[string]int
	seqID    int
	seqIDs   map[node]int
	nindent  int
}

func newWriter(obj types.Object) *writer {
	pkg := obj.GetPkg()
	src, err := os.Create(fluxPath(obj))
	if err != nil {
		panic(err)
	}
	w := &writer{src, map[*types.Package]string{}, map[string]int{}, 0, map[node]int{}, 0}
	fluxObjs[obj] = true

	w.write("package %s\n\n", pkg.Name)
	for _, obj := range append(types.Universe.Entries, pkg.Scope.Entries...) {
		w.name(obj.GetName())
	}
	return w
}

func (w *writer) write(format string, a ...interface{}) {
	fmt.Fprintf(w.src, format, a...)
}

func (w *writer) indent(format string, a ...interface{}) {
	w.write(strings.Repeat("\t", w.nindent)+format, a...)
}

func (w *writer) close() {
	w.src.Close()
}

func (w *writer) imports() {
	if len(w.pkgNames) == 0 {
		return
	}
	w.write("import (\n")
	for p, id := range w.pkgNames {
		w.write("\t")
		if id != p.Name {
			w.write(id + " ")
		}
		w.write(strconv.Quote(p.Path) + "\n")
	}
	w.write(")\n\n")
}

func (w *writer) fun(f *funcNode, vars map[*port]string) {
	vars, varsCopy := map[*port]string{}, vars
	for k, v := range varsCopy {
		vars[k] = v
	}

	w.write("func ")

	obj := f.obj
	if obj == nil {
		obj = f.output.obj
	}

	// name the results first so we can tell if a param connects directly to a result
	existing := map[string]string{}
	for _, p := range f.outputsNode.ins {
		name := w.name(p.obj.Name)
		if v, ok := vars[p]; ok {
			// there is a connection from an outer block to this result of this func literal
			existing[name] = v
		}
		vars[p] = name
	}
	name := func(p *port) string {
		name := w.name(p.obj.Name)
		for _, c := range p.conns {
			if v, ok := vars[c.dst]; ok {
				existing[v] = name
			} else {
				vars[c.dst] = name
			}
		}
		return name
	}
	params := f.inputsNode.outs
	if _, ok := obj.(method); ok {
		p := params[0]
		params = params[1:]
		w.write("(%s %s) ", name(p), w.typ(p.obj.Type))
	}
	w.write("%s(", obj.GetName())
	for i, p := range params {
		if i > 0 {
			w.write(", ")
		}
		w.write("%s %s", name(p), w.typ(p.obj.Type))
	}
	w.write(") (")
	for i, p := range f.outputsNode.ins {
		if i > 0 {
			w.write(", ")
		}
		w.write("%s %s", vars[p], w.typ(p.obj.Type))
	}
	w.write(") {\n")
	w.nindent++
	w.assignExisting(existing)
	w.nindent--
	w.block(f.funcblk, vars)
	if len(f.outputsNode.ins) > 0 {
		w.indent("\treturn\n")
	}
	w.indent("}\n")
}

func (w *writer) block(b *block, vars map[*port]string) {
	order, ok := b.nodeOrder()
	if !ok {
		fmt.Println("cyclic!")
		return
	}

	vars, varsCopy := map[*port]string{}, vars
	for k, v := range varsCopy {
		vars[k] = v
	}

	w.nindent++

	for c := range b.conns {
		if _, ok := vars[c.dst]; !ok && c.src.node.block() != b {
			name := w.name("v")
			w.indent("var %s %s\n", name, w.typ(c.dst.obj.Type))
			vars[c.dst] = name
		}
	}
	for _, n := range order {
		switch n := n.(type) {
		default:
			args := []string{}
			for _, in := range ins(n) {
				if len(in.conns) > 0 {
					args = append(args, vars[in])
				} else if in.obj.Type != nil {
					args = append(args, w.zero(in.obj.Type))
				}
			}
			results, existing := w.results(n, vars)
			switch n := n.(type) {
			case *callNode:
				if !(n.obj == nil && len(args) == 0) {
					f := ""
					switch obj := n.obj.(type) {
					default:
						f = w.qualifiedName(obj)
					case method:
						f = args[0] + "." + obj.Name
						args = args[1:]
					case nil:
						f = args[0]
						args = args[1:]
					}
					w.indent("")
					if len(results) > 0 {
						w.write(strings.Join(results, ", ") + " := ")
					}
					w.write("%s(%s)", f, strings.Join(args, ", "))
					w.seq(n)
				}
			case *deleteNode:
				if len(n.ins[0].conns) > 0 {
					w.indent("delete(%s, %s)", args[0], args[1])
				}
				w.seq(n)
			case *lenNode:
				if len(results) > 0 && len(n.ins[0].conns) > 0 {
					w.indent("%s := len(%s)", results[0], args[0])
				}
				w.seq(n)
			case *makeNode:
				if len(results) > 0 {
					if len(args) == 2 && len(n.ins[1].conns) == 0 { // array capacity input unconnected
						args = args[:1]
					}
					w.indent("%s := make(%s, %s)\n", results[0], w.typ(*n.typ.typ), strings.Join(args, ", "))
				}
			case *operatorNode:
				c := 0
				for _, p := range n.ins {
					c += len(p.conns)
				}
				if c > 0 && len(results) > 0 {
					// TODO: handle constant expressions
					if n.op == "!" {
						w.indent("%s := !%s\n", results[0], args[0])
					} else {
						w.indent("%s := %s %s %s\n", results[0], args[0], n.op, args[1])
					}
				} else {
					existing = nil
				}
			case *indexNode:
				if n.set {
					w.indent("%s[%s] = %s", args[0], args[1], args[2])
				} else if len(results) > 0 {
					w.indent("%s := %s[%s]", strings.Join(results, ", "), args[0], args[1])
				}
				w.seq(n)
			case *basicLiteralNode:
				if len(results) > 0 {
					val := n.text.GetText()
					switch n.kind {
					case token.STRING:
						val = strconv.Quote(val)
					case token.CHAR:
						val = strconv.QuoteRune([]rune(val)[0])
					}
					w.indent("const %s = %s\n", results[0], val)
				}
			case *valueNode:
				if n.set || len(results) > 0 {
					val := ""
					switch {
					case n.addr:
						val = "&"
					case n.indirect:
						val = "*"
					}
					if n.obj != nil {
						if _, ok := n.obj.(field); ok {
							val += args[0] + "." + n.obj.GetName()
							args = args[1:]
						} else {
							val += w.qualifiedName(n.obj)
						}
					} else {
						val += args[0]
						args = args[1:]
					}
					if n.set {
						w.indent("%s = %s", val, args[0])
					} else {
						switch n.obj.(type) {
						default:
							w.indent("%s := %s", results[0], val)
						case *types.Const:
							w.indent("const %s = %s", results[0], val)
						}
					}
					w.seq(n)
				}
			case *typeAssertNode:
				if len(n.ins[0].conns) > 0 && len(results) > 0 {
					w.indent("%s := %s.(%s)\n", strings.Join(results, ", "), args[0], w.typ(*n.typ.typ))
				}
			case *funcNode:
				if len(results) > 0 {
					w.indent("%s := ", results[0])
					w.fun(n, vars)
				}
			}
			w.assignExisting(existing)
		case *compositeLiteralNode:
			results, existing := w.results(n, vars)
			if len(results) > 0 {
				w.indent("%s := ", results[0])
				t, isPtr := indirect(*n.typ.typ)
				if isPtr {
					w.write("&")
				}
				w.write("%s{", w.typ(t))
				first := true
				for _, in := range n.inputs() {
					if len(in.conns) > 0 {
						if !first {
							w.write(", ")
						}
						first = false
						w.write("%s: %s", in.obj.Name, vars[in])
					}
				}
				w.write("}\n")
				w.assignExisting(existing)
			}
		case *portsNode:
		case *ifNode:
			w.indent("if ")
			if len(n.input.conns) > 0 {
				w.write(vars[n.input])
			} else {
				w.write("false")
			}
			w.write(" {\n")
			w.block(n.trueblk, vars)
			if len(n.falseblk.nodes) > 0 {
				w.indent("} else {\n")
				w.block(n.falseblk, vars)
			}
			w.indent("}")
			w.seq(n)
		case *loopNode:
			w.indent("for")
			results, existing := w.results(n.inputsNode, vars)
			if t := n.input.obj.Type; t != nil {
				switch t.(type) {
				case *types.Basic, *types.NamedType:
					i := ""
					if len(results) > 0 {
						i = results[0]
					} else {
						i = w.name("i")
					}
					w.write(" %s := %s(0); %s < %s; %s++", i, w.typ(t), i, vars[n.input], i)
				case *types.Array, *types.Slice, *types.Map, *types.Chan:
					if len(results) == 0 {
						w.write(" _ =")
					} else {
						w.write(" " + results[0])
						if len(results) == 2 {
							w.write(", " + results[1])
						}
						w.write(" :=")
					}
					w.write(" range " + vars[n.input])
				}
			} else if len(results) > 0 {
				w.write(" %s := 0;; %s++", results[0], results[0])
			}
			w.write(" {\n")
			w.nindent++
			w.assignExisting(existing)
			w.nindent--
			w.block(n.loopblk, vars)
			w.indent("}")
			w.seq(n)
		}
	}

	w.nindent--
}

func (w *writer) results(n node, vars map[*port]string) (results []string, existing map[string]string) {
	existing = map[string]string{}
	any := false
	for _, p := range outs(n) {
		name := "_"
		if len(p.conns) > 0 {
			any = true
			name = w.name(p.obj.GetName())
			for _, c := range p.conns {
				if v, ok := vars[c.dst]; ok {
					existing[v] = name
				} else {
					vars[c.dst] = name
				}
			}
		}
		results = append(results, name)
	}
	if !any {
		return nil, nil
	}
	return
}

func (w *writer) seq(n node) {
	seqIn, seqOut := seqIn(n), seqOut(n)
	in := seqIn != nil && len(seqIn.conns) > 0
	out := seqOut != nil && len(seqOut.conns) > 0
	if in || out {
		w.write(" // ")
		if in {
			for i, c := range seqIn.conns {
				if i > 0 {
					w.write(",")
				}
				w.write(strconv.Itoa(w.seqIDs[c.src.node]))
			}
		}
		w.write(";")
		if out {
			seqID := w.seqID
			w.seqID++
			w.seqIDs[n] = seqID
			w.write(strconv.Itoa(seqID))
		}
	}
	w.write("\n")
}

func (w *writer) assignExisting(m map[string]string) {
	if len(m) > 0 {
		var existingNames, sourceNames []string
		for v1, v2 := range m {
			existingNames = append(existingNames, v1)
			sourceNames = append(sourceNames, v2)
		}
		w.indent("%s = %s\n", strings.Join(existingNames, ", "), strings.Join(sourceNames, ", "))
	}
}

func (w writer) name(s string) string {
	if s == "" || s == "_" {
		s = "x"
	}
	if i, ok := w.names[s]; ok {
		w.names[s]++
		return w.name(s + strconv.Itoa(i))
	}
	w.names[s] = 2
	return s
}

func (w writer) qualifiedName(obj types.Object) string {
	n := obj.GetName()
	if p, ok := w.pkgNames[obj.GetPkg()]; ok {
		return p + "." + n
	}
	return n
}

func (w writer) typ(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		return t.Name
	case *types.NamedType:
		return w.qualifiedName(t.Obj)
	case *types.Pointer:
		return "*" + w.typ(t.Base)
	case *types.Array:
		return fmt.Sprintf("[%d]%s", t.Len, w.typ(t.Elt))
	case *types.Slice:
		return "[]" + w.typ(t.Elt)
	case *types.Map:
		return fmt.Sprintf("map[%s]%s", w.typ(t.Key), w.typ(t.Elt))
	case *types.Chan:
		s := ""
		switch t.Dir {
		case ast.SEND:
			s = "chan<- "
		case ast.RECV:
			s = "<-chan "
		default:
			s = "chan "
		}
		return s + w.typ(t.Elt)
	case *types.Signature:
		return "func" + w.signature(t)
	case *types.Interface:
		s := "interface{"
		for i, m := range t.Methods {
			if i > 0 {
				s += "; "
			}
			s += m.Name + w.signature(m.Type)
		}
		return s + "}"
	case *types.Struct:
		s := "struct{"
		for i, f := range t.Fields {
			if i > 0 {
				s += "; "
			}
			if !f.IsAnonymous && f.Name != "" {
				s += f.Name + " "
			}
			s += w.typ(f.Type)
		}
		return s + "}"
	}
	panic(fmt.Sprintf("unexpected type %#v\n", t))
}

func (w writer) signature(f *types.Signature) string {
	s := w.params(f.Params)
	if len(f.Results) > 0 {
		s += " "
		if len(f.Results) == 1 && f.Results[0].Name == "" {
			return s + w.typ(f.Results[0].Type)
		}
		return s + w.params(f.Results)
	}
	return s
}

func (w writer) params(params []*types.Var) string {
	s := "("
	for i, p := range params {
		if i > 0 {
			s += ", "
		}
		name := p.Name
		if name == "" {
			name = "_"
		}
		s += name + " "
		s += w.typ(p.Type)
	}
	return s + ")"
}

func (w writer) zero(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		switch {
		case t.Info&types.IsBoolean != 0:
			return "false"
		case t.Info&types.IsNumeric != 0:
			return "0"
		case t.Info&types.IsString != 0:
			return `""`
		default:
			return "nil"
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return "nil"
	case *types.Array, *types.Struct:
		return w.typ(t) + "{}"
	case *types.NamedType:
		switch t.Underlying.(type) {
		case *types.Array, *types.Struct:
			return w.typ(t) + "{}"
		}
		return w.zero(t.Underlying)
	}
	panic(fmt.Sprintf("unexpected type %#v\n", t))
}

func walkType(t types.Type, op func(*types.NamedType)) {
	switch t := t.(type) {
	case *types.Basic:
	case *types.NamedType:
		op(t)
	case *types.Pointer:
		walkType(t.Base, op)
	case *types.Array:
		walkType(t.Elt, op)
	case *types.Slice:
		walkType(t.Elt, op)
	case *types.Map:
		walkType(t.Key, op)
		walkType(t.Elt, op)
	case *types.Chan:
		walkType(t.Elt, op)
	case *types.Signature:
		for _, v := range append(t.Params, t.Results...) {
			walkType(v.Type, op)
		}
	case *types.Interface:
		for _, m := range t.Methods {
			walkType(m.Type, op)
		}
	case *types.Struct:
		for _, v := range t.Fields {
			walkType(v.Type, op)
		}
	default:
		panic(fmt.Sprintf("unexpected type %#v\n", t))
	}
}

func fluxPath(obj types.Object) string {
	pkg := obj.GetPkg()
	bp, err := build.Import(pkg.Path, "", build.FindOnly)
	if err != nil {
		panic(err)
	}

	name := obj.GetName()
	if !ast.IsExported(name) { // unexported names are suffixed with "-" to avoid possible conflicts on case-insensitive systems
		name += "-"
	}
	if m, ok := obj.(method); ok {
		t, _ := indirect(m.Type.Recv.Type)
		typeName := t.(*types.NamedType).Obj.Name
		if !ast.IsExported(typeName) {
			typeName += "-"
		}
		name = typeName + "." + name
	}
	return filepath.Join(bp.Dir, name+".flux.go")
}
