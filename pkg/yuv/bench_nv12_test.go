package yuv

import "testing"

// Bytes touched per frame: src read (w*h*4) + dst write (w*h*1.5).
func setBytesNV12(b *testing.B, w, h int) {
	b.SetBytes(int64(w * h * 11 / 2))
}

func benchScalar(b *testing.B, w, h int) {
	setBytesNV12(b, w, h)
	src := testBuf2(h * w * 4)
	dst := make([]byte, w*(h+h/2))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScalarBGRAtoNV12(dst, src, w, h, w*4)
	}
}

func benchDispatch(b *testing.B, w, h int) {
	setBytesNV12(b, w, h)
	p := NewNV12(w, h, w*4)
	if p == nil {
		b.Fatal("plan rejected")
	}
	src := testBuf2(h * w * 4)
	dst := make([]byte, p.Size())
	p.Into(dst, src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Into(dst, src)
	}
}

// Scalar reference points (the tablet path today).
func BenchmarkNV12_Scalar_1080p(b *testing.B)      { benchScalar(b, 1920, 1080) }
func BenchmarkNV12_Scalar_720p(b *testing.B)       { benchScalar(b, 1280, 720) }
func BenchmarkNV12_Scalar_Tablet800p(b *testing.B) { benchScalar(b, 1280, 800) }
func BenchmarkNV12_Scalar_Tablet400p(b *testing.B) { benchScalar(b, 640, 400) }

// Dispatch path (equals scalar until the XMM kernels land).
func BenchmarkNV12_Dispatch_1080p(b *testing.B)      { benchDispatch(b, 1920, 1080) }
func BenchmarkNV12_Dispatch_720p(b *testing.B)       { benchDispatch(b, 1280, 720) }
func BenchmarkNV12_Dispatch_Tablet800p(b *testing.B) { benchDispatch(b, 1280, 800) }
func BenchmarkNV12_Dispatch_Tablet400p(b *testing.B) { benchDispatch(b, 640, 400) }
