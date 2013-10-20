package main

import (
	"code.google.com/p/go.exp/go/types"
)

type indexNode struct {
	*nodeBase
	set           bool
	x, key, inVal *port
	outVal, ok    *port
	addressable   bool
}

func newIndexNode(set bool) *indexNode {
	n := &indexNode{set: set}
	n.nodeBase = newNodeBase(n)
	up := n.updateInputType
	n.x = n.newInput(&types.Var{})
	n.x.connsChanged = up
	n.key = n.newInput(&types.Var{})
	n.key.connsChanged = up
	if set {
		n.inVal = n.newInput(&types.Var{})
		n.inVal.connsChanged = up
		n.text.SetText("[]=")
	} else {
		n.outVal = n.newOutput(&types.Var{})
		n.text.SetText("[]")
	}
	n.addSeqPorts()
	up()
	return n
}

func (n *indexNode) updateInputType() {
	n.addressable = false

	var t, key, elt types.Type
	if len(n.x.conns) > 0 {
		if p := n.x.conns[0].src; p != nil {
			var ptr bool
			t, ptr = indirect(p.obj.Type)
			u := t
			if n, ok := t.(*types.NamedType); ok {
				u = n.Underlying
			}
			key = types.Typ[types.Int]
			switch u := u.(type) {
			case *types.Array:
				elt = u.Elt
				if ptr && !n.set {
					t = p.obj.Type
					elt = &types.Pointer{elt}
					n.addressable = true
				}
			case *types.Slice:
				elt = u.Elt
				if !n.set {
					elt = &types.Pointer{elt}
					n.addressable = true
				}
			case *types.Map:
				key, elt = u.Key, u.Elt
			}
		}
	} else {
		if len(n.key.conns) > 0 {
			if o := n.key.conns[0].src; o != nil {
				key = o.obj.Type
			}
		}
		if n.set && len(n.inVal.conns) > 0 {
			if o := n.inVal.conns[0].src; o != nil {
				elt = o.obj.Type
			}
		}
	}
	if t == nil {
		t = generic{}
	}
	if key == nil {
		key = generic{}
	}
	if elt == nil {
		elt = generic{}
	}

	if !n.set {
		switch t.(type) {
		default:
			if n.ok != nil {
				for _, c := range n.ok.conns {
					c.blk.removeConn(c)
				}
				n.Remove(n.ok)
				n.outs = n.outs[:1]
				n.ok = nil
			}
		case *types.Map:
			if n.ok == nil {
				n.ok = n.newOutput(&types.Var{Name: "ok", Type: types.Typ[types.Bool]})
			}
		}
	}

	n.x.setType(t)
	n.key.setType(key)
	if n.set {
		n.inVal.setType(elt)
	} else {
		n.outVal.setType(elt)
	}
	n.reform()
}
