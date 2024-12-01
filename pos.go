package term

type Point struct {
	X int
	Y int
}

type Size struct {
	Width  int
	Height int
}

type Rect struct {
	Point
	Size
}

func PaddRect(r *Rect, vpad, hpad int) {
	r.Y += vpad
	r.X += hpad
	r.Width -= (hpad + hpad)
	r.Height -= (vpad + vpad)
}

func NextLine(point *Point, r Rect) {
	point.Y++
	point.X = r.X
}

func PointInRect(p Point, r Rect) bool {
	p.X -= r.X
	p.Y -= r.Y
	return p.X >= 0 && p.Y >= 0 && p.X < r.Width && p.Y < r.Height
}

func (self *Rect) CutTop(height int) (cut Rect) {
	if height > self.Height {
		height = self.Height
	}
	cut = *self
	cut.Height = height

	self.Y += height
	self.Height -= height

	return cut
}

func (self *Rect) CutBottom(height int) (cut Rect) {
	if height > self.Height {
		height = self.Height
	}
	cut = *self
	cut.Height = height
	cut.Y += self.Height - height

	self.Height -= height

	return cut
}

func (self *Rect) CutLeft(width int) (cut Rect) {
	if width > self.Width {
		width = self.Width
	}
	cut = *self
	cut.Width = width

	self.X += width
	self.Width -= width

	return cut
}

func (self *Rect) CutRight(width int) (cut Rect) {
	if width > self.Width {
		width = self.Width
	}
	cut = *self
	cut.Width = width
	cut.X += self.Width - width

	self.Width -= width

	return cut
}
