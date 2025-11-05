package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/dshills/goterm"
)

// MockScreen implements a minimal goterm.Screen interface for testing
type MockScreen struct {
	width  int
	height int
	cells  map[string]Cell // key is "x,y"
}

func NewMockScreen(width, height int) *MockScreen {
	return &MockScreen{
		width:  width,
		height: height,
		cells:  make(map[string]Cell),
	}
}

func (m *MockScreen) Size() (int, int) {
	return m.width, m.height
}

func (m *MockScreen) Clear() {
	m.cells = make(map[string]Cell)
}

func (m *MockScreen) Show() error {
	return nil
}

func (m *MockScreen) Close() error {
	return nil
}

func (m *MockScreen) SetCell(x, y int, cell goterm.Cell) {
	key := fmt.Sprintf("%d,%d", x, y)
	m.cells[key] = Cell{
		Rune:  cell.Ch,
		Fg:    cell.Fg,
		Bg:    cell.Bg,
		Style: cell.Style,
	}
}

func (m *MockScreen) DrawText(x, y int, text string, fg, bg goterm.Color, style goterm.Style) {
	for i, ch := range text {
		cell := goterm.NewCell(ch, fg, bg, style)
		m.SetCell(x+i, y, cell)
	}
}

// MockRenderable implements a simple test component
type MockRenderable struct {
	text string
}

func (m *MockRenderable) Render(buf *Buffer, rect Rect) error {
	// Simulate component rendering by filling buffer
	for y := 0; y < rect.Height; y++ {
		for x := 0; x < rect.Width; x++ {
			if x < len(m.text) && y == 0 {
				buf.Set(x, y, Cell{Rune: rune(m.text[x])})
			} else {
				buf.Set(x, y, Cell{Rune: ' '})
			}
		}
	}
	return nil
}

// TestRendererBasicOperation tests basic renderer functionality
func TestRendererBasicOperation(t *testing.T) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	// Begin frame
	renderer.BeginFrame()

	// Mark a region dirty and render
	renderer.MarkDirty(Rect{X: 0, Y: 0, Width: 10, Height: 1})
	renderer.DrawText(0, 0, "Hello", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)

	// End frame
	frameTime, err := renderer.EndFrame()
	if err != nil {
		t.Fatalf("EndFrame failed: %v", err)
	}

	if frameTime > 16*time.Millisecond {
		t.Logf("Warning: Frame time %v exceeds 16ms target", frameTime)
	}

	t.Logf("Frame time: %v", frameTime)
}

// TestRendererIncrementalUpdate tests incremental rendering with small changes
func TestRendererIncrementalUpdate(t *testing.T) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	// First frame - full screen
	renderer.BeginFrame()
	renderer.MarkFullScreenDirty()
	for y := 0; y < 24; y++ {
		renderer.DrawText(0, y, "Initial content", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
	}
	_, err := renderer.EndFrame()
	if err != nil {
		t.Fatalf("First frame failed: %v", err)
	}

	// Second frame - small update (should be faster)
	renderer.BeginFrame()
	renderer.DrawText(0, 0, "Updated", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
	frameTime, err := renderer.EndFrame()
	if err != nil {
		t.Fatalf("Second frame failed: %v", err)
	}

	if frameTime > 16*time.Millisecond {
		t.Logf("Warning: Incremental frame time %v exceeds 16ms target", frameTime)
	}

	t.Logf("Incremental frame time: %v", frameTime)
}

// TestRendererDirtyRegionCoalescing tests that overlapping dirty regions are merged
func TestRendererDirtyRegionCoalescing(t *testing.T) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	renderer.BeginFrame()

	// Mark overlapping regions
	renderer.MarkDirty(Rect{X: 0, Y: 0, Width: 10, Height: 10})
	renderer.MarkDirty(Rect{X: 5, Y: 5, Width: 10, Height: 10})
	renderer.MarkDirty(Rect{X: 10, Y: 10, Width: 10, Height: 10})

	renderer.mu.Lock()
	regionCount := len(renderer.dirtyRegions)
	renderer.mu.Unlock()

	// Should coalesce into fewer regions
	if regionCount > 2 {
		t.Logf("Warning: %d dirty regions after coalescing (expected <= 2)", regionCount)
	}

	t.Logf("Dirty regions after coalescing: %d", regionCount)
}

