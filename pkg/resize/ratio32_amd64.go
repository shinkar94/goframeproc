//go:build amd64

package resize

import "github.com/shinkar94/goframeproc/pkg/simd"

// pshufb masks: index >= 0x80 zeroes the byte; masks are per 128-bit lane.
// Both select from a window whose lane0 holds px0..3 (or px3..6 for the
// +12-byte window): [px1, px2] and [px2, px3] respectively — the pavgb pair
// plus its shifted twin.
var hMaskP = [32]byte{
	4, 5, 6, 7, 8, 9, 10, 11,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
}

var hMaskT = [32]byte{
	8, 9, 10, 11, 12, 13, 14, 15,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
}

// hMaskPair takes the first two pixels (8 bytes) of a window.
var hMaskPair = [32]byte{
	0, 1, 2, 3, 4, 5, 6, 7,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
}

// dispatchH32 runs the horizontal 3:2 kernel for `groups` output units
// (group = 2 output pixels, 3 source pixels). The AVX2 kernel advances
// 48 source bytes per iteration while reading up to 68, so the final
// iterations would spill past the row — the tail stays scalar.
func dispatchH32(dstRow, srcRow []byte, groups int) {
	if simd.HaveAVX2() {
		if iters := groups / 4; iters > 1 {
			h32AVX2(dstRow, srcRow, iters-1)
			done := 4 * (iters - 1)
			h32Scalar(dstRow[done*8:], srcRow[done*12:], groups-done)
			return
		}
	}
	h32Scalar(dstRow, srcRow, groups)
}

// dispatchH32Nearest is the nearest twin of dispatchH32 (same bounds rules).
func dispatchH32Nearest(dstRow, srcRow []byte, groups int) {
	if simd.HaveAVX2() {
		if iters := groups / 4; iters > 1 {
			hn32AVX2(dstRow, srcRow, iters-1)
			done := 4 * (iters - 1)
			h32NearestScalar(dstRow[done*8:], srcRow[done*12:], groups-done)
			return
		}
	}
	h32NearestScalar(dstRow, srcRow, groups)
}

// dispatchAvg averages rows with pavgb semantics: 128-byte AVX2 quads,
// scalar remainder.
func dispatchAvg(dst, a, b []byte) {
	n := len(dst)
	if simd.HaveAVX2() {
		full := n &^ 127
		if full > 0 {
			avgAVX2(dst[:full], a[:full], b[:full])
		}
		avgScalar(dst[full:], a[full:], b[full:])
		return
	}
	avgScalar(dst, a, b)
}

// h32AVX2 processes `iters` iterations, each: two 32-byte loads (at +0,
// +12), 8 output pixels (32 bytes) — see ratio32_amd64.s for the layout.
// Caller guarantees rows >= 48*(iters-1)+68 source bytes and
// >= 32*iters destination bytes.
func h32AVX2(dst, src []byte, iters int)

// hn32AVX2 is the nearest twin of h32AVX2: pure pshufb copies, same bounds.
func hn32AVX2(dst, src []byte, iters int)

// avgAVX2 processes floor(len(dst)/128) 128-byte quads: dst = pavgb(a, b).
// The tail is handled by the caller.
func avgAVX2(dst, a, b []byte)
