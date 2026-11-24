package resize

import (
	"bytes"
	"math/rand"
	"testing"
)

func ratio21TestBuf(w, h, stride int) []byte {
	b := make([]byte, (h-1)*stride+w*4)
	r := rand.New(rand.NewSource(7))
	r.Read(b)
	return b
}

func TestNewRatio21Guards(t *testing.T) {
	cases := []struct {
		name                       string
		srcW, srcH, stride, dW, dH int
	}{
		{"not 2x1 width", 1281, 800, 1280 * 4, 640, 400},
		{"not 2x1 height", 1280, 801, 1280 * 4, 640, 400},
		{"stride too small", 1280, 800, 100, 640, 400},
		{"zero dst", 2, 2, 8, 0, 1},
	}
	for _, c := range cases {
		if got := NewRatio21(c.srcW, c.srcH, c.stride, c.dW, c.dH); got != nil {
			t.Fatalf("%s: expected nil plan", c.name)
		}
	}
	if NewRatio21(1280, 800, 1280*4, 640, 400) == nil {
		t.Fatal("valid 1280x800->640x400 plan rejected")
	}
}

// Into must be byte-identical to its all-scalar path — pins the dispatch
// and the asm on every architecture.
func TestRatio21IntoMatchesScalarPath(t *testing.T) {
	plans := []struct{ srcW, srcH, stride, dW, dH int }{
		{1280, 800, 1280 * 4, 640, 400}, // the tablet case
		{32, 16, 32 * 4, 16, 8},
		{20, 10, 20 * 4, 10, 5}, // dstPx 10: one vector iter + 2-px tail
		{16, 8, 24 * 4, 8, 4},   // stride > row, dstPx == 8 (single iter)
		{12, 6, 12 * 4, 6, 3},   // dstPx < 4: scalar-only
	}
	for _, p := range plans {
		r := NewRatio21(p.srcW, p.srcH, p.stride, p.dW, p.dH)
		if r == nil {
			t.Fatalf("plan %dx%d->%dx%d rejected", p.srcW, p.srcH, p.dW, p.dH)
		}
		src := ratio21TestBuf(p.srcW, p.srcH, p.stride)
		dst := make([]byte, p.dW*p.dH*4)
		r.Into(dst, src)

		want := make([]byte, len(dst))
		ratio21TwoPassScalar(want, src, r)
		if !bytes.Equal(dst, want) {
			t.Fatalf("Into %dx%d->%dx%d: differs from scalar path", p.srcW, p.srcH, p.dW, p.dH)
		}
	}
}

// IntoNearest must be byte-identical to its all-scalar path.
func TestRatio21NearestMatchesScalarPath(t *testing.T) {
	plans := []struct{ srcW, srcH, stride, dW, dH int }{
		{1280, 800, 1280 * 4, 640, 400},
		{32, 16, 32 * 4, 16, 8},
		{20, 10, 20 * 4, 10, 5},
		{16, 8, 24 * 4, 8, 4},
	}
	for _, p := range plans {
		r := NewRatio21(p.srcW, p.srcH, p.stride, p.dW, p.dH)
		src := ratio21TestBuf(p.srcW, p.srcH, p.stride)
		dst := make([]byte, p.dW*p.dH*4)
		r.IntoNearest(dst, src)

		want := make([]byte, len(dst))
		ratio21NearestScalarPath(want, src, r)
		if !bytes.Equal(dst, want) {
			t.Fatalf("IntoNearest %dx%d->%dx%d: differs from scalar path", p.srcW, p.srcH, p.dW, p.dH)
		}
	}
}

// Nearest semantics pinned explicitly on a tiny known frame: out px k is
// source pixel 2k, out row k is source row 2k.
func TestRatio21NearestKnownFrame(t *testing.T) {
	srcW, srcH := 4, 4
	stride := srcW * 4
	src := make([]byte, srcW*srcH*4)
	for px := 0; px < srcW*srcH; px++ {
		src[px*4] = byte(px) // R channel carries the pixel index
	}
	r := NewRatio21(srcW, srcH, stride, 2, 2)
	dst := make([]byte, 2*2*4)
	r.IntoNearest(dst, src)
	wantPx := []byte{0, 2, 8, 10} // rows 0,2; px 0,2
	for k, w := range wantPx {
		if got := dst[k*4]; got != w {
			t.Fatalf("nearest px %d: got R=%d want %d", k, got, w)
		}
		if dst[k*4+1] != 0 || dst[k*4+2] != 0 || dst[k*4+3] != 0 {
			t.Fatalf("nearest px %d: non-source channels polluted", k)
		}
	}
}

// The two-pass pavgb result must stay within 2 LSB of the independent
// single-pass 4-tap reference (same tolerance family as ratio32).
func TestRatio21VsRef4Tap(t *testing.T) {
	srcW, srcH, stride, dW, dH := 1280, 800, 1280*4, 640, 400
	r := NewRatio21(srcW, srcH, stride, dW, dH)
	src := ratio21TestBuf(srcW, srcH, stride)
	dst := make([]byte, dW*dH*4)
	r.Into(dst, src)

	ref := make([]byte, dW*dH*4)
	Ratio21Ref4Tap(ref, src, srcW, srcH, stride, dW, dH)

	maxDiff := 0
	for i := range dst {
		d := int(dst[i]) - int(ref[i])
		if d < 0 {
			d = -d
		}
		if d > maxDiff {
			maxDiff = d
		}
	}
	if maxDiff > 2 {
		t.Fatalf("bilinear vs 4-tap reference: maxDiff=%d exceeds 2 LSB", maxDiff)
	}
	t.Logf("maxDiff vs 4-tap reference: %d LSB", maxDiff)
}

// Zero-allocation contract of Into on a warm plan.
func TestRatio21IntoAllocs(t *testing.T) {
	r := NewRatio21(1280, 800, 1280*4, 640, 400)
	src := ratio21TestBuf(1280, 800, 1280*4)
	dst := make([]byte, 640*400*4)
	r.Into(dst, src) // warm up any lazy state
	allocs := testing.AllocsPerRun(10, func() {
		r.Into(dst, src)
	})
	if allocs != 0 {
		t.Fatalf("Into allocates %v objects per frame on a warm plan", allocs)
	}
}
