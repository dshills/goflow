package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/dshills/goterm"
)

// ScreenInterface defines the methods required from a goterm.Screen
type ScreenInterface interface {
	Size() (width, height int)
	Clear()
	Show() error
	SetCell(x, y int, cell goterm.Cell)
	DrawText(x, y int, text string, fg, bg goterm.Color, style goterm.Style)
}

// Rect represents a rectangular region on screen
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Intersects checks if two rectangles overlap
func (r Rect) Intersects(other Rect) bool {
	return r.X < other.X+other.Width &&
		r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height &&
		r.Y+r.Height > other.Y
}

// Union returns the smallest rectangle containing both rectangles
func (r Rect) Union(other Rect) Rect {
	x1 := min(r.X, other.X)
	y1 := min(r.Y, other.Y)
	x2 := max(r.X+r.Width, other.X+other.Width)
	y2 := max(r.Y+r.Height, other.Y+other.Height)
	return Rect{
		X:      x1,
		Y:      y1,
		Width:  x2 - x1,
		Height: y2 - y1,
	}
}

// Contains checks if a point is within the rectangle
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width &&
		y >= r.Y && y < r.Y+r.Height
}

// Cell represents a single terminal cell with content and styling
type Cell struct {
	Rune  rune
	Fg    goterm.Color
	Bg    goterm.Color
	Style goterm.Style
}

// Equals compares two cells for equality
func (c Cell) Equals(other Cell) bool {
	return c.Rune == other.Rune &&
		c.Fg == other.Fg &&
		c.Bg == other.Bg &&
		c.Style == other.Style
}

// Buffer represents a 2D grid of cells
type Buffer struct {
	Width  int
	Height int
	Cells  []Cell
}

// NewBuffer creates a new buffer with given dimensions
func NewBuffer(width, height int) *Buffer {
	return &Buffer{
		Width:  width,
		Height: height,
		Cells:  make([]Cell, width*height),
	}
}

// Get retrieves a cell at the given coordinates
func (b *Buffer) Get(x, y int) Cell {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return Cell{}
	}
	return b.Cells[y*b.Width+x]
}

// Set updates a cell at the given coordinates
func (b *Buffer) Set(x, y int, cell Cell) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return
	}
	b.Cells[y*b.Width+x] = cell
}

// Clear fills the buffer with empty cells
func (b *Buffer) Clear() {
	for i := range b.Cells {
		b.Cells[i] = Cell{Rune: ' '}
	}
}

// Resize changes the buffer dimensions, preserving content where possible
func (b *Buffer) Resize(width, height int) {
	if width == b.Width && height == b.Height {
		return
	}

	newCells := make([]Cell, width*height)
	for y := 0; y < min(height, b.Height); y++ {
		for x := 0; x < min(width, b.Width); x++ {
			newCells[y*width+x] = b.Cells[y*b.Width+x]
		}
	}

	b.Width = width
	b.Height = height
	b.Cells = newCells
}

// BufferPool manages reusable buffers to reduce allocations
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Buffer{}
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get(width, height int) *Buffer {
	buf := p.pool.Get().(*Buffer)
	if buf.Width != width || buf.Height != height {
		buf.Resize(width, height)
	}
	buf.Clear()
	return buf
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf *Buffer) {
	p.pool.Put(buf)
}

// Renderable represents any component that can be rendered
type Renderable interface {
	Render(buf *Buffer, rect Rect) error
}

// Renderer implements incremental canvas rendering with dirty region tracking
type Renderer struct {
	screen       ScreenInterface
	frontBuffer  *Buffer
	backBuffer   *Buffer
	dirtyRegions []Rect
	bufferPool   *BufferPool
	mu           sync.Mutex
	profiler     *Profiler
	lastRender   time.Time
}

// NewRenderer creates a new optimized renderer
func NewRenderer(screen ScreenInterface) *Renderer {
	width, height := screen.Size()
	return &Renderer{
		screen:       screen,
		frontBuffer:  NewBuffer(width, height),
		backBuffer:   NewBuffer(width, height),
		dirtyRegions: make([]Rect, 0, 16),
		bufferPool:   NewBufferPool(),
		profiler:     NewProfiler(),
		lastRender:   time.Now(),
	}
}

// BeginFrame starts a new frame render cycle
func (r *Renderer) BeginFrame() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear back buffer
	r.backBuffer.Clear()

	// Reset dirty regions for new frame
	r.dirtyRegions = r.dirtyRegions[:0]
}

// MarkDirty marks a rectangular region as needing redraw
func (r *Renderer) MarkDirty(rect Rect) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.markDirtyInternal(rect)
}

// MarkFullScreenDirty marks the entire screen as needing redraw
func (r *Renderer) MarkFullScreenDirty() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dirtyRegions = []Rect{{
		X:      0,
		Y:      0,
		Width:  r.backBuffer.Width,
		Height: r.backBuffer.Height,
	}}
}

