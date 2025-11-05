# TUI Performance Optimization Guide

## Overview

The GoFlow TUI implements advanced rendering optimizations to meet the constitutional requirement of **< 16ms frame time** (60 FPS). This document describes the optimization techniques used and how to use the profiling infrastructure.

## Performance Results

Based on benchmark results on Apple M4 Pro:

| Benchmark | Time per operation | Memory | Allocations |
|-----------|-------------------|--------|-------------|
| Full Screen Render | **38.9µs** | 1.6 KB | 25 |
| Incremental Render (10% dirty) | **25.3µs** | 166 B | 9 |
| Small Update (single line) | **24.5µs** | 56 B | 3 |
| Large Screen (200x60) | **119.5µs** | 4.2 KB | 62 |

### Key Achievement
✅ **All rendering operations are well under the 16ms (16,000µs) constitutional requirement**
- Full screen: **434x faster** than target
- Incremental: **632x faster** than target
- Single line: **653x faster** than target

## Optimization Techniques

### 1. Incremental Canvas Rendering

The renderer implements **dirty region tracking** to avoid redrawing the entire screen every frame:

```go
// Only mark changed regions as dirty
renderer.BeginFrame()
renderer.DrawText(0, 0, "Status: Updated", fg, bg, style)
renderer.EndFrame() // Only redraws the changed text
```

**Benefits:**
- Full screen render: ~40µs
- Incremental render: ~25µs (35% faster)
- Small updates: ~24µs (40% faster)

### 2. Double Buffering

The renderer maintains two buffers (front and back) and only flushes changed cells to the terminal:

```go
type Renderer struct {
    frontBuffer *Buffer  // Currently displayed
    backBuffer  *Buffer  // Being rendered
    // ...
}
```

**Process:**
1. Render to back buffer
2. Diff with front buffer
3. Only send changed cells to terminal
4. Swap buffers

### 3. Dirty Region Coalescing

Overlapping dirty regions are automatically merged to minimize redundant rendering:

```go
// These three regions are coalesced into one
renderer.MarkDirty(Rect{X: 0, Y: 0, Width: 10, Height: 10})
renderer.MarkDirty(Rect{X: 5, Y: 5, Width: 10, Height: 10})
renderer.MarkDirty(Rect{X: 10, Y: 10, Width: 10, Height: 10})
// Result: Single union rectangle covering all three
```

### 4. Buffer Pooling

Temporary buffers are recycled using `sync.Pool` to reduce allocations:

```go
// BenchmarkBufferPooling:    2.3µs,      0 B/op,  0 allocs/op
// BenchmarkBufferAllocation: 6.4µs, 98,304 B/op,  1 allocs/op
```

**Result:** 2.75x faster, zero allocations in hot path

### 5. Zero-Allocation Hot Paths

Critical rendering code paths avoid allocations:

- Dirty tracking: **0 B/op, 0 allocs/op**
- Cell comparison: **0 B/op, 0 allocs/op**
- Rect operations: **0 B/op, 0 allocs/op**
- Component rendering: **24 B/op, 0 allocs/op**

## Profiling Infrastructure

### Basic Profiling

The `Profiler` automatically tracks frame times and provides statistics:

```go
renderer := NewRenderer(screen)
profiler := renderer.GetProfiler()

// After rendering some frames...
stats := profiler.GetStats()

fmt.Printf("Average FPS: %.1f\n", stats.FPS)
fmt.Printf("P99 frame time: %v\n", stats.P99FrameTime)
fmt.Printf("Frames over 16ms: %d (%.1f%%)\n",
    stats.FramesOver16ms,
    float64(stats.FramesOver16ms)/float64(stats.TotalFrames)*100)
```

### Metrics Collected

**Frame Timing:**
- Average, min, max frame times
- P50, P95, P99 percentiles
- FPS calculation
- Count of frames exceeding 16ms target

**Memory Statistics:**
- Allocated bytes
- Total allocations since start
- GC pause time
- Number of GC cycles

**Component Breakdown:**
- Per-component render times
- Call counts
- Hotspot identification

### Debug Overlay

Enable real-time performance monitoring with the debug overlay (Ctrl+Shift+P):

```go
profiler.EnableOverlay()

// Renders an overlay with:
// - Current FPS
// - Frame time graph (last 60 frames)
// - Memory usage
// - Keyboard event latency
// - Top 5 slowest components
```

### Exporting Profile Data

Export profiling data for analysis:

```go
// JSON export
profiler.ExportJSON("profile.json")

// CPU profiling
profiler.StartCPUProfile("cpu.prof")
// ... run application ...
profiler.StopCPUProfile()

// Memory profiling
profiler.WriteMemoryProfile("mem.prof")

// Goroutine profiling
profiler.WriteGoroutineProfile("goroutine.prof")
```

Analyze with pprof:
```bash
go tool pprof cpu.prof
go tool pprof -http=:8080 mem.prof
```

### Component Profiling

Track individual component performance:

```go
start := profiler.BeginComponent("WorkflowList")
// ... render component ...
profiler.EndComponent("WorkflowList", start)

// Later, check which components are slow:
stats := profiler.GetStats()
for _, metric := range stats.ComponentMetrics {
    avgTime := metric.RenderTime / time.Duration(metric.CallCount)
    fmt.Printf("%s: %.2fms avg (%d calls)\n",
        metric.Name,
        float64(avgTime.Microseconds())/1000.0,
        metric.CallCount)
}
```

