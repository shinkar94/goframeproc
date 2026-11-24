//go:build 386

package simd

import "golang.org/x/sys/cpu"

// Same lane layout as the amd64 maskSSE: per BGRA quad swap byte 0 with 2.
var maskSSE386 = [16]byte{2, 1, 0, 3, 6, 5, 4, 7, 10, 9, 8, 11, 14, 13, 12, 15}

// scalarSwap386 is the portable reference used for tails and as the
// no-SSSE3 fallback (the same loop as the previous swap_other.go fallback).
func scalarSwap386(dst, src []byte) {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	for i := 0; i+4 <= n; i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], src[i+3]
	}
}

// SwapAVX2 converts the full src frame BGRA->RGBA into dst. On 386 the
// historical swap_other.go fallback was a full scalar swap, so this entry
// keeps the full-frame contract: the SSSE3 XMM kernel handles the
// 16-byte-aligned prefix (п. 12 of the lab journal — measured ×3.5 on a
// Silvermont Atom Z3735F), the scalar loop covers the remainder. Without
// SSSE3 it degrades to the plain scalar swap.
func SwapAVX2(src []byte, dst []byte) {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	if cpu.X86.HasSSSE3 {
		full := n &^ 15
		if full > 0 {
			swapSSE386(src[:full], dst[:full])
		}
		scalarSwap386(dst[full:n], src[full:n])
		return
	}
	scalarSwap386(dst[:n], src[:n])
}

// SwapSSE is an alias of SwapAVX2 on 386 (both route to the same SSSE3-gated
// XMM kernel; the name is kept for API parity with amd64).
func SwapSSE(src []byte, dst []byte) { SwapAVX2(src, dst) }

// swapSSE386 processes floor(len(src)/16)*16 bytes; the tail is handled by
// the caller. SSSE3 pshufb in the XMM variant.
func swapSSE386(src []byte, dst []byte)
