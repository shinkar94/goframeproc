//go:build amd64

package yuv

import "github.com/shinkar94/goframeproc/pkg/simd"

func dispatchY(dstY, srcRow []byte, w int) {
	// SSE4.1, not just SSSE3: the kernels use PMOVZXBW.
	if !simd.HaveSSE41() {
		yScalar(dstY, srcRow, w)
		return
	}
	w4 := w &^ 3
	if w4 > 0 {
		yRowSSE(dstY[:w4], srcRow[:w4*4])
	}
	if w4 < w {
		yScalar(dstY[w4:], srcRow[w4*4:], w-w4)
	}
}

func dispatchUV(dstUV, srcRow0, srcRow1 []byte, w2 int) {
	if !simd.HaveSSE41() {
		uvScalar(dstUV, srcRow0, srcRow1, w2)
		return
	}
	w2a := w2 &^ 1
	if w2a > 0 {
		uvRowSSE(dstUV[:2*w2a], srcRow0[:8*w2a], srcRow1[:8*w2a])
	}
	if w2a < w2 {
		uvScalar(dstUV[2*w2a:], srcRow0[8*w2a:], srcRow1[8*w2a:], w2-w2a)
	}
}

// yRowSSE converts floor(len(dstY)/4) 4-px blocks via SSSE3; the tail is
// the caller's.
func yRowSSE(dstY []byte, srcRow []byte)

// uvRowSSE converts floor(len(srcRow0)/16) lanes (2 chroma blocks each);
// the tail is the caller's.
func uvRowSSE(dstUV []byte, srcRow0 []byte, srcRow1 []byte)