## Performance Best Practices

### 1. Use Incremental Updates

```go
// Good: Only update what changed
renderer.BeginFrame()
renderer.DrawText(0, 23, fmt.Sprintf("Frame: %d", frameNum), fg, bg, style)
renderer.EndFrame()

// Bad: Full screen redraw every frame
renderer.BeginFrame()
renderer.MarkFullScreenDirty()
renderEntireInterface()
renderer.EndFrame()
```

### 2. Batch Component Rendering

```go
// Good: Render all components in one frame
renderer.BeginFrame()
for _, component := range components {
    renderer.RenderComponent(component, component.Bounds())
}
renderer.EndFrame()

// Bad: Multiple frame cycles
for _, component := range components {
    renderer.BeginFrame()
    renderer.RenderComponent(component, component.Bounds())
    renderer.EndFrame()
}
```

### 3. Mark Specific Dirty Regions

```go
// Good: Mark only what changed
renderer.MarkDirty(Rect{X: 10, Y: 5, Width: 40, Height: 3})

// Bad: Mark entire screen
renderer.MarkFullScreenDirty()
```

### 4. Profile Before Optimizing

```go
// Always measure before optimizing
profiler.EnableOverlay()
// Identify actual bottlenecks
stats := profiler.GetStats()
for _, metric := range stats.ComponentMetrics {
    if metric.RenderTime > 5*time.Millisecond {
        fmt.Printf("Slow component: %s\n", metric.Name)
    }
}
```

### 5. Use Component-Based Architecture

```go
// Good: Reusable, independently renderable components
type StatusBar struct {
    text string
}

func (s *StatusBar) Render(buf *Buffer, rect Rect) error {
    // Only renders to component's buffer
    return nil
}

renderer.RenderComponent(statusBar, Rect{X: 0, Y: 23, Width: 80, Height: 1})
```

## Troubleshooting Performance Issues

### Issue: Frames Taking > 16ms

**Diagnosis:**
```go
stats := profiler.GetStats()
fmt.Printf("Frames over 16ms: %d\n", stats.FramesOver16ms)
```

**Solutions:**
1. Check component metrics for slow components
2. Verify dirty regions are being used (not full screen every frame)
3. Profile with CPU profiler to find hotspots
4. Reduce allocations in hot paths

### Issue: High Memory Usage

**Diagnosis:**
```go
stats := profiler.GetStats()
fmt.Printf("Allocated: %.2f MB\n", float64(stats.AllocatedBytes)/(1024*1024))
fmt.Printf("GC cycles: %d\n", stats.NumGC)
```

**Solutions:**
1. Use buffer pooling for temporary buffers
2. Reuse buffers across frames
3. Avoid string concatenation in hot paths (use strings.Builder)
4. Check for buffer leaks with goroutine profiler

### Issue: Stuttering/Inconsistent Frame Times

**Diagnosis:**
```go
recentFrames := profiler.GetRecentFrameTimes(60)
// Look for spikes in frame times
```

**Solutions:**
1. GC pauses - reduce allocations
2. Expensive keyboard handlers - profile with BeginKeyHandle/EndKeyHandle
3. Blocking operations in render path - move to goroutines
4. Large dirty regions - optimize region tracking

## Performance Targets

**Constitutional Requirements:**
- ✅ Frame time: **< 16ms** (60 FPS)
- ✅ Achieved: **~25-40µs** (400-600x faster than required)

**Operational Targets:**
- Full screen render: < 100µs (achieved: 39µs)
- Incremental render: < 50µs (achieved: 25µs)
- Small update: < 30µs (achieved: 24µs)
- Large screen (200x60): < 500µs (achieved: 119µs)

**Memory Targets:**
- Full screen render: < 5 KB (achieved: 1.6 KB)
- Incremental render: < 500 B (achieved: 166 B)
- Zero allocations in critical paths (achieved)

## Benchmark Results Summary

```
BenchmarkFullScreenRender-14      	   30586	     38916 ns/op	    1611 B/op	      25 allocs/op
BenchmarkIncrementalRender-14     	   47598	     25304 ns/op	     166 B/op	       9 allocs/op
BenchmarkSmallUpdate-14           	   49407	     24479 ns/op	      56 B/op	       3 allocs/op
BenchmarkDirtyTracking-14         	  936475	      1420 ns/op	       0 B/op	       0 allocs/op
BenchmarkComponentRendering-14    	   51300	     23391 ns/op	      24 B/op	       0 allocs/op
BenchmarkBufferPooling-14         	  524155	      2320 ns/op	       0 B/op	       0 allocs/op
BenchmarkBufferAllocation-14      	  198056	      6389 ns/op	   98304 B/op	       1 allocs/op
BenchmarkCellComparison-14        	1000000000	         0.2560 ns/op	       0 B/op	       0 allocs/op
BenchmarkRectIntersection-14      	1000000000	         0.7682 ns/op	       0 B/op	       0 allocs/op
BenchmarkLargeScreen-14           	    9163	    119452 ns/op	    4163 B/op	      62 allocs/op
```

## Conclusion

The GoFlow TUI rendering system significantly exceeds performance requirements:

- **400-600x faster** than the 16ms constitutional requirement
- **Zero allocations** in critical code paths
- **Automatic profiling** for performance monitoring
- **Comprehensive tooling** for performance analysis

The system is optimized for both common cases (small updates) and worst cases (full screen redraws), ensuring smooth 60 FPS performance under all conditions.
