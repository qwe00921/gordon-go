package gui

import (
	. "code.google.com/p/gordon-go/util"
	gl "github.com/chsc/gogl/gl21"
)

type View interface {
	base() *ViewBase
	win() *Window

	Add(View)
	Remove(View)
	Close()

	Move(Point)
	SetRect(Rectangle)

	TookKeyFocus()
	LostKeyFocus()

	KeyPress(KeyEvent)
	KeyRelease(KeyEvent)

	Paint()
}

type KeyEvent struct {
	Key                     int
	action                  int
	Repeat                  bool
	Text                    string // only present on Press and Repeat, not Release
	Shift, Ctrl, Alt, Super bool
	Command                 bool // platform-independent command key (Super on OS X, Ctrl elsewhere)
}

type MouserView interface {
	Mouser
	View
}

type Scroller interface {
	Scroll(ScrollEvent)
}

type ScrollerView interface {
	Scroller
	View
}

type ScrollEvent struct {
	Pos, Delta Point
}

type ViewBase struct {
	Self     View
	parent   View
	children []View
	hidden   bool
	rect     Rectangle
	pos      Point
}

func NewView(self View) *ViewBase {
	v := &ViewBase{}
	if self == nil {
		self = v
	}
	v.Self = self
	return v
}

func (v *ViewBase) base() *ViewBase { return v }
func (v *ViewBase) win() *Window {
	if v.parent == nil {
		return nil
	}
	return v.parent.win()
}

func Parent(v View) View       { return v.base().parent }
func NumChildren(v View) int   { return len(v.base().children) }
func Child(v View, i int) View { return v.base().children[i] }
func (v *ViewBase) Add(u View) {
	if Parent(u) != nil {
		Parent(u).Remove(u)
	}
	v.children = append(v.children, u)
	u.base().parent = v.Self
	Repaint(v.Self)
}
func (v *ViewBase) Remove(u View) {
	SliceRemove(&v.children, u)
	u.base().parent = nil
	Repaint(v.Self)
}
func (v *ViewBase) Close() {
	if v.parent != nil {
		v.parent.Remove(v.Self)
	}
}

func Show(v View) { v.base().hidden = false; Repaint(v) }
func Hide(v View) { v.base().hidden = true; Repaint(v) }

func Raise(v View) {
	if Parent(v) != nil {
		p := Parent(v).base()
		for i, view := range p.children {
			if view == v {
				p.children = append(append(p.children[:i], p.children[i+1:]...), view)
				Repaint(p)
				return
			}
		}
	}
}
func Lower(v View) {
	if Parent(v) != nil {
		p := Parent(v).base()
		for i, view := range p.children {
			if view == v {
				p.children = append(p.children[i:i+1], append(p.children[:i], p.children[i+1:]...)...)
				Repaint(p)
				return
			}
		}
	}
}

func Pos(v View) Point           { return v.base().pos }
func (v *ViewBase) Move(p Point) { v.pos = p; Repaint(v.Self) }
func MoveCenter(v View, p Point) { v.Move(p.Sub(Size(v).Div(2))) }
func MoveOrigin(v View, p Point) { v.Move(p.Add(Rect(v).Min)) }

func Rect(v View) Rectangle { return v.base().rect }
func RectInParent(v View) Rectangle {
	r := Rect(v)
	return Rectangle{MapToParent(r.Min, v), MapToParent(r.Max, v)}
}
func Center(v View) Point               { return Rect(v).Min.Add(Size(v).Div(2)) }
func CenterInParent(v View) Point       { return MapToParent(Center(v), v) }
func Size(v View) Point                 { return Rect(v).Size() }
func Width(v View) float64              { return Rect(v).Dx() }
func Height(v View) float64             { return Rect(v).Dy() }
func (v *ViewBase) SetRect(r Rectangle) { v.rect = r; Repaint(v.Self) }
func Pan(v View, p Point) {
	r := Rect(v)
	v.SetRect(r.Add(p.Sub(r.Min)))
}
func Resize(v View, s Point) {
	r := Rect(v)
	r.Max = r.Min.Add(s)
	v.SetRect(r)
}
func ResizeToFit(v View, margin float64) {
	rect := ZR
	for i := 0; i < NumChildren(v); i++ {
		c := Child(v, i)
		r := RectInParent(c)
		if i == 0 {
			rect = r
		} else {
			rect = rect.Union(r)
		}
	}
	v.SetRect(rect.Inset(-margin))
}

