package main

import (
	."code.google.com/p/gordon-go/gui"
)

type loopNode struct {
	*ViewBase
	AggregateMouseHandler
	blk *block
	input *input
	inputsNode *portsNode
	loopblk *block
	focused bool
}

func newLoopNode(b *block) *loopNode {
	n := &loopNode{}
	n.ViewBase = NewView(n)
	n.AggregateMouseHandler = AggregateMouseHandler{NewClickKeyboardFocuser(n), NewViewDragger(n)}
	n.blk = b
	n.input = newInput(n, &Value{})
	n.input.connsChanged = func() {
		if conns := n.input.conns; len(conns) > 0 {
			if o := conns[0].src; o != nil { n.updateInputType(o.val.typ) }
		} else {
			n.updateInputType(nil)
		}
	}
	n.AddChild(n.input)
	n.loopblk = newBlock(n)
	n.inputsNode = newInputsNode(n.loopblk)
	n.inputsNode.newOutput(&Value{})
	n.loopblk.addNode(n.inputsNode)
	n.AddChild(n.loopblk)
	go n.loopblk.animate()
	
	n.input.MoveCenter(Pt(-2*portSize, 0))
	n.updateInputType(nil)
	return n
}

func (n loopNode) block() *block { return n.blk }
func (n loopNode) inputs() []*input { return []*input{n.input} }
func (n loopNode) outputs() []*output { return nil }
func (n loopNode) inConns() []*connection {
	return append(n.input.conns, n.loopblk.inConns()...)
}
func (n loopNode) outConns() []*connection {
	return n.loopblk.outConns()
}

func (n *loopNode) updateInputType(t Type) {
	in := n.inputsNode
	switch t.(type) {
	case nil, *NamedType, *ChanType:
		if len(in.outs) == 2 {
			in.RemoveChild(in.outs[1])
			in.outs = in.outs[:1]
		}
	case *ArrayType, *SliceType, *MapType:
		if len(in.outs) == 1 {
			in.newOutput(&Value{})
		}
	}
	switch t.(type) {
	case nil:
	case *NamedType:
	case *ArrayType:
	case *SliceType:
	case *MapType:
	case *ChanType:
	}
}

func (n *loopNode) positionBlocks() {
	b := n.loopblk
	y2 := b.Size().Y / 2
	b.Move(Pt(0, -y2))
	n.inputsNode.MoveOrigin(b.Rect().Min.Add(Pt(0, y2)))
	ResizeToFit(n, 0)
}

func (n *loopNode) Move(p Point) {
	n.ViewBase.Move(p)
	for _, c := range n.input.conns { c.reform() }
}

func (n *loopNode) TookKeyboardFocus() { n.focused = true; n.Repaint() }
func (n *loopNode) LostKeyboardFocus() { n.focused = false; n.Repaint() }

func (n *loopNode) KeyPressed(event KeyEvent) {
	switch event.Key {
	case KeyLeft, KeyRight, KeyUp, KeyDown:
		n.blk.outermost().focusNearestView(n, event.Key)
	case KeyEscape:
		n.blk.TakeKeyboardFocus()
	default:
		n.ViewBase.KeyPressed(event)
	}
}

func (n loopNode) Paint() {
	SetColor(map[bool]Color{false:{.5, .5, .5, 1}, true:{.3, .3, .7, 1}}[n.focused])
	DrawLine(ZP, Pt(-2*portSize, 0))
}
