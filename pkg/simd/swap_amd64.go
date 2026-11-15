//go:build amd64

package simd

var maskSSE = [16]byte{2, 1, 0, 3, 6, 5, 4, 7, 10, 9, 8, 11, 14, 13, 12, 15}

var maskAVX2Lo = [32]byte{
	2, 1, 0, 3, 6, 5, 4, 7, 10, 9, 8, 11, 14, 13, 12, 15,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
}

var maskAVX2Hi = [32]byte{
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
	18, 17, 16, 19, 22, 21, 20, 23, 26, 25, 24, 27, 30, 29, 28, 31,
}

// SwapAVX2 processes floor(len(src)/32)*32 bytes with VPOR-combined VPSHUFB
// masks (8 pixels per instruction). The tail is handled by the caller.
func SwapAVX2(src []byte, dst []byte)

// SwapSSE processes floor(len(src)/16)*16 bytes with SSSE3 PSHUFB.
// The tail is handled by the caller.
func SwapSSE(src []byte, dst []byte)