// TestBufferResizing tests buffer resizing behavior
func TestBufferResizing(t *testing.T) {
	buf := NewBuffer(80, 24)

	// Set some content
	buf.Set(5, 5, Cell{Rune: 'X'})

	// Resize larger
	buf.Resize(100, 30)
	if buf.Width != 100 || buf.Height != 30 {
		t.Errorf("Resize failed: got %dx%d, want 100x30", buf.Width, buf.Height)
	}

	// Content should be preserved
	cell := buf.Get(5, 5)
	if cell.Rune != 'X' {
		t.Errorf("Content not preserved after resize: got %c, want X", cell.Rune)
	}

	// Resize smaller
	buf.Resize(40, 12)
	if buf.Width != 40 || buf.Height != 12 {
		t.Errorf("Resize failed: got %dx%d, want 40x12", buf.Width, buf.Height)
	}
}

// TestRectOperations tests rectangle geometry operations
func TestRectOperations(t *testing.T) {
	r1 := Rect{X: 0, Y: 0, Width: 10, Height: 10}
	r2 := Rect{X: 5, Y: 5, Width: 10, Height: 10}
	r3 := Rect{X: 20, Y: 20, Width: 10, Height: 10}

	// Test intersection
	if !r1.Intersects(r2) {
		t.Error("Expected r1 and r2 to intersect")
	}
	if r1.Intersects(r3) {
		t.Error("Expected r1 and r3 not to intersect")
	}

	// Test union
	union := r1.Union(r2)
	if union.X != 0 || union.Y != 0 || union.Width != 15 || union.Height != 15 {
		t.Errorf("Union incorrect: got %+v, want {0 0 15 15}", union)
	}

	// Test contains
	if !r1.Contains(5, 5) {
		t.Error("Expected r1 to contain point (5, 5)")
	}
	if r1.Contains(15, 15) {
		t.Error("Expected r1 not to contain point (15, 15)")
	}
}

// Benchmarks

// BenchmarkFullScreenRender measures full screen rendering performance
func BenchmarkFullScreenRender(b *testing.B) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.BeginFrame()
		renderer.MarkFullScreenDirty()

		// Simulate full screen content
		for y := 0; y < 24; y++ {
			text := fmt.Sprintf("Line %2d: This is a full line of text that fills the width", y)
			renderer.DrawText(0, y, text, goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
		}

		frameTime, err := renderer.EndFrame()
		if err != nil {
			b.Fatal(err)
		}

		if frameTime > 16*time.Millisecond {
			b.Logf("Frame %d exceeded 16ms: %v", i, frameTime)
		}
	}
}

// BenchmarkIncrementalRender measures incremental rendering (10% dirty)
func BenchmarkIncrementalRender(b *testing.B) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	// Initial full render
	renderer.BeginFrame()
	renderer.MarkFullScreenDirty()
	for y := 0; y < 24; y++ {
		renderer.DrawText(0, y, "Initial content", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
	}
	renderer.EndFrame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.BeginFrame()

		// Update only 10% of screen (2-3 lines)
		for y := 0; y < 3; y++ {
			text := fmt.Sprintf("Updated line %d frame %d", y, i)
			renderer.DrawText(0, y, text, goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
		}

		frameTime, err := renderer.EndFrame()
		if err != nil {
			b.Fatal(err)
		}

		if frameTime > 16*time.Millisecond {
			b.Logf("Frame %d exceeded 16ms: %v", i, frameTime)
		}
	}
}

// BenchmarkSmallUpdate measures very small updates (single line)
func BenchmarkSmallUpdate(b *testing.B) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	// Initial render
	renderer.BeginFrame()
	renderer.MarkFullScreenDirty()
	for y := 0; y < 24; y++ {
		renderer.DrawText(0, y, "Static content", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
	}
	renderer.EndFrame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.BeginFrame()

		// Update single line (status bar)
		text := fmt.Sprintf("Frame: %d", i)
		renderer.DrawText(0, 23, text, goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)

		frameTime, err := renderer.EndFrame()
		if err != nil {
			b.Fatal(err)
		}

		if frameTime > 2*time.Millisecond {
			b.Logf("Small update frame %d exceeded 2ms: %v", i, frameTime)
		}
	}
}

