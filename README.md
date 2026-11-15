# GoFrameProc

**Version:** v0.3.0

High-performance library for video frame processing in Go with support for resizing and pixel format conversion.

This package was specifically designed as a complementary addition to the [github.com/shinkar94/godesktopdup](https://github.com/shinkar94/godesktopdup) package. The main goal of this package is to provide the ability to scale frames for background tasks, such as thumbnail generation, preview creation, or low-resolution monitoring. By scaling frames down before processing or transmission, you can significantly reduce memory usage, network bandwidth, and processing overhead, making it ideal for applications that need to handle screen captures efficiently without impacting system performance. Compression functionality is planned for future releases.

## Features

- **Image resizing** with two interpolation algorithms:
  - `Nearest` — fast, no smoothing
  - `Bilinear` — high quality, with smooth transitions
- **Pixel format conversion**: RGBA ↔ BGRA (32-bit word swap, parallel in `performance` mode)
- **Resolution presets**: 320p, 420p, 720p, 1080p, original
- **Two performance strategies**:
  - `SequentialResizeStrategy` — single-threaded (CPU efficient)
  - `ParallelResizeStrategy` — multi-threaded (maximum speed)
- **Zero-allocation processing** via `ProcessFrameInto` with a reused output buffer

## Installation

```bash
go get github.com/shinkar94/goframeproc@v0.3.0
```

## Usage

### Basic frame resizing

```go
package main

import (
    "github.com/shinkar94/goframeproc"
    "github.com/shinkar94/goframeproc/pkg/constants"
    "github.com/shinkar94/goframeproc/pkg/resize"
)

func main() {
    // Source BGRA buffer (e.g., from screen capture)
    buffer := make([]byte, 1920*1080*4)

    opts := frameproc.CompressionOptions{
        InputFormat:     constants.FormatBGRA,
        OutputFormat:    constants.FormatRGBA,
        Resolution:      constants.Res720p,
        Mode:            resize.Bilinear,
        PerformanceMode: "performance", // or "saving"
    }

    result, err := frameproc.ProcessFrame(buffer, 1920, 1080, 1920*4, opts)
    if err != nil {
        panic(err)
    }

    // result contains a 1280x720 frame in RGBA format
}
```

### Zero-allocation processing (recommended for streaming loops)

```go
// Allocate once, reuse for every frame
dst := make([]byte, 1920*1080*4)

for range ticker.C {
    // No allocations as long as dst is large enough for the target size
    result, err := frameproc.ProcessFrameInto(dst, buffer, 1920, 1080, 1920*4, opts)
    if err != nil {
        panic(err)
    }
    // result is a slice of dst — valid until the next call
}
```

### Direct use of resize package

```go
import "github.com/shinkar94/goframeproc/pkg/resize"

// Single-threaded resize
result := resize.ResizeGeneric(
    src,           // source buffer
    1920, 1080,    // source size
    1920*4,        // stride
    1280, 720,     // target size
    resize.Bilinear,
    "RGBA",        // output format
    resize.SequentialResizeStrategy{},
    "BGRA",        // input format
)
```

## Resolution Presets

| Preset | Landscape | Portrait |
|--------|-----------|----------|
| 320p   | 568×320   | 320×568  |
| 420p   | 740×420   | 420×740  |
| 720p   | 1280×720  | 720×1280 |
| 1080p  | 1920×1080 | 1080×1920|

## Choosing modes

The general tradeoff: **smaller output frame → less CPU, memory and bandwidth, but worse small-detail readability. Closer to original → better quality and higher load.**

- **Resolution** — output frame size (see presets above)
- **Resize Mode**:
  - `Nearest` — fastest, jagged edges on downscale
  - `Bilinear` — smooth result, ~3× slower
- **Performance Mode**:
  - `saving` — single core, minimal interference with other tasks (best for weak machines and background jobs)
  - `performance` — all cores (2–4× faster processing)

Reference presets: Weak PC (320p/Nearest/saving), Office PC (720p/Nearest/saving), Balanced (720p/Bilinear/performance), Strong PC (original/Bilinear/performance).

## Architecture

```
goframeproc/
├── frameproc.go          # Main API: ProcessFrame, ProcessFrameInto, BGRAtoImage
├── frameproc_test.go     # Regression and equivalence tests
├── bench_test.go         # Benchmarks (including GOMAXPROCS=2 weak-machine profile)
├── pkg/
│   ├── constants/
│   │   └── constants.go  # Format and preset constants
│   └── resize/
│       ├── resize.go         # Entry point ResizeGeneric / ResizeGenericInto
│       ├── resize-common.go  # Core: interpolation, lookup tables
│       ├── resize_test.go    # Interpolation unit tests
│       └── types.go          # Types and interfaces
```

## Performance

### Optimizations

1. **Precomputed LookupTables** (cached per size combination)
   - All coordinate mappings are computed once
   - Including byte offsets to avoid multiplications in hot loops
   - In parallel mode, tables are reused by all workers

2. **Fixed-point arithmetic (16.16)**
   - Bilinear interpolation without float operations in hot loops
   - Weight-sum trick in `bilinearInterp`: w00 is derived from 2^32 instead of multiplying, which keeps corners correct at zero overflow cost

3. **Bounds check elimination**
   - Hints to Go compiler to eliminate bounds checks
   - Optimized checks only at image boundaries

4. **Separate functions for Nearest/Bilinear**
   - Avoid switch on every pixel
   - Better CPU branch prediction

5. **Zero allocations in steady state**
   - `ProcessFrameInto` + reused output buffer
   - BGRA↔RGBA conversion via 32-bit word swaps (parallel up to 8 workers in `performance` mode)

### Benchmarks

Measured on i5-12450H (12 threads), 1920×1080 BGRA → RGBA frame (see `bench_test.go`):

| Operation | Sequential | Parallel | Sequential (reused dst) |
|-----------|------------|----------|------------------------|
| 1080p → 1080p (format swap only) | ~3.0 ms | ~1.0 ms | ~2.8 ms (0 allocs) |
| 1080p → 720p, Nearest | ~2.1 ms | ~0.5 ms | 0 allocs |
| 1080p → 720p, Bilinear | ~7.9 ms | ~2.0 ms | ~7.9 ms (0 allocs) |

In a live 30 FPS stream the whole `ProcessFrameInto` stage takes ~1.0 ms (1080p original, performance mode) with zero allocations per frame.

### Recommendations for mode selection

- **`PerformanceMode: "saving"`** — for background tasks when resource efficiency is important
- **`PerformanceMode: "performance"`** — for real-time applications (streaming, recording)

## API Reference

### `ProcessFrame`

```go
func ProcessFrame(buffer []byte, width, height int, stride int, opts CompressionOptions) ([]byte, error)
```

Main frame processing function. Performs resize and/or format conversion. Allocates a new output buffer per call.

**Returned errors:**
- `ErrEmptyBuffer` — empty input buffer
- `ErrInvalidSize` — invalid width/height
- `ErrBufferSizeMismatch` — buffer size doesn't match resolution

### `ProcessFrameInto`

```go
func ProcessFrameInto(dst []byte, buffer []byte, width, height int, stride int, opts CompressionOptions) ([]byte, error)
```

Same as `ProcessFrame`, but writes into the caller-provided buffer. If `dst` has enough capacity it is reused (no allocation); otherwise a new buffer is allocated. Returns the used slice of `dst`.

### `CompressionOptions`

```go
type CompressionOptions struct {
    OutputFormat    string                     // "RGBA" or "BGRA"
    InputFormat     string                     // "RGBA" or "BGRA"
    Quality         int                        // 1-100 (reserved)
    Resolution      constants.ResolutionPreset // "320p", "720p", etc.
    Mode            resize.Mode                // Nearest or Bilinear
    PerformanceMode string                     // "saving" or "performance"
}
```

### `resize.ResizeGeneric` / `resize.ResizeGenericInto`

```go
func ResizeGeneric(src []byte, srcW, srcH, stride, targetW, targetH int,
    mode Mode, format string, resizeStrategy ResizeStrategy, inputFormat string) []byte

func ResizeGenericInto(dst []byte, src []byte, srcW, srcH, stride, targetW, targetH int,
    mode Mode, format string, resizeStrategy ResizeStrategy, inputFormat string) []byte
```

Low-level resize functions with full parameter control. `ResizeGenericInto` writes into the provided buffer.

### Conversion utilities

```go
// BGRA buffer -> image.RGBA
func BGRAtoImage(buffer []byte, width, height int) (*image.RGBA, error)

// RGBA buffer -> image.RGBA (optimized copy)
func RGBAtoImage(buffer []byte, width, height int) (*image.RGBA, error)
```

## License

MIT - see [LICENSE](LICENSE) file for details.
