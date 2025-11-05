package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"sync"
	"time"
)

// ProfileToken represents a profiling session for a single frame
type ProfileToken struct {
	FrameID   uint64
	StartTime time.Time
}

// ComponentMetrics tracks performance metrics for a specific component
type ComponentMetrics struct {
	Name       string
	RenderTime time.Duration
	CallCount  int
}

// ProfileStats provides statistical analysis of profiling data
type ProfileStats struct {
	// Frame timing statistics
	TotalFrames    int
	AvgFrameTime   time.Duration
	MinFrameTime   time.Duration
	MaxFrameTime   time.Duration
	P50FrameTime   time.Duration
	P95FrameTime   time.Duration
	P99FrameTime   time.Duration
	FPS            float64
	FramesOver16ms int // Frames exceeding 60 FPS target

	// Render timing
	AvgRenderTime time.Duration
	MinRenderTime time.Duration
	MaxRenderTime time.Duration

	// Keyboard handling
	AvgKeyHandleTime time.Duration
	MaxKeyHandleTime time.Duration
	TotalKeyEvents   int

	// Memory statistics
	AllocatedBytes   uint64
	TotalAllocations uint64
	GCPauses         time.Duration
	NumGC            uint32

	// Component breakdown
	ComponentMetrics []ComponentMetrics

	// Time range
	ProfileStartTime time.Time
	ProfileDuration  time.Duration
}

// Profiler collects and analyzes TUI performance metrics
type Profiler struct {
	// Frame timing
	frameTimes     []time.Duration
	renderTimes    []time.Duration
	keyHandleTimes []time.Duration

	// Frame tracking
	currentFrameID  uint64
	startTime       time.Time
	lastFrameTime   time.Time
	framesOver16ms  int
	totalKeyEvents  int
	maxFrameHistory int

	// Component profiling
	componentMetrics map[string]*ComponentMetrics
	componentTimings []struct {
		component string
		duration  time.Duration
		timestamp time.Time
	}

	// Memory tracking
	memStatsStart runtime.MemStats
	lastMemStats  runtime.MemStats

	// Display overlay
	showOverlay    bool
	overlayEnabled bool

	// Mutex for thread safety
	mu sync.RWMutex

	// Profile capture
	cpuProfileFile *os.File
	memProfileFile *os.File
	capturing      bool
}

// NewProfiler creates a new performance profiler
func NewProfiler() *Profiler {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &Profiler{
		frameTimes:       make([]time.Duration, 0, 1000),
		renderTimes:      make([]time.Duration, 0, 1000),
		keyHandleTimes:   make([]time.Duration, 0, 1000),
		componentMetrics: make(map[string]*ComponentMetrics),
		componentTimings: make([]struct {
			component string
			duration  time.Duration
			timestamp time.Time
		}, 0, 1000),
		startTime:       time.Now(),
		lastFrameTime:   time.Now(),
		maxFrameHistory: 1000,
		memStatsStart:   memStats,
		lastMemStats:    memStats,
		overlayEnabled:  false,
		showOverlay:     false,
	}
}

// BeginFrame starts profiling a new frame
func (p *Profiler) BeginFrame() ProfileToken {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentFrameID++
	token := ProfileToken{
		FrameID:   p.currentFrameID,
		StartTime: time.Now(),
	}

	return token
}

// EndFrame completes frame profiling and records metrics
func (p *Profiler) EndFrame(token ProfileToken, frameTime time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Record frame time
	p.frameTimes = append(p.frameTimes, frameTime)
	if len(p.frameTimes) > p.maxFrameHistory {
		p.frameTimes = p.frameTimes[1:]
	}

	// Track frames exceeding 16ms (60 FPS target)
	if frameTime > 16*time.Millisecond {
		p.framesOver16ms++
	}

	p.lastFrameTime = time.Now()

	// Update memory stats periodically (every 60 frames to reduce overhead)
	if p.currentFrameID%60 == 0 {
		runtime.ReadMemStats(&p.lastMemStats)
	}
}

// BeginRender starts profiling a render operation
func (p *Profiler) BeginRender() time.Time {
	return time.Now()
}

// EndRender completes render profiling
func (p *Profiler) EndRender(start time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	renderTime := time.Since(start)
	p.renderTimes = append(p.renderTimes, renderTime)
	if len(p.renderTimes) > p.maxFrameHistory {
		p.renderTimes = p.renderTimes[1:]
	}
}

// BeginKeyHandle starts profiling keyboard input handling
func (p *Profiler) BeginKeyHandle() time.Time {
	return time.Now()
}

// EndKeyHandle completes keyboard handling profiling
func (p *Profiler) EndKeyHandle(start time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	handleTime := time.Since(start)
	p.keyHandleTimes = append(p.keyHandleTimes, handleTime)
	if len(p.keyHandleTimes) > p.maxFrameHistory {
		p.keyHandleTimes = p.keyHandleTimes[1:]
	}
	p.totalKeyEvents++
}

