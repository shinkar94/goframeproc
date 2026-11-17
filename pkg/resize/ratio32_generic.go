//go:build !amd64

package resize

func dispatchH32(dstRow, srcRow []byte, groups int) {
	h32Scalar(dstRow, srcRow, groups)
}

func dispatchH32Nearest(dstRow, srcRow []byte, groups int) {
	h32NearestScalar(dstRow, srcRow, groups)
}

func dispatchAvg(dst, a, b []byte) {
	avgScalar(dst, a, b)
}
