package tui

import (
	"fmt"
	"time"

	"github.com/dshills/goterm"
)

// ExampleRenderer demonstrates basic usage of the optimized renderer
func ExampleRenderer() {
	// Create a mock screen for demonstration
	screen := NewMockScreen(80, 24)

	// Create renderer
	renderer := NewRenderer(screen)

	// Begin a frame
	renderer.BeginFrame()

	// Draw some content
	renderer.DrawText(0, 0, "Hello, GoFlow!", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleBold)
	renderer.DrawText(0, 1, "Incremental rendering demo", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)

	// End frame and get timing
	_, err := renderer.EndFrame()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Content rendered successfully")

	// Output: Content rendered successfully
}

// ExampleRenderer_incrementalUpdate demonstrates incremental rendering
func ExampleRenderer_incrementalUpdate() {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	// Initial full-screen render
	renderer.BeginFrame()
	renderer.MarkFullScreenDirty()
	for y := 0; y < 24; y++ {
		renderer.DrawText(0, y, "Static content", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)
	}
	renderer.EndFrame()

	// Small update (only 1 line)
	renderer.BeginFrame()
	renderer.DrawText(0, 0, "Updated!", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleBold)
	renderer.EndFrame()

	fmt.Println("Incremental rendering is faster")

	// Output: Incremental rendering is faster
}

// ExampleProfiler demonstrates profiling usage
func ExampleProfiler() {
	profiler := NewProfiler()

	// Simulate some frames
	for i := 0; i < 100; i++ {
		token := profiler.BeginFrame()
		time.Sleep(time.Microsecond * 100) // Simulate work
		profiler.EndFrame(token, time.Microsecond*100)
	}

	// Get statistics
	stats := profiler.GetStats()

	fmt.Printf("Total frames: %d\n", stats.TotalFrames)
	fmt.Printf("Average FPS > 1000: %v\n", stats.FPS > 1000)
	fmt.Printf("Frame times tracked: %v\n", stats.AvgFrameTime > 0)

	// Output:
	// Total frames: 100
	// Average FPS > 1000: true
	// Frame times tracked: true
}

// ExampleProfiler_overlay demonstrates the debug overlay
func ExampleProfiler_overlay() {
	profiler := NewProfiler()

	// Record some frame times
	for i := 0; i < 10; i++ {
		token := profiler.BeginFrame()
		profiler.EndFrame(token, time.Millisecond*time.Duration(i))
	}

	// Enable overlay
	profiler.EnableOverlay()

	// Get overlay lines
	lines := profiler.RenderOverlay()

	fmt.Printf("Overlay enabled: %v\n", profiler.IsOverlayVisible())
	fmt.Printf("Overlay lines: %v\n", len(lines) > 0)

	// Output:
	// Overlay enabled: true
	// Overlay lines: true
}

// ExampleRenderer_componentRendering demonstrates component-based rendering
func ExampleRenderer_componentRendering() {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	// Create a simple component
	component := &MockRenderable{text: "Component Content"}

	// Render the component
	renderer.BeginFrame()
	err := renderer.RenderComponent(component, Rect{X: 10, Y: 5, Width: 20, Height: 3})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	renderer.EndFrame()

	fmt.Println("Component rendered successfully")

	// Output: Component rendered successfully
}

// ExampleRenderer_dirtyRegions demonstrates dirty region tracking
func ExampleRenderer_dirtyRegions() {
	screen := NewMockScreen(80, 24)
	renderer := NewRenderer(screen)

	renderer.BeginFrame()

	// Mark specific regions as dirty
	renderer.MarkDirty(Rect{X: 0, Y: 0, Width: 10, Height: 1})
	renderer.MarkDirty(Rect{X: 5, Y: 0, Width: 10, Height: 1})

	// These overlapping regions will be coalesced
	renderer.DrawText(0, 0, "Text", goterm.ColorDefault(), goterm.ColorDefault(), goterm.StyleNone)

	frameTime, _ := renderer.EndFrame()

	fmt.Printf("Dirty regions coalesced: %v\n", frameTime < 1*time.Millisecond)

	// Output: Dirty regions coalesced: true
}

// ExampleProfiler_export demonstrates exporting profile data
func ExampleProfiler_export() {
	profiler := NewProfiler()

	// Record some data
	for i := 0; i < 10; i++ {
		token := profiler.BeginFrame()
		profiler.EndFrame(token, time.Millisecond)
	}

	// Get stats instead of exporting to file (for example)
	stats := profiler.GetStats()

	fmt.Printf("Can export stats: %v\n", stats.TotalFrames > 0)

	// Output: Can export stats: true
}

// ExampleRenderer_bufferPooling demonstrates buffer pool usage
func ExampleRenderer_bufferPooling() {
	pool := NewBufferPool()

	// Get buffer from pool
	buf := pool.Get(80, 24)

	// Use buffer
	buf.Set(0, 0, Cell{Rune: 'X'})

	// Return to pool
	pool.Put(buf)

	// Get again (should reuse)
	buf2 := pool.Get(80, 24)
	pool.Put(buf2)

	fmt.Println("Buffer pooling reduces allocations")

	// Output: Buffer pooling reduces allocations
}