// BeginComponent starts profiling a specific component render
func (p *Profiler) BeginComponent(name string) time.Time {
	return time.Now()
}

// EndComponent completes component profiling
func (p *Profiler) EndComponent(name string, start time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	duration := time.Since(start)

	// Update component metrics
	metric, exists := p.componentMetrics[name]
	if !exists {
		metric = &ComponentMetrics{Name: name}
		p.componentMetrics[name] = metric
	}
	metric.RenderTime += duration
	metric.CallCount++

	// Record timing
	p.componentTimings = append(p.componentTimings, struct {
		component string
		duration  time.Duration
		timestamp time.Time
	}{
		component: name,
		duration:  duration,
		timestamp: time.Now(),
	})

	if len(p.componentTimings) > p.maxFrameHistory {
		p.componentTimings = p.componentTimings[1:]
	}
}

// GetStats computes and returns comprehensive profiling statistics
func (p *Profiler) GetStats() ProfileStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := ProfileStats{
		TotalFrames:      len(p.frameTimes),
		ProfileStartTime: p.startTime,
		ProfileDuration:  time.Since(p.startTime),
		FramesOver16ms:   p.framesOver16ms,
		TotalKeyEvents:   p.totalKeyEvents,
	}

	// Calculate frame time statistics
	if len(p.frameTimes) > 0 {
		stats.AvgFrameTime = p.average(p.frameTimes)
		stats.MinFrameTime = p.minimum(p.frameTimes)
		stats.MaxFrameTime = p.maximum(p.frameTimes)
		stats.P50FrameTime = p.percentile(p.frameTimes, 0.50)
		stats.P95FrameTime = p.percentile(p.frameTimes, 0.95)
		stats.P99FrameTime = p.percentile(p.frameTimes, 0.99)

		// Calculate FPS based on average frame time
		if stats.AvgFrameTime > 0 {
			stats.FPS = float64(time.Second) / float64(stats.AvgFrameTime)
		}
	}

	// Calculate render time statistics
	if len(p.renderTimes) > 0 {
		stats.AvgRenderTime = p.average(p.renderTimes)
		stats.MinRenderTime = p.minimum(p.renderTimes)
		stats.MaxRenderTime = p.maximum(p.renderTimes)
	}

	// Calculate keyboard handling statistics
	if len(p.keyHandleTimes) > 0 {
		stats.AvgKeyHandleTime = p.average(p.keyHandleTimes)
		stats.MaxKeyHandleTime = p.maximum(p.keyHandleTimes)
	}

	// Memory statistics
	stats.AllocatedBytes = p.lastMemStats.Alloc
	stats.TotalAllocations = p.lastMemStats.Mallocs - p.memStatsStart.Mallocs
	stats.NumGC = p.lastMemStats.NumGC - p.memStatsStart.NumGC
	stats.GCPauses = time.Duration(p.lastMemStats.PauseTotalNs - p.memStatsStart.PauseTotalNs)

	// Component metrics
	stats.ComponentMetrics = make([]ComponentMetrics, 0, len(p.componentMetrics))
	for _, metric := range p.componentMetrics {
		stats.ComponentMetrics = append(stats.ComponentMetrics, *metric)
	}
	// Sort by total render time (descending)
	sort.Slice(stats.ComponentMetrics, func(i, j int) bool {
		return stats.ComponentMetrics[i].RenderTime > stats.ComponentMetrics[j].RenderTime
	})

	return stats
}

// GetRecentFrameTimes returns the last N frame times (up to maxFrameHistory)
func (p *Profiler) GetRecentFrameTimes(n int) []time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if n > len(p.frameTimes) {
		n = len(p.frameTimes)
	}

	result := make([]time.Duration, n)
	copy(result, p.frameTimes[len(p.frameTimes)-n:])
	return result
}

// EnableOverlay enables the debug overlay display
func (p *Profiler) EnableOverlay() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.overlayEnabled = true
	p.showOverlay = true
}

// DisableOverlay disables the debug overlay display
func (p *Profiler) DisableOverlay() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.showOverlay = false
	p.overlayEnabled = false
}

// ToggleOverlay toggles the debug overlay display
func (p *Profiler) ToggleOverlay() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.overlayEnabled {
		p.showOverlay = !p.showOverlay
	}
}

// IsOverlayVisible returns whether the overlay is currently visible
func (p *Profiler) IsOverlayVisible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.showOverlay
}

