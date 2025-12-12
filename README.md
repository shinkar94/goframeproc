# GoFrameProc

**Version:** v0.1.0

High-performance library for video frame processing in Go with support for resizing and pixel format conversion.

This package was specifically designed as a complementary addition to the [github.com/shinkar94/godesktopdup](https://github.com/shinkar94/godesktopdup) package. The main goal of this package is to provide the ability to scale frames for background tasks, such as thumbnail generation, preview creation, or low-resolution monitoring. By scaling frames down before processing or transmission, you can significantly reduce memory usage, network bandwidth, and processing overhead, making it ideal for applications that need to handle screen captures efficiently without impacting system performance. Compression functionality is planned for future releases.

## Features

- **Image resizing** with two interpolation algorithms:
  - `Nearest` — fast, no smoothing
  - `Bilinear` — high quality, with smooth transitions
- **Pixel format conversion**: RGBA ↔ BGRA
- **Resolution presets**: 320p, 420p, 720p, 1080p, original
- **Two performance strategies**:
  - `SequentialResizeStrategy` — single-threaded (CPU efficient)
  - `ParallelResizeStrategy` — multi-threaded (maximum speed)

## Installation

```bash
go get github.com/shinkar94/frameproc@v0.1.0
```

## Usage

### Basic frame resizing

```go
package main

import (
    "frameproc"
    "frameproc/pkg/constants"
    "frameproc/pkg/resize"
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

### Direct use of resize package

```go
import "frameproc/pkg/resize"

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

## Architecture

```
frameproc/
├── frameproc.go          # Main API: ProcessFrame, BGRAtoImage
├── pkg/
│   ├── constants/
│   │   └── constants.go  # Format and preset constants
│   └── resize/
│       ├── resize.go         # Entry point ResizeGeneric
│       ├── resize-common.go  # Core: interpolation, lookup tables
│       └── types.go          # Types and interfaces
```

## Performance

### Optimizations

1. **Precomputed LookupTables**
   - All coordinate mappings are computed once
   - Including byte offsets to avoid multiplications in hot loops
   - In parallel mode, tables are reused by all workers

2. **Fixed-point arithmetic (16.16)**
   - Bilinear interpolation without float operations in hot loops
   - ~30% faster than float version

3. **Bounds check elimination**
   - Hints to Go compiler to eliminate bounds checks
   - Optimized checks only at image boundaries

4. **Separate functions for Nearest/Bilinear**
   - Avoid switch on every pixel
   - Better CPU branch prediction

5. **Minimal allocations**
   - Buffers allocated once
   - No intermediate copies

### Benchmarks (estimated)

| Operation | Sequential | Parallel (8 cores) |
|-----------|------------|-------------------|
| 1080p → 720p, Nearest | ~2ms | ~0.4ms |
| 1080p → 720p, Bilinear | ~6ms | ~1.2ms |
| 1080p → 320p, Bilinear | ~2ms | ~0.5ms |

### Recommendations for mode selection

- **`PerformanceMode: "saving"`** — for background tasks when resource efficiency is important
- **`PerformanceMode: "performance"`** — for real-time applications (streaming, recording)

## API Reference

### `ProcessFrame`

```go
func ProcessFrame(buffer []byte, width, height int, stride int, opts CompressionOptions) ([]byte, error)
```

Main frame processing function. Performs resize and/or format conversion.

**Returned errors:**
- `ErrEmptyBuffer` — empty input buffer
- `ErrInvalidSize` — invalid width/height
- `ErrBufferSizeMismatch` — buffer size doesn't match resolution

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

### `resize.ResizeGeneric`

```go
func ResizeGeneric(src []byte, srcW, srcH, stride, targetW, targetH int, 
    mode Mode, format string, resizeStrategy ResizeStrategy, inputFormat string) []byte
```

Low-level resize function with full parameter control.

### Conversion utilities

```go
// BGRA buffer -> image.RGBA
func BGRAtoImage(buffer []byte, width, height int) (*image.RGBA, error)

// RGBA buffer -> image.RGBA (optimized copy)
func RGBAtoImage(buffer []byte, width, height int) (*image.RGBA, error)
```

## License

MIT - see [LICENSE](LICENSE) file for details.
