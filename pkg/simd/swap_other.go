//go:build !amd64

package simd

// Portable fallback for non-amd64 targets.
func SwapAVX2(src []byte, dst []byte) { scalarSwap(dst, src) }
func SwapSSE(src []byte, dst []byte)  { scalarSwap(dst, src) }

func scalarSwap(dst, src []byte) {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	for i := 0; i+4 <= n; i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], src[i+3]
	}
}
