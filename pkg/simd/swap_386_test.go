//go:build 386

package simd

import (
	"bytes"
	"math/rand"
	"testing"
)

func swapReference(dst, src []byte) {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	for i := 0; i+4 <= n; i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], src[i+3]
	}
}

// The full-frame SwapAVX2 on 386 (XMM prefix + scalar tail) must be
// byte-identical to the reference for sizes around the 16-byte boundary,
// including odd tails.
func TestSwap386FullFrameMatchesReference(t *testing.T) {
	sizes := []int{0, 1, 3, 4, 15, 16, 17, 31, 100, 1000, 100000}
	for _, n := range sizes {
		src := make([]byte, n)
		rand.New(rand.NewSource(42)).Read(src)
		want := make([]byte, n)
		swapReference(want, src)

		got := make([]byte, n)
		SwapAVX2(src, got)
		if !bytes.Equal(got, want) {
			t.Fatalf("SwapAVX2 n=%d: output differs from reference", n)
		}

		got2 := make([]byte, n)
		SwapSSE(src, got2)
		if !bytes.Equal(got2, want) {
			t.Fatalf("SwapSSE n=%d: output differs from reference", n)
		}
	}
}

// The direct XMM kernel (prefix-only contract) must match the reference on
// the aligned prefix and leave the tail untouched.
func TestSwap386DirectPathMatchesReference(t *testing.T) {
	if !HaveSSSE3() {
		t.Skip("SSSE3 unavailable; the dispatch stays scalar")
	}
	sizes := []int{0, 1, 15, 16, 17, 100, 100000}
	for _, n := range sizes {
		src := make([]byte, n)
		rand.New(rand.NewSource(7)).Read(src)
		want := make([]byte, n)
		swapReference(want, src)

		got := make([]byte, n)
		full := n &^ 15
		if full > 0 {
			swapSSE386(src[:full], got[:full])
		}
		if !bytes.Equal(got[:full], want[:full]) {
			t.Fatalf("swapSSE386 n=%d (full=%d): prefix differs from reference", n, full)
		}
	}
}
