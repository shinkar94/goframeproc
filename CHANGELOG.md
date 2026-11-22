# Changelog

## v0.5.2 (uncommitted — under review)

### Changed
- **Toolchain requirement restored to go 1.24.0**: v0.5.1 unintentionally
  bumped the go directive to 1.26.0 (`go mod tidy` picked the newest
  x/sys v0.50.0, which itself requires go 1.26.0). v0.5.2 pins
  `golang.org/x/sys v0.30.0` (HasSSSE3/HasAVX2 on windows/386 included)
  and restores the go 1.24.0 directive. Consumers of v0.5.1 should
  upgrade to v0.5.2.

## v0.5.1 (uncommitted — under review)

### Changed
- **Real CPU feature detection on windows/386** (`pkg/simd/cpu_386.go`): the
  previous `!amd64` stub always reported `false`, while 386-class CPUs
  (e.g. Silvermont Atoms) do support SSSE3/SSE4.2 — SSE paths were
  permanently unavailable on windows/386. Detection now goes through
  `golang.org/x/sys/cpu` (`X86.HasSSSE3` / `X86.HasAVX2`, OSXSAVE + XCR0
  YMM-state included); amd64 keeps its asm-based detection. Prerequisite
  for future 386 SIMD kernels; behavior-neutral for current amd64 paths.

### Added
- New dependency: `golang.org/x/sys` (the package's first external
  dependency — the de-facto standard for OS/CPU interaction in Go).

### Validated
- windows/386 test binary (WOW64 on an i5): `SSSE3=true AVX2=true` —
  detection consistency test against `x/sys/cpu` passes.

## v0.5.0 (uncommitted — under review)

### Added
- **Fixed-ratio 3:2 downscale kernels** (`pkg/resize/ratio32*.go`): libyuv-style specialized bilinear and nearest for the exact 3:2 ratio (1920×1080 → 1280×720, and any 3:2 multiples). At an exact 3:2 the bilinear phases cycle with period 2 (copy / pavgb 50-50 blend), so the per-pixel table gather disappears entirely: source is consumed contiguously with `pshufb` extraction and `VPAVGB` blends (AVX2, scalar fallback on older CPUs).
  - `resize.NewRatio32(srcW, srcH, stride, dstW, dstH) *Ratio32` — returns nil unless the target is exactly 3:2 of the source (callers fall back to `ResizeGenericInto`).
  - `(*Ratio32).Into(dst, src)` — bilinear; `(*Ratio32).IntoNearest(dst, src)` — nearest (pure copies, every third source row is never sampled and skipped).
  - Both are allocation-free per frame after the plan is built.
- Regression tests: byte-identical SIMD vs all-scalar path, ≤2 LSB vs an independent 32-bit exact-ratio reference, nearest vs floor exact-ratio reference, zero-alloc, buffer guards (6 sizes, 6×3 up to 4K).

### Changed
- **CPU feature detection cached at package init** (`pkg/simd/cpu_amd64.go`): `HaveAVX2()`/`HaveSSSE3()` no longer run CPUID per call. CPUID is serializing (~500 ns for the 3 leaves + XGETBV) and dispatchers query it per frame/row — re-querying cost ~1 ms per 1080p frame. Fixes both the new Ratio32 kernels and the existing v0.4.0 conversion dispatch.

### Mapping note
The fixed-ratio kernels use the exact-ratio mapping (`src = 3/2 * dst`), not the edge-to-edge mapping (`x*(srcW-1)/(dstW-1)`) of the generic resize path. Positions drift by up to 0.5 px near the right/bottom edge. Measured on live desktop content: PSNR vs a Catmull-Rom reference 30.3–35.3 dB (ON) vs 34.0–47.7 dB (generic path) depending on content; nearest is unaffected (PSNR identical). No existing API changes its behavior — the generic path keeps edge-to-edge semantics.

### Measurements (i5-12450H, 1080p BGRA→720p RGBA, plan built once, 0 allocs)

| Path | v0.4.0 (scalar resize) | v0.5.0 (Ratio32) |
|---|---|---|
| Bilinear, saving (1 core) | 7.87 ms | **1.56 ms (×5.0)** incl. RGBA swap |
| Bilinear, live stream, weak PC (GOMAXPROCS=2) | 8.57 ms | **1.77 ms (×4.8)** |
| Bilinear, live stream, 12 cores | 3.02 ms | **1.95 ms (×1.5)** — single core |
| Nearest, saving | 2.15 ms | **1.28 ms (×1.7)** incl. RGBA swap |
| Resize kernel only (bilinear) | — | 1.20 ms |

## v0.4.0 (uncommitted — under review)

### Added
- **Native SIMD conversion backend** (`pkg/simd`): AVX2/SSE4 `pshufb` kernels written in Go assembly with runtime CPUID dispatch (AVX2 > SSSE3 > scalar fallback, OS YMM-state check via XGETBV). `convertFormatInto` uses it automatically for frames ≥ 8 pixels; `saving` and `performance` modes both benefit since SIMD works on a single core.
- Regression test `TestConvertSIMDMatchesScalar` (byte-identical output across all vector boundary sizes) and A/B benchmarks.

### Measurements (i5-12450H, 1080p BGRA→RGBA)

| Path | v0.3.0 | v0.4.0 |
|---|---|---|
| Convert, 1 core (scalar ref vs SIMD kernel) | 2.07 ms | **0.58 ms (×3.6)** |
| Live stream 1080p original, GOMAXPROCS=2 (weak PC) | ~2.1 ms | **~1.05 ms (×2.0)** |
| Live stream 1080p original, 12 cores | 1.02 ms | 1.00 ms (parallel scalar was already memory-bound) |

## v0.3.0 (uncommitted — under review)

### Fixed
- **Bilinear corner pixels**: at `fx=fy=0` the weight `w00 = 65536*65536` overflowed uint32, rendering the 4 corner pixels of the frame as transparent black (0x00000000). `w00` is now derived from the weight sum (2^32) with no overflow and no speed regression. Regression test: `TestBilinearCornersNotZero`.

### Added
- **`ProcessFrameInto(dst, ...)`** — process the frame into a caller-provided buffer. With sufficient `dst` capacity there are zero allocations; `ProcessFrame` remains a wrapper (signatures and behavior unchanged).
- `LookupTables` cache keyed by (srcW, srcH, targetW, targetH, stride) — steady-state streaming no longer recomputes tables.

### Changed
- **`convertFormat` (BGRA↔RGBA)** — per-byte swap replaced with 32-bit word operations; in `performance` mode large frames (>480000 pixels) are split across up to 8 worker goroutines. `saving` mode stays single-threaded.

### Measurements (i5-12450H, 1920×1080 BGRA→RGBA, live stream @ 30 FPS)

| Scenario | v0.2.0 | v0.3.0 | Allocations/frame |
|---|---|---|---|
| 1080p original, performance | 4.6–5.0 ms | **1.02 ms** | 8.3 MB → **0** |
| 1080p original, saving | ~4.6 ms | **3.03 ms** | 8.3 MB → **0** |
| 720p bilinear, performance | 3.33 ms | **2.61 ms** | 3.7 MB → **0** |
