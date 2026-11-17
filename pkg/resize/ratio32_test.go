package resize

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/shinkar94/goframeproc/pkg/simd"
)

var ratio32Cases = []struct{ sw, sh, tw, th int }{
	{1920, 1080, 1280, 720},
	{960, 540, 640, 360},
	{192, 108, 128, 72},
	{12, 9, 8, 6}, // exercises 1 asm unit + scalar remainder
	{6, 3, 4, 2},  // all-scalar path
	{3840, 2160, 2560, 1440},
}

func ratio32TestBuffer(n int) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(int64(n) + 100)).Read(b)
	return b
}

func maxChannelDiff(a, b []byte) int {
	maxDiff := 0
	for i := range a {
		d := int(a[i]) - int(b[i])
		if d < 0 {
			d = -d
		}
		if d > maxDiff {
			maxDiff = d
		}
	}
	return maxDiff
}

// The dispatched (SIMD) path must stay within 2 LSB of the independent
// single-pass 32-bit exact-ratio reference: one LSB per 16-bit pass (pavgb
// rounds up where the 32-bit floor rounds down); pure-copy pixels are
// identical. The all-scalar path is checked first so mapping bugs are
// distinguishable from asm bugs.
func TestRatio32MatchesRef4Tap(t *testing.T) {
	for _, c := range ratio32Cases {
		src := ratio32TestBuffer(c.sw * c.sh * 4)
		stride := c.sw * 4

		want := make([]byte, c.tw*c.th*4)
		Ratio32Ref4Tap(want, src, c.sw, c.sh, stride, c.tw, c.th)

		p := NewRatio32(c.sw, c.sh, stride, c.tw, c.th)
		if p == nil {
			t.Fatalf("%dx%d -> %dx%d: plan rejected", c.sw, c.sh, c.tw, c.th)
		}

		scalar := make([]byte, c.tw*c.th*4)
		ratio32TwoPassScalar(scalar, src, p)
		if d := maxChannelDiff(scalar, want); d > 2 {
			t.Fatalf("%dx%d -> %dx%d: scalar two-pass vs 32-bit ref: %d LSB exceeds 2",
				c.sw, c.sh, c.tw, c.th, d)
		}

		got := make([]byte, c.tw*c.th*4)
		p.Into(got, src)
		if d := maxChannelDiff(got, want); d > 2 {
			t.Fatalf("%dx%d -> %dx%d: dispatched path vs 32-bit ref: %d LSB exceeds 2",
				c.sw, c.sh, c.tw, c.th, d)
		}
		t.Logf("%dx%d -> %dx%d: maxDiff=%d", c.sw, c.sh, c.tw, c.th, maxChannelDiff(got, want))
	}
}

// The dispatched (SIMD) path must be byte-identical to the all-scalar path —
// same fixed-point math, asm is just data movement around the same pavgb.
func TestRatio32AsmByteIdentical(t *testing.T) {
	for _, c := range ratio32Cases {
		src := ratio32TestBuffer(c.sw * c.sh * 4)
		stride := c.sw * 4
		p := NewRatio32(c.sw, c.sh, stride, c.tw, c.th)
		if p == nil {
			t.Fatalf("%dx%d -> %dx%d: plan rejected", c.sw, c.sh, c.tw, c.th)
		}

		viaSIMD := make([]byte, c.tw*c.th*4)
		p.Into(viaSIMD, src)

		viaScalar := make([]byte, c.tw*c.th*4)
		ratio32TwoPassScalar(viaScalar, src, p)

		if !bytes.Equal(viaSIMD, viaScalar) {
			if simd.HaveAVX2() {
				t.Fatalf("%dx%d -> %dx%d: AVX2 path differs from scalar", c.sw, c.sh, c.tw, c.th)
			}
			t.Fatalf("%dx%d -> %dx%d: paths differ", c.sw, c.sh, c.tw, c.th)
		}
	}
}

func TestNewRatio32Validation(t *testing.T) {
	ok := []struct{ sw, sh, stride, tw, th int }{
		{1920, 1080, 1920 * 4, 1280, 720},
		{1920, 1080, 1920*4 + 64, 1280, 720}, // padded stride is fine
		{6, 3, 24, 4, 2},
	}
	for _, c := range ok {
		if NewRatio32(c.sw, c.sh, c.stride, c.tw, c.th) == nil {
			t.Fatalf("%dx%d stride=%d -> %dx%d: expected valid", c.sw, c.sh, c.stride, c.tw, c.th)
		}
	}
	bad := []struct{ sw, sh, stride, tw, th int }{
		{1920, 1080, 1920 * 4, 1279, 720},   // dstW odd
		{1920, 1080, 1920 * 4, 1280, 719},   // dstH odd
		{1919, 1080, 1919 * 4, 1280, 720},   // srcW not exactly 3*dstW/2
		{1920, 1079, 1920 * 4, 1280, 720},   // srcH not exactly 3*dstH/2
		{1920, 1080, 1920*4 - 1, 1280, 720}, // stride too small
		{1920, 1080, 1920 * 4, 1920, 1080},  // not a downscale
		{0, 0, 0, 0, 0},                     // degenerate
	}
	for _, c := range bad {
		if NewRatio32(c.sw, c.sh, c.stride, c.tw, c.th) != nil {
			t.Fatalf("%dx%d stride=%d -> %dx%d: expected nil", c.sw, c.sh, c.stride, c.tw, c.th)
		}
	}
}

