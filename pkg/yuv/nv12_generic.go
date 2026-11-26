//go:build !amd64 && !386

package yuv

// Portable fallback for non-x86 builds.
func dispatchY(dstY, srcRow []byte, w int) {
	yScalar(dstY, srcRow, w)
}

func dispatchUV(dstUV, srcRow0, srcRow1 []byte, w2 int) {
	uvScalar(dstUV, srcRow0, srcRow1, w2)
}
