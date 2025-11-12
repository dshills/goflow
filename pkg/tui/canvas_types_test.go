package tui

import (
	"testing"
)

func TestPosition(t *testing.T) {
	pos := NewPosition(10, 20)
	if pos.X != 10 || pos.Y != 20 {
		t.Errorf("NewPosition failed: got (%d, %d), want (10, 20)", pos.X, pos.Y)
	}
}

func TestSize(t *testing.T) {
	size := NewSize(30, 40)
	if size.Width != 30 || size.Height != 40 {
		t.Errorf("NewSize failed: got (%d, %d), want (30, 40)", size.Width, size.Height)
	}
}

func TestBoundingBox_Contains(t *testing.T) {
	tests := []struct {
		name     string
		box      BoundingBox
		pos      Position
		expected bool
	}{
		{
			name:     "point inside box",
			box:      NewBoundingBox(10, 10, 20, 20),
			pos:      Position{X: 15, Y: 15},
			expected: true,
		},
		{
			name:     "point on top-left corner",
			box:      NewBoundingBox(10, 10, 20, 20),
			pos:      Position{X: 10, Y: 10},
			expected: true,
		},
		{
			name:     "point outside box (left)",
			box:      NewBoundingBox(10, 10, 20, 20),
			pos:      Position{X: 5, Y: 15},
			expected: false,
		},
		{
			name:     "point outside box (right)",
			box:      NewBoundingBox(10, 10, 20, 20),
			pos:      Position{X: 30, Y: 15},
			expected: false,
		},
		{
			name:     "point outside box (above)",
			box:      NewBoundingBox(10, 10, 20, 20),
			pos:      Position{X: 15, Y: 5},
			expected: false,
		},
		{
			name:     "point outside box (below)",
			box:      NewBoundingBox(10, 10, 20, 20),
			pos:      Position{X: 15, Y: 30},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.box.Contains(tt.pos)
			if result != tt.expected {
				t.Errorf("Contains(%v) = %v, want %v", tt.pos, result, tt.expected)
			}
		})
	}
}

func TestBoundingBox_Intersects(t *testing.T) {
	tests := []struct {
		name     string
		box1     BoundingBox
		box2     BoundingBox
		expected bool
	}{
		{
			name:     "overlapping boxes",
			box1:     NewBoundingBox(10, 10, 20, 20),
			box2:     NewBoundingBox(20, 20, 20, 20),
			expected: true,
		},
		{
			name:     "touching boxes (edge to edge)",
			box1:     NewBoundingBox(10, 10, 20, 20),
			box2:     NewBoundingBox(30, 10, 20, 20),
			expected: false,
		},
		{
			name:     "separated boxes",
			box1:     NewBoundingBox(10, 10, 20, 20),
			box2:     NewBoundingBox(50, 50, 20, 20),
			expected: false,
		},
		{
			name:     "one box inside another",
			box1:     NewBoundingBox(10, 10, 40, 40),
			box2:     NewBoundingBox(20, 20, 10, 10),
			expected: true,
		},
		{
			name:     "identical boxes",
			box1:     NewBoundingBox(10, 10, 20, 20),
			box2:     NewBoundingBox(10, 10, 20, 20),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.box1.Intersects(tt.box2)
			if result != tt.expected {
				t.Errorf("Intersects() = %v, want %v", result, tt.expected)
			}

			// Intersection should be symmetric
			result2 := tt.box2.Intersects(tt.box1)
			if result2 != tt.expected {
				t.Errorf("Intersects() (reversed) = %v, want %v", result2, tt.expected)
			}
		})
	}
}

func TestBoundingBox_BottomRight(t *testing.T) {
	box := NewBoundingBox(10, 10, 20, 20)
	br := box.BottomRight()

	expectedX := 29 // 10 + 20 - 1
	expectedY := 29 // 10 + 20 - 1

	if br.X != expectedX || br.Y != expectedY {
		t.Errorf("BottomRight() = (%d, %d), want (%d, %d)", br.X, br.Y, expectedX, expectedY)
	}
}

func TestBoundingBox_Center(t *testing.T) {
	tests := []struct {
		name     string
		box      BoundingBox
		expected Position
	}{
		{
			name:     "even dimensions",
			box:      NewBoundingBox(10, 10, 20, 20),
			expected: Position{X: 20, Y: 20}, // 10 + 20/2
		},
		{
			name:     "odd dimensions",
			box:      NewBoundingBox(0, 0, 21, 21),
			expected: Position{X: 10, Y: 10}, // 0 + 21/2 (integer division)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			center := tt.box.Center()
			if center.X != tt.expected.X || center.Y != tt.expected.Y {
				t.Errorf("Center() = (%d, %d), want (%d, %d)",
					center.X, center.Y, tt.expected.X, tt.expected.Y)
			}
		})
	}
}

