package resize

import "testing"

func TestBilinearInterpExactPixel(t *testing.T) {
	cases := []struct {
		name string
		c00  uint8
	}{
		{"zero", 0x00},
		{"mid", 0x7F},
		{"max", 0xFF},
	}
	for _, c := range cases {
		if got := bilinearInterp(c.c00, 0, 0, 0, 0, 0); got != c.c00 {
			t.Errorf("%s: bilinearInterp(fx=0, fy=0) = %d, want %d", c.name, got, c.c00)
		}
	}
}
