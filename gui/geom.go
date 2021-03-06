package gui

import (
	. "math"
	"strconv"
)

// A Point is an X, Y coordinate pair. The axes increase right and down.
type Point struct {
	X, Y float64
}

func (p Point) XY() (float64, float64) { return p.X, p.Y }

// String returns a string representation of p like "(3,4)".
func (p Point) String() string {
	return "(" + strconv.FormatFloat(p.X, 'f', -1, 64) + "," + strconv.FormatFloat(p.Y, 'f', -1, 64) + ")"
}

// Add returns the vector p+q.
func (p Point) Add(q Point) Point {
	return Point{p.X + q.X, p.Y + q.Y}
}

// Sub returns the vector p-q.
func (p Point) Sub(q Point) Point {
	return Point{p.X - q.X, p.Y - q.Y}
}

// Mul returns the vector p*k.
func (p Point) Mul(k float64) Point {
	return Point{p.X * k, p.Y * k}
}

// Div returns the vector p/k.
func (p Point) Div(k float64) Point {
	return Point{p.X / k, p.Y / k}
}

func (p Point) Dot(q Point) float64 {
	return p.X*q.X + p.Y*q.Y
}

func (p Point) Cross(q Point) float64 {
	return p.X*q.Y - p.Y*q.X
}

func (p Point) Len() float64 {
	return Sqrt(p.X*p.X + p.Y*p.Y)
}

func (p Point) Angle() float64 {
	return Atan2(p.Y, p.X)
}

// In returns whether p is in r.
func (p Point) In(r Rectangle) bool {
	return r.Min.X <= p.X && p.X <= r.Max.X &&
		r.Min.Y <= p.Y && p.Y <= r.Max.Y
}

// Mod returns the point q in r such that p.X-q.X is a multiple of r's width
// and p.Y-q.Y is a multiple of r's height.
func (p Point) Mod(r Rectangle) Point {
	w, h := r.Dx(), r.Dy()
	p = p.Sub(r.Min)
	p.X = Remainder(p.X, w)
	if p.X < 0 {
		p.X += w
	}
	p.Y = Remainder(p.Y, h)
	if p.Y < 0 {
		p.Y += h
	}
	return p.Add(r.Min)
}

// Eq returns whether p and q are equal.
func (p Point) Eq(q Point) bool {
	return p.X == q.X && p.Y == q.Y
}

// ZP is the zero Point.
var ZP Point

// Pt is shorthand for Point{X, Y}.
func Pt(X, Y float64) Point {
	return Point{X, Y}
}

// A Rectangle contains the points with Min.X <= X < Max.X, Min.Y <= Y < Max.Y.
// It is well-formed if Min.X <= Max.X and likewise for Y. Points are always
// well-formed. A rectangle's methods always return well-formed outputs for
// well-formed inputs.
type Rectangle struct {
	Min, Max Point
}

// String returns a string representation of r like "(3,4)-(6,5)".
func (r Rectangle) String() string {
	return r.Min.String() + "-" + r.Max.String()
}

// Dx returns r's width.
func (r Rectangle) Dx() float64 {
	return r.Max.X - r.Min.X
}

// Dy returns r's height.
func (r Rectangle) Dy() float64 {
	return r.Max.Y - r.Min.Y
}

// Size returns r's width and height.
func (r Rectangle) Size() Point {
	return Point{
		r.Max.X - r.Min.X,
		r.Max.Y - r.Min.Y,
	}
}

func (r Rectangle) Center() Point {
	return r.Min.Add(r.Size().Div(2))
}

// Add returns the rectangle r translated by p.
func (r Rectangle) Add(p Point) Rectangle {
	return Rectangle{
		Point{r.Min.X + p.X, r.Min.Y + p.Y},
		Point{r.Max.X + p.X, r.Max.Y + p.Y},
	}
}

// Sub returns the rectangle r translated by -p.
func (r Rectangle) Sub(p Point) Rectangle {
	return Rectangle{
		Point{r.Min.X - p.X, r.Min.Y - p.Y},
		Point{r.Max.X - p.X, r.Max.Y - p.Y},
	}
}