func SetKeyFocus(v View) {
	if w := v.win(); w != nil {
		w.setKeyFocus(v)
	}
}
func KeyFocus(v View) View {
	if w := v.win(); w != nil {
		return w.keyFocus
	}
	return nil
}

func (v *ViewBase) TookKeyFocus() {}
func (v *ViewBase) LostKeyFocus() {}

func (v *ViewBase) KeyPress(event KeyEvent) {
	if v.parent != nil {
		v.parent.KeyPress(event)
	}
}
func (v *ViewBase) KeyRelease(event KeyEvent) {
	if v.parent != nil {
		v.parent.KeyRelease(event)
	}
}

func SetMouser(m MouserView, button int) {
	if w := m.win(); w != nil {
		w.setMouser(m, button)
	}
}

func MouseParent(v View, m MouseEvent) {
	for v != nil {
		m.Pos = MapToParent(m.Pos, v)
		v = Parent(v)
		if v, ok := v.(Mouser); ok {
			v.Mouse(m)
			return
		}
	}
}

func Repaint(v View) {
	if w := v.win(); w != nil {
		w.repaint()
	}
}

func (v *ViewBase) paint() {
	if v.hidden {
		return
	}
	gl.PushMatrix()
	defer gl.PopMatrix()
	d := MapToParent(ZP, v)
	gl.Translated(gl.Double(d.X), gl.Double(d.Y), 0)
	v.Self.Paint()
	for _, child := range v.children {
		child.base().paint()
	}
}
func (v ViewBase) Paint() {}

func Do(v View, f func()) {
	w := v.win()
	if w == nil {
		panic("gui.Do called on windowless View")
	}
	w.Do(f)
}

func DoChan(v View) chan<- func() {
	w := v.win()
	if w == nil {
		panic("gui.DoChan called on windowless View")
	}
	return w.do
}

func ViewAt(v View, p Point) View { return viewAtFunc(v, p, func(v View) View { return v }) }
func viewAtFunc(v View, p Point, f func(View) View) View {
	if !p.In(Rect(v)) {
		return nil
	}
	for i := NumChildren(v) - 1; i >= 0; i-- {
		child := Child(v, i)
		view := viewAtFunc(child, MapFromParent(p, child), f)
		if view != nil {
			return view
		}
	}
	return f(v)
}

func MapToParent(p Point, v View) Point {
	return p.Sub(Rect(v).Min).Add(Pos(v))
}

func MapFromParent(p Point, v View) Point {
	return p.Sub(Pos(v)).Add(Rect(v).Min)
}

func Map(p Point, from, to View) Point {
	v := commonParent(from, to)
	if v == nil {
		// It is impossible to map between views without a common parent.
		// Typically (always?), this happens because one of the views has been removed from the tree, probably to be deleted, in which case it is fine to return an incorrect Point.
		return ZP
	}
	for from != v {
		p = MapToParent(p, from)
		from = Parent(from)
	}
	for to != v {
		p = MapFromParent(p, to)
		to = Parent(to)
	}
	return p
}

func commonParent(v1, v2 View) (p View) {
	for ; v1 != nil; v1 = Parent(v1) {
		for v2 := v2; v2 != nil; v2 = Parent(v2) {
			if v1 == v2 {
				return v1
			}
		}
	}
	return nil
}
