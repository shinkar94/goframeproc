//go:build !amd64 && !386

package simd

// Portable fallback for non-amd64/non-386 targets (386 has its own SSSE3-gated XMM path — swap_386.go).
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