// Inset returns the rectangle r inset by n, which may be negative. If either
// of r's dimensions is less than 2*n then an empty rectangle near the center
// of r will be returned.
func (r Rectangle) Inset(n float64) Rectangle {
	if r.Dx() < 2*n {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += n
		r.Max.X -= n
	}
	if r.Dy() < 2*n {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += n
		r.Max.Y -= n
	}
	return r
}

// Intersect returns the largest rectangle contained by both r and s. If the
// two rectangles do not overlap then the zero rectangle will be returned.
func (r Rectangle) Intersect(s Rectangle) Rectangle {
	if r.Min.X < s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y < s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X > s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y > s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	if r.Min.X > r.Max.X || r.Min.Y > r.Max.Y {
		return ZR
	}
	return r
}

// Union returns the smallest rectangle that contains both r and s.
func (r Rectangle) Union(s Rectangle) Rectangle {
	if r.Min.X > s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y > s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X < s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y < s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	return r
}

// Empty returns whether the rectangle contains no points.
func (r Rectangle) Empty() bool {
	return r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y
}

// Eq returns whether r and s are equal.
func (r Rectangle) Eq(s Rectangle) bool {
	return r.Min.X == s.Min.X && r.Min.Y == s.Min.Y &&
		r.Max.X == s.Max.X && r.Max.Y == s.Max.Y
}

// Overlaps returns whether r and s have a non-empty intersection.
func (r Rectangle) Overlaps(s Rectangle) bool {
	return r.Min.X < s.Max.X && s.Min.X < r.Max.X &&
		r.Min.Y < s.Max.Y && s.Min.Y < r.Max.Y
}

// In returns whether every point in r is in s.
func (r Rectangle) In(s Rectangle) bool {
	if r.Empty() {
		return true
	}
	// Note that r.Max is an exclusive bound for r, so that r.In(s)
	// does not require that r.Max.In(s).
	return s.Min.X <= r.Min.X && r.Max.X <= s.Max.X &&
		s.Min.Y <= r.Min.Y && r.Max.Y <= s.Max.Y
}

// Canon returns the canonical version of r. The returned rectangle has minimum
// and maximum coordinates swapped if necessary so that it is well-formed.
func (r Rectangle) Canon() Rectangle {
	if r.Max.X < r.Min.X {
		r.Min.X, r.Max.X = r.Max.X, r.Min.X
	}
	if r.Max.Y < r.Min.Y {
		r.Min.Y, r.Max.Y = r.Max.Y, r.Min.Y
	}
	return r
}

// ZR is the zero Rectangle.
var ZR Rectangle

// PointToLine returns the Point on line segment (x, y) that is nearest to p.
func PointToLine(p, x, y Point) Point {
	xy := y.Sub(x)
	xp := p.Sub(x)
	t := xp.Dot(xy) / xy.Dot(xy)
	switch {
	case t < 0:
		return x
	default:
		return x.Add(xy.Mul(t))
	case t > 1:
		return y
	}
}

// LineToLine returns the Points z and z2 on line segments (p, p2) and (q, q2), respectively,
// such that the distance from z to z2 is minimized.
func LineToLine(p, p2, q, q2 Point) (z, z2 Point) {
	r := p2.Sub(p)
	s := q2.Sub(q)
	rs := r.Cross(s)
	qpr := q.Sub(p).Cross(r)
	if rs == 0 {
		if qpr == 0 {
			// collinear
		} else {
			// parallel
		}
		return ZP, Pt(10000, 10000)
	}

	if u := qpr / rs; u >= 0 && u <= 1 {
		// intersecting
		z = q.Add(s.Mul(u))
		return z, z
	}

	min := MaxFloat64
	for i, P := range []struct{ p, x, y Point }{{p, q, q2}, {p2, q, q2}, {q, p, p2}, {q2, p, p2}} {
		q := PointToLine(P.p, P.x, P.y)
		if len := P.p.Sub(q).Len(); len < min {
			min = len
			if i < 2 {
				z, z2 = P.p, q
			} else {
				z, z2 = q, P.p
			}
		}
	}
	return
}
