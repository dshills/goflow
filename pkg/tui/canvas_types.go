package tui

// Position represents a logical coordinate on the canvas
type Position struct {
	X int
	Y int
}

// Size represents the dimensions of a rectangular area
type Size struct {
	Width  int
	Height int
}

// BoundingBox represents a rectangular area on the canvas
type BoundingBox struct {
	TopLeft Position
	Size    Size
}

// NewPosition creates a new position
func NewPosition(x, y int) Position {
	return Position{X: x, Y: y}
}

// NewSize creates a new size
func NewSize(width, height int) Size {
	return Size{Width: width, Height: height}
}

// NewBoundingBox creates a new bounding box
func NewBoundingBox(x, y, width, height int) BoundingBox {
	return BoundingBox{
		TopLeft: Position{X: x, Y: y},
		Size:    Size{Width: width, Height: height},
	}
}

// Contains checks if a position is within the bounding box
func (bb BoundingBox) Contains(pos Position) bool {
	return pos.X >= bb.TopLeft.X &&
		pos.X < bb.TopLeft.X+bb.Size.Width &&
		pos.Y >= bb.TopLeft.Y &&
		pos.Y < bb.TopLeft.Y+bb.Size.Height
}

// Intersects checks if two bounding boxes intersect
func (bb BoundingBox) Intersects(other BoundingBox) bool {
	// No intersection if one box is completely to the left, right, above, or below the other
	if bb.TopLeft.X >= other.TopLeft.X+other.Size.Width ||
		other.TopLeft.X >= bb.TopLeft.X+bb.Size.Width {
		return false
	}

	if bb.TopLeft.Y >= other.TopLeft.Y+other.Size.Height ||
		other.TopLeft.Y >= bb.TopLeft.Y+bb.Size.Height {
		return false
	}

	return true
}

// BottomRight returns the bottom-right corner position
func (bb BoundingBox) BottomRight() Position {
	return Position{
		X: bb.TopLeft.X + bb.Size.Width - 1,
		Y: bb.TopLeft.Y + bb.Size.Height - 1,
	}
}

// Center returns the center position of the bounding box
func (bb BoundingBox) Center() Position {
	return Position{
		X: bb.TopLeft.X + bb.Size.Width/2,
		Y: bb.TopLeft.Y + bb.Size.Height/2,
	}
}

// ScalePosition scales a position by a zoom factor
func ScalePosition(pos Position, zoomFactor float64) Position {
	return Position{
		X: int(float64(pos.X) * zoomFactor),
		Y: int(float64(pos.Y) * zoomFactor),
	}
}

// UnscalePosition reverses a zoom transformation on a position
func UnscalePosition(pos Position, zoomFactor float64) Position {
	if zoomFactor == 0 {
		return pos
	}
	return Position{
		X: int(float64(pos.X) / zoomFactor),
		Y: int(float64(pos.Y) / zoomFactor),
	}
}

// TranslatePosition applies a viewport offset to a position
func TranslatePosition(pos Position, viewportX, viewportY int) Position {
	return Position{
		X: pos.X - viewportX,
		Y: pos.Y - viewportY,
	}
}

// UntranslatePosition reverses a viewport offset on a position
func UntranslatePosition(pos Position, viewportX, viewportY int) Position {
	return Position{
		X: pos.X + viewportX,
		Y: pos.Y + viewportY,
	}
}

// LogicalToTerminal converts logical coordinates to terminal coordinates
// Applies both viewport translation and zoom scaling
func LogicalToTerminal(logical Position, viewportX, viewportY int, zoomFactor float64) Position {
	// First translate by viewport offset
	translated := TranslatePosition(logical, viewportX, viewportY)
	// Then scale by zoom factor
	return ScalePosition(translated, zoomFactor)
}

// TerminalToLogical converts terminal coordinates to logical coordinates
// Reverses zoom scaling and viewport translation
func TerminalToLogical(terminal Position, viewportX, viewportY int, zoomFactor float64) Position {
	// First unscale from zoom
	unscaled := UnscalePosition(terminal, zoomFactor)
	// Then untranslate viewport offset
	return UntranslatePosition(unscaled, viewportX, viewportY)
}