// RenderOverlay generates overlay text for display (returns lines of text)
func (p *Profiler) RenderOverlay() []string {
	stats := p.GetStats()
	recentFrames := p.GetRecentFrameTimes(60)

	lines := []string{
		"=== Performance Overlay ===",
		fmt.Sprintf("FPS: %.1f (avg: %.2fms, p99: %.2fms)",
			stats.FPS,
			float64(stats.AvgFrameTime.Microseconds())/1000.0,
			float64(stats.P99FrameTime.Microseconds())/1000.0),
		fmt.Sprintf("Frames: %d total, %d over 16ms (%.1f%%)",
			stats.TotalFrames,
			stats.FramesOver16ms,
			float64(stats.FramesOver16ms)/float64(stats.TotalFrames)*100),
		fmt.Sprintf("Memory: %.2f MB allocated, %d GCs",
			float64(stats.AllocatedBytes)/(1024*1024),
			stats.NumGC),
		fmt.Sprintf("Keys: %d events, avg: %.2fms",
			stats.TotalKeyEvents,
			float64(stats.AvgKeyHandleTime.Microseconds())/1000.0),
		"",
		"Recent Frame Times (last 60):",
		p.renderFrameGraph(recentFrames, 50),
		"",
	}

	// Add top components if any
	if len(stats.ComponentMetrics) > 0 {
		lines = append(lines, "Top Components:")
		for i, metric := range stats.ComponentMetrics {
			if i >= 5 {
				break
			}
			avgTime := metric.RenderTime / time.Duration(metric.CallCount)
			lines = append(lines, fmt.Sprintf("  %s: %.2fms avg (%d calls)",
				metric.Name,
				float64(avgTime.Microseconds())/1000.0,
				metric.CallCount))
		}
	}

	return lines
}

// renderFrameGraph creates a simple ASCII graph of frame times
func (p *Profiler) renderFrameGraph(frameTimes []time.Duration, width int) string {
	if len(frameTimes) == 0 {
		return ""
	}

	// Find max frame time for scaling
	maxTime := time.Duration(0)
	for _, t := range frameTimes {
		if t > maxTime {
			maxTime = t
		}
	}

	if maxTime == 0 {
		return ""
	}

	// Create graph (height of 5 rows)
	graph := make([][]rune, 5)
	for i := range graph {
		graph[i] = make([]rune, width)
		for j := range graph[i] {
			graph[i][j] = ' '
		}
	}

	// Plot frame times
	step := len(frameTimes) / width
	if step == 0 {
		step = 1
	}

	for i := 0; i < width && i*step < len(frameTimes); i++ {
		t := frameTimes[i*step]
		height := int(float64(t) / float64(maxTime) * 4)
		if height > 4 {
			height = 4
		}

		// Fill from bottom up
		for h := 4; h > 4-height && h >= 0; h-- {
			if t > 16*time.Millisecond {
				graph[h][i] = '█' // Red/highlight for slow frames
			} else {
				graph[h][i] = '▓'
			}
		}
	}

	// Convert to string
	result := ""
	for _, row := range graph {
		result += string(row) + "\n"
	}

	return result
}

// ExportJSON exports profiling data to JSON format
func (p *Profiler) ExportJSON(filename string) error {
	stats := p.GetStats()

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create profile file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stats); err != nil {
		return fmt.Errorf("failed to encode profile data: %w", err)
	}

	return nil
}

// StartCPUProfile begins CPU profiling
func (p *Profiler) StartCPUProfile(filename string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.capturing {
		return fmt.Errorf("profiling already in progress")
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile: %w", err)
	}

	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	p.cpuProfileFile = file
	p.capturing = true
	return nil
}

// StopCPUProfile ends CPU profiling
func (p *Profiler) StopCPUProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.capturing || p.cpuProfileFile == nil {
		return fmt.Errorf("no CPU profiling in progress")
	}

	pprof.StopCPUProfile()
	if err := p.cpuProfileFile.Close(); err != nil {
		return fmt.Errorf("failed to close CPU profile: %w", err)
	}

	p.cpuProfileFile = nil
	p.capturing = false
	return nil
}

// WriteMemoryProfile writes a memory profile snapshot
func (p *Profiler) WriteMemoryProfile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create memory profile: %w", err)
	}
	defer file.Close()

	runtime.GC() // Get up-to-date statistics
	if err := pprof.WriteHeapProfile(file); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}

	return nil
}

// WriteGoroutineProfile writes a goroutine profile snapshot
func (p *Profiler) WriteGoroutineProfile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create goroutine profile: %w", err)
	}
	defer file.Close()

	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return fmt.Errorf("goroutine profile not available")
	}

	if err := profile.WriteTo(file, 2); err != nil {
		return fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	return nil
}

// Reset clears all profiling data
func (p *Profiler) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.frameTimes = p.frameTimes[:0]
	p.renderTimes = p.renderTimes[:0]
	p.keyHandleTimes = p.keyHandleTimes[:0]
	p.componentMetrics = make(map[string]*ComponentMetrics)
	p.componentTimings = p.componentTimings[:0]
	p.framesOver16ms = 0
	p.totalKeyEvents = 0
	p.startTime = time.Now()
	p.lastFrameTime = time.Now()
	runtime.ReadMemStats(&p.memStatsStart)
	p.lastMemStats = p.memStatsStart
}

// Helper functions for statistics

func (p *Profiler) average(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func (p *Profiler) minimum(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func (p *Profiler) maximum(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func (p *Profiler) percentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Create a sorted copy
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
