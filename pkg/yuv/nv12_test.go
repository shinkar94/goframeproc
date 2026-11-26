package yuv

import (
	"bytes"
	"math/rand"
	"runtime"
	"testing"
)

func testBuf2(n int) []byte {
	b := make([]byte, n)
	r := rand.New(rand.NewSource(42))
	r.Read(b)
	return b
}

// The dispatch must be byte-identical to the exported scalar reference on
// every size family: vector-boundary widths (multiples of 4/8 and odd
// tails down to w=2), padded source strides, short frames.
func TestDispatchMatchesScalar(t *testing.T) {
	widths := []int{2, 4, 6, 8, 10, 12, 16, 18, 32, 34, 64, 100, 254, 320}
	heights := []int{2, 4, 6, 8, 10, 16, 18, 32}
	pads := []int{0, 4, 16}
	for _, w := range widths {
		for _, h := range heights {
			for _, pad := range pads {
				stride := w*4 + pad
				src := testBuf2((h-1)*stride + w*4)
				want := make([]byte, w*(h+h/2))
				ScalarBGRAtoNV12(want, src, w, h, stride)

				p := NewNV12(w, h, stride)
				if p == nil {
					t.Fatalf("plan rejected w=%d h=%d stride=%d", w, h, stride)
				}
				got := make([]byte, p.Size())
				p.Into(got, src)
				if !bytes.Equal(got, want) {
					t.Fatalf("dispatch w=%d h=%d stride=%d: output differs", w, h, stride)
				}
			}
		}
	}
}

func TestPlanRejectsInvalid(t *testing.T) {
	cases := []struct{ w, h, stride int }{
		{0, 2, 8}, {2, 0, 8}, {-2, 2, 8}, {2, -2, 8},
		{1, 2, 8}, {2, 1, 8}, // odd w/h
		{2, 2, 7}, // stride < 4*w
	}
	for _, c := range cases {
		if p := NewNV12(c.w, c.h, c.stride); p != nil {
			t.Fatalf("NewNV12(%d,%d,%d) accepted", c.w, c.h, c.stride)
		}
	}
}

func TestPanicsShortBuffers(t *testing.T) {
	p := NewNV12(8, 4, 32)
	src := testBuf2(3*32 + 32)
	dst := make([]byte, p.Size())

	shortDst := make([]byte, p.Size()-1)
	if !panics(func() { p.Into(shortDst, src) }) {
		t.Fatal("short dst accepted")
	}
	shortSrc := src[:3*32+16] // last row truncated
	if !panics(func() { p.Into(dst, shortSrc) }) {
		t.Fatal("short src accepted")
	}
}

func panics(f func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	f()
	return
}

// The plan must not allocate on the hot path (the streaming pattern reuses
// dst and plan).
func TestZeroAlloc(t *testing.T) {
	p := NewNV12(320, 240, 320*4)
	src := testBuf2(240 * 320 * 4)
	dst := make([]byte, p.Size())
	p.Into(dst, src)
	if n := testing.AllocsPerRun(100, func() { p.Into(dst, src) }); n != 0 {
		t.Fatalf("allocs per frame: %v", n)
	}
}

func TestDetectedFeatures(t *testing.T) {
	t.Logf("GOARCH=%s", runtime.GOARCH)
}
