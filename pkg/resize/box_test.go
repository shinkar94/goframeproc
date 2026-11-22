package resize

import (
	"bytes"
	"testing"
)

// A 3x1 downscale of vertical 1px stripes (white every 3rd pixel) must
// produce a uniform field: each output pixel averages exactly one white and
// two black pixels -> 85 (round-half-up of 255/3 = 85).
func TestBoxStripesUniform(t *testing.T) {
	const sw, sh, tw, th = 9, 3, 3, 1
	src := make([]byte, sw*sh*4)
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			v := byte(0)
			if x%3 == 0 {
				v = 255
			}
			o := (y*sw + x) * 4
			src[o], src[o+1], src[o+2], src[o+3] = v, v, v, 255
		}
	}
	dst := make([]byte, tw*th*4)
	ResizeGenericInto(dst, src, sw, sh, sw*4, tw, th, Box, "BGRA", SequentialResizeStrategy{}, "BGRA")
	for x := 0; x < tw; x++ {
		for _, ch := range dst[x*4 : x*4+3] {
			if ch != 85 {
				t.Fatalf("stripe average at x=%d = %d, want 85", x, ch)
			}
		}
		if dst[x*4+3] != 255 {
			t.Fatalf("alpha at x=%d = %d, want 255", x, dst[x*4+3])
		}
	}
}

// Format conversion must happen after averaging: a uniform BGRA source
// (B=255) downscaled to RGBA must place 255 in the R slot.
func TestBoxFormatConversion(t *testing.T) {
	const sw, sh, tw, th = 2, 2, 1, 1
	src := make([]byte, sw*sh*4)
	for i := 0; i < sw*sh; i++ {
		src[i*4], src[i*4+1], src[i*4+2], src[i*4+3] = 255, 0, 0, 255 // B=255
	}
	dst := make([]byte, tw*th*4)
	ResizeGenericInto(dst, src, sw, sh, sw*4, tw, th, Box, "RGBA", SequentialResizeStrategy{}, "BGRA")
	if dst[0] != 0 || dst[2] != 255 {
		t.Fatalf("BGRA->RGBA mapping wrong: got R=%d B=%d, want R=0 B=255", dst[0], dst[2])
	}
}

// Round-half-up: average 127.5 must round to 128.
func TestBoxRounding(t *testing.T) {
	const sw, sh, tw, th = 2, 2, 1, 1
	vals := []byte{255, 255, 0, 0}
	src := make([]byte, sw*sh*4)
	for i := 0; i < sw*sh; i++ {
		src[i*4], src[i*4+1], src[i*4+2], src[i*4+3] = vals[i], vals[i], 0, 255
	}
	dst := make([]byte, tw*th*4)
	ResizeGenericInto(dst, src, sw, sh, sw*4, tw, th, Box, "BGRA", SequentialResizeStrategy{}, "BGRA")
	if dst[0] != 128 {
		t.Fatalf("round-half-up: got %d, want 128", dst[0])
	}
}

// Sequential and parallel strategies must produce byte-identical output
// (every output row is independent in the box kernel).
func TestBoxSequentialEqualsParallel(t *testing.T) {
	const sw, sh, tw, th = 1920, 1080, 1280, 720
	src := make([]byte, sw*sh*4)
	for i := range src {
		src[i] = byte(i * 7 % 251)
	}
	seq := make([]byte, tw*th*4)
	ResizeGenericInto(seq, src, sw, sh, sw*4, tw, th, Box, "RGBA", SequentialResizeStrategy{}, "BGRA")
	par := make([]byte, tw*th*4)
	ResizeGenericInto(par, src, sw, sh, sw*4, tw, th, Box, "RGBA", ParallelResizeStrategy{}, "BGRA")
	if !bytes.Equal(seq, par) {
		t.Fatal("parallel box output differs from sequential")
	}
}