// Philosophy: Into must not allocate per frame (buffers live in the plan).
func TestRatio32IntoZeroAlloc(t *testing.T) {
	src := ratio32TestBuffer(1920 * 1080 * 4)
	dst := make([]byte, 1280*720*4)
	p := NewRatio32(1920, 1080, 1920*4, 1280, 720)
	if p == nil {
		t.Fatal("plan rejected")
	}
	p.Into(dst, src) // warm up
	a := testing.AllocsPerRun(10, func() { p.Into(dst, src) })
	if a != 0 {
		t.Fatalf("Into allocates %.1f objects/frame", a)
	}
}

// Panic guards for undersized buffers.
func TestRatio32BufferGuards(t *testing.T) {
	p := NewRatio32(12, 9, 48, 8, 6)
	if p == nil {
		t.Fatal("plan rejected")
	}
	src := make([]byte, 12*9*4)
	dst := make([]byte, 8*6*4)
	p.Into(dst, src) // exact sizes must pass

	r := rand.New(rand.NewSource(3))
	smallSrc := make([]byte, 12*9*4-1)
	r.Read(smallSrc)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic on short src")
			}
		}()
		p.Into(dst, smallSrc)
	}()
	smallDst := make([]byte, 8*6*4-1)
	r.Read(smallDst)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic on short dst")
			}
		}()
		p.Into(smallDst, src)
	}()
}

// nearestRef is the independent exact-ratio floor reference: out(x,y)
// = src(floor(3x/2), floor(3y/2)) — the same truncation rule the generic
// nearest path uses (int(xf)).
func nearestRef(dst, src []byte, srcW, srcH, stride, dstW, dstH int) {
	for y := 0; y < dstH; y++ {
		iy := (3 * y) / 2
		row := src[iy*stride:]
		for x := 0; x < dstW; x++ {
			ix := (3 * x) / 2
			copy(dst[(y*dstW+x)*4:(y*dstW+x)*4+4], row[ix*4:ix*4+4])
		}
	}
}

func TestRatio32NearestMatchesReference(t *testing.T) {
	for _, c := range ratio32Cases {
		src := ratio32TestBuffer(c.sw * c.sh * 4)
		stride := c.sw * 4
		p := NewRatio32(c.sw, c.sh, stride, c.tw, c.th)
		if p == nil {
			t.Fatalf("%dx%d -> %dx%d: plan rejected", c.sw, c.sh, c.tw, c.th)
		}

		want := make([]byte, c.tw*c.th*4)
		nearestRef(want, src, c.sw, c.sh, stride, c.tw, c.th)

		got := make([]byte, c.tw*c.th*4)
		p.IntoNearest(got, src)
		if !bytes.Equal(got, want) {
			if simd.HaveAVX2() {
				t.Fatalf("%dx%d -> %dx%d: AVX2 nearest path differs from reference", c.sw, c.sh, c.tw, c.th)
			}
			t.Fatalf("%dx%d -> %dx%d: nearest paths differ", c.sw, c.sh, c.tw, c.th)
		}
	}
}

// IntoNearest must not allocate per frame.
func TestRatio32NearestZeroAlloc(t *testing.T) {
	src := ratio32TestBuffer(1920 * 1080 * 4)
	dst := make([]byte, 1280*720*4)
	p := NewRatio32(1920, 1080, 1920*4, 1280, 720)
	p.IntoNearest(dst, src)
	a := testing.AllocsPerRun(10, func() { p.IntoNearest(dst, src) })
	if a != 0 {
		t.Fatalf("IntoNearest allocates %.1f objects/frame", a)
	}
}

// Benchmarks for release evidence (i5-12450H reference numbers in CHANGELOG).
func BenchmarkRatio32Into_1080p_720p(b *testing.B) {
	src := ratio32TestBuffer(1920 * 1080 * 4)
	dst := make([]byte, 1280*720*4)
	p := NewRatio32(1920, 1080, 1920*4, 1280, 720)
	if p == nil {
		b.Fatal("plan rejected")
	}
	p.Into(dst, src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Into(dst, src)
	}
}

func BenchmarkRatio32Nearest_1080p_720p(b *testing.B) {
	src := ratio32TestBuffer(1920 * 1080 * 4)
	dst := make([]byte, 1280*720*4)
	p := NewRatio32(1920, 1080, 1920*4, 1280, 720)
	if p == nil {
		b.Fatal("plan rejected")
	}
	p.IntoNearest(dst, src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.IntoNearest(dst, src)
	}
}