// BenchmarkDirtyTracking measures dirty region tracking overhead
func BenchmarkDirtyTracking(b *testing.B) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.BeginFrame()

		// Mark multiple small regions
		for j := 0; j < 10; j++ {
			renderer.MarkDirty(Rect{X: j * 8, Y: j, Width: 8, Height: 1})
		}
	}
}

// BenchmarkComponentRendering measures component rendering performance
func BenchmarkComponentRendering(b *testing.B) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)
	component := &MockRenderable{text: "Test Component"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.BeginFrame()

		// Render multiple components
		for y := 0; y < 20; y += 2 {
			err := renderer.RenderComponent(component, Rect{X: 0, Y: y, Width: 40, Height: 1})
			if err != nil {
				b.Fatal(err)
			}
		}

		_, err := renderer.EndFrame()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBufferPooling measures buffer pool performance
func BenchmarkBufferPooling(b *testing.B) {
	pool := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get(80, 24)
		// Simulate some work
		for y := 0; y < 24; y++ {
			for x := 0; x < 80; x++ {
				buf.Set(x, y, Cell{Rune: 'X'})
			}
		}
		pool.Put(buf)
	}
}

// BenchmarkBufferAllocation measures buffer allocation without pooling
func BenchmarkBufferAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := NewBuffer(80, 24)
		// Simulate some work
		for y := 0; y < 24; y++ {
			for x := 0; x < 80; x++ {
				buf.Set(x, y, Cell{Rune: 'X'})
			}
		}
		_ = buf
	}
}

// BenchmarkCellComparison measures cell equality checking performance
func BenchmarkCellComparison(b *testing.B) {
	cell1 := Cell{Rune: 'A', Fg: goterm.ColorDefault(), Bg: goterm.ColorDefault(), Style: goterm.StyleBold}
	cell2 := Cell{Rune: 'A', Fg: goterm.ColorDefault(), Bg: goterm.ColorDefault(), Style: goterm.StyleBold}
	cell3 := Cell{Rune: 'B', Fg: goterm.ColorDefault(), Bg: goterm.ColorDefault(), Style: goterm.StyleBold}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cell1.Equals(cell2)
		_ = cell1.Equals(cell3)
	}
}

// BenchmarkRectIntersection measures rectangle intersection performance
func BenchmarkRectIntersection(b *testing.B) {
	r1 := Rect{X: 10, Y: 10, Width: 50, Height: 30}
	r2 := Rect{X: 30, Y: 20, Width: 40, Height: 25}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r1.Intersects(r2)
		_ = r1.Union(r2)
	}
}

// BenchmarkLargeScreen measures rendering on large terminal sizes
func BenchmarkLargeScreen(b *testing.B) {
	screen := NewMockScreen(200, 60)
	renderer := NewRenderer(screen)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.BeginFrame()
		renderer.MarkFullScreenDirty()

		for y := 0; y < 60; y++ {
			text := fmt.Sprintf("Line %02d: Lorem ipsum dolor sit amet, consectetur adipiscing elit", y)
			renderer.DrawText(0, y, text, goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
		}

		frameTime, err := renderer.EndFrame()
		if err != nil {
			b.Fatal(err)
		}

		if frameTime > 16*time.Millisecond {
			b.Logf("Large screen frame %d exceeded 16ms: %v", i, frameTime)
		}
	}
}

// TestProfilerIntegration tests profiler integration with renderer
func TestProfilerIntegration(t *testing.T) {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)
	profiler := renderer.GetProfiler()

	// Run several frames
	for i := 0; i < 10; i++ {
		renderer.BeginFrame()
		renderer.MarkFullScreenDirty()
		for y := 0; y < 24; y++ {
			renderer.DrawText(0, y, "Test content", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
		}
		_, err := renderer.EndFrame()
		if err != nil {
			t.Fatalf("Frame %d failed: %v", i, err)
		}
	}

	// Check profiler stats
	stats := profiler.GetStats()
	if stats.TotalFrames != 10 {
		t.Errorf("Expected 10 frames, got %d", stats.TotalFrames)
	}

	if stats.AvgFrameTime == 0 {
		t.Error("Average frame time is zero")
	}

	t.Logf("Profiler stats: avg=%v, p99=%v, FPS=%.1f",
		stats.AvgFrameTime, stats.P99FrameTime, stats.FPS)
}
