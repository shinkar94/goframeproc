//go:build 386 || amd64

package yuv

import (
	"bytes"
	"testing"

	"github.com/shinkar94/goframeproc/pkg/simd"
)

// Direct asm-path pinning: the SSE4.1 kernels must be byte-identical to
// the scalar rows at every block-boundary width. Skipped on CPUs without
// SSE4.1 — the dispatch falls back to scalar there.
func TestYRowSSEMatchesScalar(t *testing.T) {
	if !simd.HaveSSE41() {
		t.Skip("no SSE4.1 on this machine")
	}
	widths := []int{4, 8, 12, 16, 32, 64, 100, 252, 320}
	for _, w := range widths {
		srcRow := testBuf2(w * 4)
		want := make([]byte, w)
		yScalar(want, srcRow, w)
		got := make([]byte, w)
		yRowSSE(got, srcRow)
		if !bytes.Equal(got, want) {
			t.Fatalf("yRowSSE w=%d: differs from scalar", w)
		}
	}
}

func TestUVRowSSEMatchesScalar(t *testing.T) {
	if !simd.HaveSSE41() {
		t.Skip("no SSE4.1 on this machine")
	}
	// The kernel processes lanes of 2 blocks; odd tails belong to the
	// dispatch's scalar job and are covered by TestDispatchMatchesScalar.
	blocks := []int{2, 4, 6, 8, 16, 32, 50}
	for _, w2 := range blocks {
		srcRow0 := testBuf2(w2 * 8)
		srcRow1 := testBuf2(w2 * 8)
		want := make([]byte, 2*w2)
		uvScalar(want, srcRow0, srcRow1, w2)
		got := make([]byte, 2*w2)
		uvRowSSE(got, srcRow0, srcRow1)
		if !bytes.Equal(got, want) {
			t.Fatalf("uvRowSSE w2=%d: differs from scalar", w2)
		}
	}
}
