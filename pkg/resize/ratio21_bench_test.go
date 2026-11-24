package resize

import "testing"

// The 16:10 tablet case: 1280x800 -> 640x400.
// Dispatched vs all-scalar path makes the SIMD win measurable on every
// architecture.

const ratio21SrcW, ratio21SrcH, ratio21Stride = 1280, 800, 1280 * 4
const ratio21DstW, ratio21DstH = 640, 400

func ratio21BenchSrc() []byte { return ratio21TestBuf(ratio21SrcW, ratio21SrcH, ratio21Stride) }

func BenchmarkRatio21Into_800p(b *testing.B) {
	r := NewRatio21(ratio21SrcW, ratio21SrcH, ratio21Stride, ratio21DstW, ratio21DstH)
	src := ratio21BenchSrc()
	dst := make([]byte, ratio21DstW*ratio21DstH*4)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Into(dst, src)
	}
}

func BenchmarkRatio21Scalar_800p(b *testing.B) {
	r := NewRatio21(ratio21SrcW, ratio21SrcH, ratio21Stride, ratio21DstW, ratio21DstH)
	src := ratio21BenchSrc()
	dst := make([]byte, ratio21DstW*ratio21DstH*4)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ratio21TwoPassScalar(dst, src, r)
	}
}

func BenchmarkRatio21Nearest_800p(b *testing.B) {
	r := NewRatio21(ratio21SrcW, ratio21SrcH, ratio21Stride, ratio21DstW, ratio21DstH)
	src := ratio21BenchSrc()
	dst := make([]byte, ratio21DstW*ratio21DstH*4)
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.IntoNearest(dst, src)
	}
}
