package frameproc

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/shinkar94/goframeproc/pkg/simd"
)

// SIMD swap must be byte-identical to the scalar word swap for every size
// around the vector boundaries.
func TestConvertSIMDMatchesScalar(t *testing.T) {
	if !simd.HaveAVX2() && !simd.HaveSSSE3() {
		t.Skip("no SIMD features on this machine")
	}
	sizes := []int{0, 1, 2, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 100, 1000, 1920 * 1080}
	for _, pixels := range sizes {
		src := make([]byte, pixels*4)
		for i := range src {
			src[i] = byte(i%251) ^ byte(i/253)
		}
		want := make([]byte, pixels*4)
		for i := 0; i+4 <= pixels*4; i += 4 {
			want[i], want[i+1], want[i+2], want[i+3] = src[i+2], src[i+1], src[i], src[i+3]
		}
		got := make([]byte, pixels*4)
		convertFormatInto(got, src, pixels, false)
		if !bytes.Equal(got, want) {
			t.Fatalf("pixels=%d: SIMD convert output differs from scalar reference", pixels)
		}
	}
}

func BenchmarkConvert_ScalarReference_1080p(b *testing.B) {
	src := testBuffer()
	dst := make([]byte, len(src))
	n := len(src) / 4
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			v := binary.LittleEndian.Uint32(src[j*4 : j*4+4])
			binary.LittleEndian.PutUint32(dst[j*4:j*4+4], (v&0xFF00FF00)|((v&0x00FF0000)>>16)|((v&0x000000FF)<<16))
		}
	}
}

func BenchmarkConvert_SIMD_1080p(b *testing.B) {
	if !simd.HaveAVX2() {
		b.Skip("no AVX2")
	}
	src := testBuffer()
	dst := make([]byte, len(src))
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		simd.SwapAVX2(src, dst)
	}
}
