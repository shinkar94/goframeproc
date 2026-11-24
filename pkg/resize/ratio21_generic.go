//go:build !amd64 && !386

package resize

// Non-amd64/non-386 platforms run the fixed-ratio 2:1 kernels scalar:
// the specialization itself (contiguous 8-in/4-out consumption, no weight
// gather) is the win. The 386 XMM dispatch lives in ratio21_386.go.

func dispatchH21(dstRow, srcRow []byte, dstPx int) {
	h21Scalar(dstRow, srcRow, dstPx)
}

func dispatchH21Nearest(dstRow, srcRow []byte, dstPx int) {
	h21NearestScalar(dstRow, srcRow, dstPx)
}

func dispatchAvg21(dst, a, b []byte) {
	avgScalar(dst, a, b)
}