// RenderComponent renders a component to the back buffer at the given rect
func (r *Renderer) RenderComponent(component Renderable, rect Rect) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get a temporary buffer from pool for component rendering
	tempBuf := r.bufferPool.Get(rect.Width, rect.Height)
	defer r.bufferPool.Put(tempBuf)

	// Render component to temporary buffer
	if err := component.Render(tempBuf, Rect{X: 0, Y: 0, Width: rect.Width, Height: rect.Height}); err != nil {
		return fmt.Errorf("component render failed: %w", err)
	}

	// Copy component buffer to back buffer at specified position
	for y := 0; y < rect.Height; y++ {
		for x := 0; x < rect.Width; x++ {
			cell := tempBuf.Get(x, y)
			r.backBuffer.Set(rect.X+x, rect.Y+y, cell)
		}
	}

	// Mark this region as dirty
	r.markDirtyInternal(rect)

	return nil
}

// DrawText draws text at the given position in the back buffer
func (r *Renderer) DrawText(x, y int, text string, fg, bg goterm.Color, style goterm.Style) {
	r.mu.Lock()

	for i, ch := range text {
		r.backBuffer.Set(x+i, y, Cell{
			Rune:  ch,
			Fg:    fg,
			Bg:    bg,
			Style: style,
		})
	}

	// Mark the text region as dirty (needs to be done while locked)
	r.markDirtyInternal(Rect{X: x, Y: y, Width: len(text), Height: 1})

	r.mu.Unlock()
}

// markDirtyInternal is the internal version without locking (caller must hold lock)
func (r *Renderer) markDirtyInternal(rect Rect) {
	// Coalesce overlapping dirty regions to minimize rendering
	coalesced := false
	for i := 0; i < len(r.dirtyRegions); i++ {
		if r.dirtyRegions[i].Intersects(rect) {
			r.dirtyRegions[i] = r.dirtyRegions[i].Union(rect)
			coalesced = true

			// Check if this newly enlarged region intersects with others
			for j := i + 1; j < len(r.dirtyRegions); {
				if r.dirtyRegions[i].Intersects(r.dirtyRegions[j]) {
					r.dirtyRegions[i] = r.dirtyRegions[i].Union(r.dirtyRegions[j])
					// Remove the merged region
					r.dirtyRegions = append(r.dirtyRegions[:j], r.dirtyRegions[j+1:]...)
				} else {
					j++
				}
			}
			break
		}
	}

	if !coalesced {
		r.dirtyRegions = append(r.dirtyRegions, rect)
	}
}

// EndFrame completes the render cycle and flushes to screen
// Returns the frame rendering time
func (r *Renderer) EndFrame() (time.Duration, error) {
	start := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Track frame time with profiler (outside lock to avoid recursion)
	defer func() {
		frameTime := time.Since(start)
		token := ProfileToken{FrameID: 0, StartTime: start}
		r.profiler.EndFrame(token, frameTime)
		r.lastRender = time.Now()
	}()

	// If no dirty regions, nothing to do
	if len(r.dirtyRegions) == 0 {
		return time.Since(start), nil
	}

	// Render only dirty regions to screen
	for _, rect := range r.dirtyRegions {
		if err := r.renderDirtyRegion(rect); err != nil {
			return time.Since(start), fmt.Errorf("render dirty region failed: %w", err)
		}
	}

	// Show the screen (flushes terminal buffer)
	if err := r.screen.Show(); err != nil {
		return time.Since(start), fmt.Errorf("screen show failed: %w", err)
	}

	// Swap buffers - back becomes front for next diff
	r.frontBuffer, r.backBuffer = r.backBuffer, r.frontBuffer

	return time.Since(start), nil
}

// renderDirtyRegion renders a specific dirty region to the screen
func (r *Renderer) renderDirtyRegion(rect Rect) error {
	// Clamp rectangle to screen bounds
	x1 := max(0, rect.X)
	y1 := max(0, rect.Y)
	x2 := min(r.backBuffer.Width, rect.X+rect.Width)
	y2 := min(r.backBuffer.Height, rect.Y+rect.Height)

	// Render each cell in the dirty region
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			backCell := r.backBuffer.Get(x, y)
			frontCell := r.frontBuffer.Get(x, y)

			// Only update if cell changed
			if !backCell.Equals(frontCell) {
				// Convert local Cell to goterm.Cell
				gotermCell := goterm.NewCell(backCell.Rune, backCell.Fg, backCell.Bg, backCell.Style)
				r.screen.SetCell(x, y, gotermCell)
			}
		}
	}

	return nil
}

// HandleResize updates buffer sizes when terminal is resized
func (r *Renderer) HandleResize(width, height int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.frontBuffer.Resize(width, height)
	r.backBuffer.Resize(width, height)

	// Mark entire screen as dirty after resize
	r.dirtyRegions = []Rect{{
		X:      0,
		Y:      0,
		Width:  width,
		Height: height,
	}}
}

// GetBufferSize returns the current buffer dimensions
func (r *Renderer) GetBufferSize() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.backBuffer.Width, r.backBuffer.Height
}

// GetProfiler returns the renderer's profiler for metrics collection
func (r *Renderer) GetProfiler() *Profiler {
	return r.profiler
}

// GetLastFrameTime returns the last measured frame time
func (r *Renderer) GetLastFrameTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return time.Since(r.lastRender)
}

// Clear clears the back buffer
func (r *Renderer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backBuffer.Clear()
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
