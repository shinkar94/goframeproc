# GoFrameProc

**Version:** v0.6.0

High-performance library for video frame processing in Go with support for resizing and pixel format conversion.

This package was specifically designed as a complementary addition to the [github.com/shinkar94/godesktopdup](https://github.com/shinkar94/godesktopdup) package. The main goal of this package is to provide the ability to scale frames for background tasks, such as thumbnail generation, preview creation, or low-resolution monitoring. By scaling frames down before processing or transmission, you can significantly reduce memory usage, network bandwidth, and processing overhead, making it ideal for applications that need to handle screen captures efficiently without impacting system performance. Compression functionality is planned for future releases.

## Features

- **Image resizing** with three interpolation algorithms:
  - `Nearest` — fast, no smoothing
  - `Bilinear` — high quality, with smooth transitions
  - `Box` — area average; the aliasing-free choice for heavy downscales (>2×)
- **Fixed-ratio 3:2 downscale** (`Ratio32`): libyuv-style SIMD kernels for the exact 3:2 ratio (1920×1080 → 1280×720 and any 3:2 multiples) — bilinear in 1.6 ms instead of 7.9 ms on one core, nearest in 1.3 ms (see Benchmarks)
- **Pixel format conversion**: RGBA ↔ BGRA — native SIMD backend (AVX2/SSE `pshufb` with runtime CPU dispatch, scalar fallback), parallel word-swap in `performance` mode
- **Resolution presets**: 320p, 420p, 720p, 1080p, original
- **Two performance strategies**:
  - `SequentialResizeStrategy` — single-threaded (CPU efficient)
  - `ParallelResizeStrategy` — multi-threaded (maximum speed)
- **Zero-allocation processing** via `ProcessFrameInto` with a reused output buffer; the `Ratio32` plan is allocation-free per frame as well

## Installation

```bash
go get github.com/shinkar94/goframeproc@v0.6.0
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

### Fixed-ratio 3:2 downscale (fastest 1080p → 720p path)

```go
import "github.com/shinkar94/goframeproc/pkg/resize"

// Build the plan once (returns nil unless the target is exactly 3:2 of
// the source — then fall back to ResizeGenericInto).
plan := resize.NewRatio32(1920, 1080, 1920*4, 1280, 720)
if plan != nil {
    dstBGRA := make([]byte, 1280*720*4)
    for range ticker.C {
        // Allocation-free per frame: bilinear SIMD (pshufb + VPAVGB) or
        // plan.IntoNearest(dstBGRA, src) for the nearest variant.
        plan.Into(dstBGRA, src)
    }
}
```

Note: the fixed-ratio kernels use the exact-ratio mapping (`src = 3/2 * dst`), not the edge-to-edge mapping of the generic path — positions drift by up to 0.5 px near the right/bottom edge. The generic path keeps its semantics unchanged.

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
  - `Box` — area average; the aliasing-free choice for heavy downscales
    (>2×): preserves thin lines and text that bilinear decimation can skip
    or fringe (ClearType subpixel speckle); ~1.5× slower than bilinear
- **Performance Mode**:
  - `saving` — single core, minimal interference with other tasks (best for weak machines and background jobs)
  - `performance` — all cores (2–4× faster processing)

Reference presets: Weak PC (320p/Nearest/saving), Office PC (720p/Nearest/saving), Balanced (720p/Bilinear/performance), Strong PC (original/Bilinear/performance).

**1080p → 720p fast path**: when the source is exactly 3:2 of the target (e.g. 1920×1080 → 1280×720), the `Ratio32` plan (see below) is 3–5× faster than the generic bilinear path on one core and beats even the multi-threaded generic path while using a single core — ideal for `saving` mode and weak machines.

## Architecture

```
goframeproc/
├── frameproc.go          # Main API: ProcessFrame, ProcessFrameInto, BGRAtoImage
├── frameproc_test.go     # Regression and equivalence tests
├── bench_test.go         # Benchmarks (including GOMAXPROCS=2 weak-machine profile)
├── pkg/
│   ├── constants/
│   │   └── constants.go     # Format and preset constants
│   ├── simd/
│   │   ├── cpu_amd64.go     # Cached CPU feature detection (CPUID + XGETBV at init)
│   │   ├── swap_amd64.go/.s # SIMD BGRA->RGBA swap (AVX2/SSE pshufb, scalar fallback)
│   │   └── swap_other.go    # Portable fallbacks
│   └── resize/
│       ├── resize.go         # Entry point ResizeGeneric / ResizeGenericInto
│       ├── resize-common.go  # Core: interpolation, lookup tables
│       ├── ratio32*.go/.s    # Fixed-ratio 3:2 SIMD kernels (bilinear + nearest)
│       ├── resize_test.go    # Interpolation unit tests
│       ├── ratio32_test.go   # Fixed-ratio regression tests and benchmarks
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
   - BGRA↔RGBA conversion via SIMD `pshufb` swap (AVX2/SSE, since v0.4.0; scalar fallback on older CPUs)
   - `Ratio32` plans hold the intermediate buffer — zero per-frame allocations

6. **Fixed-ratio 3:2 kernels (v0.5.0)**
   - At an exact 3:2 ratio the bilinear phases cycle with period 2 (copy / 50-50 blend), so the per-pixel table gather disappears: source rows are consumed contiguously and processed with vector instructions
   - The nearest variant is pure copies and skips every third source row
   - Same floor/50-50 rounding semantics as the generic paths, byte-exact against the scalar fallback

### Benchmarks

Measured on i5-12450H (12 threads), 1920×1080 BGRA source (see `bench_test.go`, `pkg/resize/ratio32_test.go`):

**Generic path (any resolution):**

| Operation | Sequential | Parallel |
|-----------|------------|----------|
| 1080p → 1080p (format swap only, SIMD) | ~0.6 ms | ~0.6 ms (memory-bound) |
| 1080p → 720p, Nearest | ~2.1 ms | ~0.5 ms |
| 1080p → 720p, Bilinear | ~7.9 ms | ~2.0 ms |

**Fixed-ratio 3:2 path (v0.5.0, 1080p → 720p, plan built once, resize kernel only, 0 allocs):**

| Operation | Single core (AVX2) |
|-----------|--------------------|
| `Ratio32.Into` (bilinear) | **1.28 ms** |
| `Ratio32.IntoNearest` | **0.87 ms** |

In a live 30 FPS stream the whole `ProcessFrameInto` stage takes ~1.0 ms (1080p original, any performance mode) with zero allocations per frame; the 1080p → 720p pipeline with `Ratio32` + swap runs at ~1.9 ms per frame single-core (vs ~8.6 ms scalar on a weak machine).

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

### `resize.NewRatio32` / `(*resize.Ratio32)` — fixed-ratio 3:2

```go
// Returns nil unless dstW/dstH are even, srcW == 3*dstW/2, srcH == 3*dstH/2
// and stride >= srcW*4. Callers should fall back to ResizeGenericInto.
func NewRatio32(srcW, srcH, stride, dstW, dstH int) *Ratio32

// Fixed-ratio 3:2 bilinear (exact-ratio mapping, pavgb rounding).
func (*Ratio32) Into(dst, src []byte)

// Fixed-ratio 3:2 nearest (exact-ratio floor mapping, pure copies).
func (*Ratio32) IntoNearest(dst, src []byte)
```

Build the plan once, reuse for every frame — zero allocations per call. Requires AVX2 for the fast path (automatic scalar fallback elsewhere). See the note about the exact-ratio mapping in the usage section.

### `pkg/simd` — low-level SIMD swap

```go
// BGRA <-> RGBA byte swap: AVX2/SSE pshufb on amd64 (dispatched by the
// caller), scalar fallback on other platforms.
func SwapAVX2(src []byte, dst []byte)
func SwapSSE(src []byte, dst []byte)

// CPU feature reports (cached at package init).
func HaveSSSE3() bool
func HaveAVX2() bool
```

Used internally by `ProcessFrameInto`; exported for reuse in custom pipelines. `SwapAVX2`/`SwapSSE` process the aligned prefix of the buffers — handle the tail with a scalar swap.

### Conversion utilities

```go
// BGRA buffer -> image.RGBA
func BGRAtoImage(buffer []byte, width, height int) (*image.RGBA, error)

// RGBA buffer -> image.RGBA (optimized copy)
func RGBAtoImage(buffer []byte, width, height int) (*image.RGBA, error)
```

## License

MIT - see [LICENSE](LICENSE) file for details.
