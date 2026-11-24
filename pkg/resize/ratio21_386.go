//go:build 386

package resize

import "github.com/shinkar94/goframeproc/pkg/simd"

// pshufb masks (XMM, 16 bytes = one 4-px window): [p0, p2] and [p1, p3].
// Same lane layout as the amd64 128-bit masks.
var maskE21_386 = [16]byte{0, 1, 2, 3, 8, 9, 10, 11, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

var maskO21_386 = [16]byte{4, 5, 6, 7, 12, 13, 14, 15, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

// dispatchH21 runs the horizontal 2:1 kernel for dstPx output pixels.
// XMM loads are exact: the iteration consumes two 16-byte windows at +0 and
// +16 (8 source px) and produces 4 output px (16 bytes), advancing 32 src /
// 16 dst — reads stay inside an 8*dstPx row for every full iteration, so
// only the dstPx%4 tail stays scalar (pshufb + PAVGB: measured ×11.5 on a
// Silvermont Atom Z3735F versus the scalar kernel).
func dispatchH21(dstRow, srcRow []byte, dstPx int) {
	if simd.HaveSSSE3() {
		if iters := dstPx / 4; iters > 0 {
			h21SSE386(dstRow, srcRow, iters)
			done := 4 * iters
			h21Scalar(dstRow[done*4:], srcRow[done*8:], dstPx-done)
			return
		}
	}
	h21Scalar(dstRow, srcRow, dstPx)
}

// dispatchH21Nearest is the nearest twin of dispatchH21 (same bounds rules,
// pure pshufb copies).
func dispatchH21Nearest(dstRow, srcRow []byte, dstPx int) {
	if simd.HaveSSSE3() {
		if iters := dstPx / 4; iters > 0 {
			hn21SSE386(dstRow, srcRow, iters)
			done := 4 * iters
			h21NearestScalar(dstRow[done*4:], srcRow[done*8:], dstPx-done)
			return
		}
	}
	h21NearestScalar(dstRow, srcRow, dstPx)
}

// dispatchAvg21 averages rows with pavgb semantics (XMM 64-byte quads on
// SSSE3 — PAVGB is available on every SSSE3 part; scalar remainder).
func dispatchAvg21(dst, a, b []byte) {
	n := len(dst)
	if simd.HaveSSSE3() {
		full := n &^ 63
		if full > 0 {
			avgSSE386(dst[:full], a[:full], b[:full])
		}
		avgScalar(dst[full:], a[full:], b[full:])
		return
	}
	avgScalar(dst, a, b)
}

// h21SSE386 processes `iters` iterations, each: two 16-byte loads (+0, +16),
// 4 output pixels (16 bytes) — see ratio21_386.s.
// Caller guarantees rows >= 32*iters source bytes and >= 16*iters dst bytes.
func h21SSE386(dst, src []byte, iters int)

// hn21SSE386 is the nearest twin of h21SSE386: pure pshufb copies, same bounds.
func hn21SSE386(dst, src []byte, iters int)

// avgSSE386 processes floor(len(dst)/64) 64-byte quads: dst = pavgb(a, b).
// The tail is handled by the caller.
func avgSSE386(dst, a, b []byte)
