//go:build amd64

package resize

import "github.com/shinkar94/goframeproc/pkg/simd"

// pshufb masks (per 128-bit lane): extract the even/odd source pixels of an
// 8-pixel window (32 bytes) into lane0 as [p0,p2,p4,p6] / [p1,p3,p5,p7].
var maskE21 = [32]byte{
	0, 1, 2, 3, 8, 9, 10, 11, 16, 17, 18, 19, 24, 25, 26, 27,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
}

var maskO21 = [32]byte{
	4, 5, 6, 7, 12, 13, 14, 15, 20, 21, 22, 23, 28, 29, 30, 31,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
}

// dispatchH21 runs the horizontal 2:1 kernel for dstPx output pixels.
// The AVX2 kernel advances 32 source bytes per iteration while reading up
// to 47 past its start (second 32-byte load at +16), so the final
// iteration would spill past the row — the tail stays scalar (the h32AVX2
// discipline).
func dispatchH21(dstRow, srcRow []byte, dstPx int) {
	if simd.HaveAVX2() {
		if iters := dstPx / 4; iters > 1 {
			h21AVX2(dstRow, srcRow, iters-1)
			done := 4 * (iters - 1)
			h21Scalar(dstRow[done*4:], srcRow[done*8:], dstPx-done)
			return
		}
	}
	h21Scalar(dstRow, srcRow, dstPx)
}

// dispatchH21Nearest is the nearest twin of dispatchH21 (same bounds rules).
func dispatchH21Nearest(dstRow, srcRow []byte, dstPx int) {
	if simd.HaveAVX2() {
		if iters := dstPx / 4; iters > 1 {
			hn21AVX2(dstRow, srcRow, iters-1)
			done := 4 * (iters - 1)
			h21NearestScalar(dstRow[done*4:], srcRow[done*8:], dstPx-done)
			return
		}
	}
	h21NearestScalar(dstRow, srcRow, dstPx)
}

// dispatchAvg21 is the vertical pass of Ratio21 — same pavgb dispatcher as
// ratio32 uses on amd64.
func dispatchAvg21(dst, a, b []byte) {
	dispatchAvg(dst, a, b)
}

// h21AVX2 processes `iters` iterations, each: two 32-byte loads (+0, +16),
// 4 output pixels (16 bytes) — see ratio21_amd64.s for the layout.
// Caller guarantees rows >= 32*(iters-1)+48 source bytes and >= 16*iters
// destination bytes.
func h21AVX2(dst, src []byte, iters int)

// hn21AVX2 is the nearest twin of h21AVX2: pure pshufb copies, same bounds.
func hn21AVX2(dst, src []byte, iters int)
