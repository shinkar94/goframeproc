# Changelog

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