func TestScalePosition(t *testing.T) {
	tests := []struct {
		name       string
		pos        Position
		zoomFactor float64
		expected   Position
	}{
		{
			name:       "100% zoom (no scaling)",
			pos:        Position{X: 10, Y: 20},
			zoomFactor: 1.0,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "200% zoom",
			pos:        Position{X: 10, Y: 20},
			zoomFactor: 2.0,
			expected:   Position{X: 20, Y: 40},
		},
		{
			name:       "50% zoom",
			pos:        Position{X: 10, Y: 20},
			zoomFactor: 0.5,
			expected:   Position{X: 5, Y: 10},
		},
		{
			name:       "150% zoom",
			pos:        Position{X: 10, Y: 20},
			zoomFactor: 1.5,
			expected:   Position{X: 15, Y: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScalePosition(tt.pos, tt.zoomFactor)
			if result.X != tt.expected.X || result.Y != tt.expected.Y {
				t.Errorf("ScalePosition() = (%d, %d), want (%d, %d)",
					result.X, result.Y, tt.expected.X, tt.expected.Y)
			}
		})
	}
}

func TestUnscalePosition(t *testing.T) {
	tests := []struct {
		name       string
		pos        Position
		zoomFactor float64
		expected   Position
	}{
		{
			name:       "100% zoom (no scaling)",
			pos:        Position{X: 10, Y: 20},
			zoomFactor: 1.0,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "reverse 200% zoom",
			pos:        Position{X: 20, Y: 40},
			zoomFactor: 2.0,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "reverse 50% zoom",
			pos:        Position{X: 5, Y: 10},
			zoomFactor: 0.5,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "zero zoom (edge case)",
			pos:        Position{X: 10, Y: 20},
			zoomFactor: 0.0,
			expected:   Position{X: 10, Y: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnscalePosition(tt.pos, tt.zoomFactor)
			if result.X != tt.expected.X || result.Y != tt.expected.Y {
				t.Errorf("UnscalePosition() = (%d, %d), want (%d, %d)",
					result.X, result.Y, tt.expected.X, tt.expected.Y)
			}
		})
	}
}

func TestTranslatePosition(t *testing.T) {
	tests := []struct {
		name      string
		pos       Position
		viewportX int
		viewportY int
		expected  Position
	}{
		{
			name:      "no offset",
			pos:       Position{X: 10, Y: 20},
			viewportX: 0,
			viewportY: 0,
			expected:  Position{X: 10, Y: 20},
		},
		{
			name:      "positive offset",
			pos:       Position{X: 10, Y: 20},
			viewportX: 5,
			viewportY: 5,
			expected:  Position{X: 5, Y: 15},
		},
		{
			name:      "negative offset",
			pos:       Position{X: 10, Y: 20},
			viewportX: -5,
			viewportY: -5,
			expected:  Position{X: 15, Y: 25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslatePosition(tt.pos, tt.viewportX, tt.viewportY)
			if result.X != tt.expected.X || result.Y != tt.expected.Y {
				t.Errorf("TranslatePosition() = (%d, %d), want (%d, %d)",
					result.X, result.Y, tt.expected.X, tt.expected.Y)
			}
		})
	}
}

func TestUntranslatePosition(t *testing.T) {
	tests := []struct {
		name      string
		pos       Position
		viewportX int
		viewportY int
		expected  Position
	}{
		{
			name:      "no offset",
			pos:       Position{X: 10, Y: 20},
			viewportX: 0,
			viewportY: 0,
			expected:  Position{X: 10, Y: 20},
		},
		{
			name:      "reverse positive offset",
			pos:       Position{X: 5, Y: 15},
			viewportX: 5,
			viewportY: 5,
			expected:  Position{X: 10, Y: 20},
		},
		{
			name:      "reverse negative offset",
			pos:       Position{X: 15, Y: 25},
			viewportX: -5,
			viewportY: -5,
			expected:  Position{X: 10, Y: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UntranslatePosition(tt.pos, tt.viewportX, tt.viewportY)
			if result.X != tt.expected.X || result.Y != tt.expected.Y {
				t.Errorf("UntranslatePosition() = (%d, %d), want (%d, %d)",
					result.X, result.Y, tt.expected.X, tt.expected.Y)
			}
		})
	}
}

func TestLogicalToTerminal(t *testing.T) {
	tests := []struct {
		name       string
		logical    Position
		viewportX  int
		viewportY  int
		zoomFactor float64
		expected   Position
	}{
		{
			name:       "no transformation",
			logical:    Position{X: 10, Y: 20},
			viewportX:  0,
			viewportY:  0,
			zoomFactor: 1.0,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "viewport offset only",
			logical:    Position{X: 10, Y: 20},
			viewportX:  5,
			viewportY:  5,
			zoomFactor: 1.0,
			expected:   Position{X: 5, Y: 15},
		},
		{
			name:       "zoom only",
			logical:    Position{X: 10, Y: 20},
			viewportX:  0,
			viewportY:  0,
			zoomFactor: 2.0,
			expected:   Position{X: 20, Y: 40},
		},
		{
			name:       "both viewport and zoom",
			logical:    Position{X: 10, Y: 20},
			viewportX:  5,
			viewportY:  5,
			zoomFactor: 2.0,
			expected:   Position{X: 10, Y: 30}, // (10-5)*2 = 10, (20-5)*2 = 30
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LogicalToTerminal(tt.logical, tt.viewportX, tt.viewportY, tt.zoomFactor)
			if result.X != tt.expected.X || result.Y != tt.expected.Y {
				t.Errorf("LogicalToTerminal() = (%d, %d), want (%d, %d)",
					result.X, result.Y, tt.expected.X, tt.expected.Y)
			}
		})
	}
}

func TestTerminalToLogical(t *testing.T) {
	tests := []struct {
		name       string
		terminal   Position
		viewportX  int
		viewportY  int
		zoomFactor float64
		expected   Position
	}{
		{
			name:       "no transformation",
			terminal:   Position{X: 10, Y: 20},
			viewportX:  0,
			viewportY:  0,
			zoomFactor: 1.0,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "reverse viewport offset only",
			terminal:   Position{X: 5, Y: 15},
			viewportX:  5,
			viewportY:  5,
			zoomFactor: 1.0,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "reverse zoom only",
			terminal:   Position{X: 20, Y: 40},
			viewportX:  0,
			viewportY:  0,
			zoomFactor: 2.0,
			expected:   Position{X: 10, Y: 20},
		},
		{
			name:       "reverse both viewport and zoom",
			terminal:   Position{X: 10, Y: 30},
			viewportX:  5,
			viewportY:  5,
			zoomFactor: 2.0,
			expected:   Position{X: 10, Y: 20}, // 10/2+5 = 10, 30/2+5 = 20
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TerminalToLogical(tt.terminal, tt.viewportX, tt.viewportY, tt.zoomFactor)
			if result.X != tt.expected.X || result.Y != tt.expected.Y {
				t.Errorf("TerminalToLogical() = (%d, %d), want (%d, %d)",
					result.X, result.Y, tt.expected.X, tt.expected.Y)
			}
		})
	}
}

// Test round-trip conversions
func TestCoordinateConversionRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		logical    Position
		viewportX  int
		viewportY  int
		zoomFactor float64
	}{
		{
			name:       "100% zoom no offset",
			logical:    Position{X: 100, Y: 200},
			viewportX:  0,
			viewportY:  0,
			zoomFactor: 1.0,
		},
		{
			name:       "200% zoom with offset",
			logical:    Position{X: 100, Y: 200},
			viewportX:  50,
			viewportY:  50,
			zoomFactor: 2.0,
		},
		{
			name:       "Even coordinates with 50% zoom",
			logical:    Position{X: 100, Y: 200},
			viewportX:  50,
			viewportY:  50,
			zoomFactor: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Logical -> Terminal -> Logical should give original position
			// Note: Due to integer rounding, there may be ±1 pixel precision loss
			terminal := LogicalToTerminal(tt.logical, tt.viewportX, tt.viewportY, tt.zoomFactor)
			result := TerminalToLogical(terminal, tt.viewportX, tt.viewportY, tt.zoomFactor)

			// Allow for ±1 pixel rounding error due to integer division
			deltaX := abs(result.X - tt.logical.X)
			deltaY := abs(result.Y - tt.logical.Y)

			if deltaX > 1 || deltaY > 1 {
				t.Errorf("Round-trip conversion failed: got (%d, %d), want (%d, %d) (delta: %d, %d)",
					result.X, result.Y, tt.logical.X, tt.logical.Y, deltaX, deltaY)
			}
		})
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
